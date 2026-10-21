package licensing

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// Timing constants of licensing/service.py.
const (
	StatusCacheSeconds = 60
	VerifyEverySeconds = 7 * 86400
)

// Options are LicenseService's constructor arguments; every seam the parity harness needs
// is injectable. Zero values mean Python's defaults.
type Options struct {
	Home    string // license home, normally DefaultHome()
	DataDir string // the app data folder (logs/runs.jsonl feeds the time anchor)
	Enforce bool   // packaged builds enforce; development never does
	// HWID and LegacyHWID read the machine IDs. Both nil means this PC (MachineID and
	// LegacyMachineID); HWID set and LegacyHWID nil means no legacy ID, as in Python.
	HWID          func() (string, error)
	LegacyHWID    func() (string, error)
	Clock         func() float64       // nil means time.time()
	Keys          map[string][]byte    // nil means PublicKeys(); an empty map means none
	ClientFactory ClientFactory        // nil means DefaultClientFactory
	NativeFolder  string               // "" means "."
	ExpectedFiles map[string]string    // integrity.EXPECTED (empty in development)
	Debugger      func() bool          // nil means DebuggerAttached
	Logf          func(string, ...any) // nil means log.Printf
	// Registry is the third place of the clock values: WindowsRegistry{} in the app, a fake
	// in tests, nil for none (Python's registry=None default).
	Registry Registry
	// Parts returns the trial part hashes; nil means this PC (MachineTrialParts).
	Parts func() (*pyjson.Object, error)
}

// Service owns this computer's license state (service.LicenseService).
type Service struct {
	options    Options
	certFile   string
	configFile string
	stateFile  string
	anchor     TimeAnchor
	trialClock TrialClock

	mu         sync.Mutex // cache, machine IDs and bound ID
	filesMu    sync.Mutex // serialises read-modify-write of the home files, like Python's RLock
	cached     *Status
	cachedAt   float64
	hwid       string
	hwidDone   bool
	legacyDone bool
	legacy     string
	boundHWID  string
}

// DefaultHome is %LOCALAPPDATA%\ScadaBench (or ~/AppData/Local/ScadaBench).
func DefaultHome() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(base, "ScadaBench")
}

// New returns the service. Like Python, an enforcing service exports
// STUDIO_LICENSE_CERT_PATH so native workers check the same certificate.
func New(options Options) *Service {
	if options.Clock == nil {
		options.Clock = func() float64 { return float64(time.Now().UnixNano()) / 1e9 }
	}
	if options.ClientFactory == nil {
		options.ClientFactory = DefaultClientFactory
	}
	if options.Debugger == nil {
		options.Debugger = DebuggerAttached
	}
	if options.Logf == nil {
		options.Logf = log.Printf
	}
	if options.Parts == nil {
		options.Parts = MachineTrialParts
	}
	if options.HWID == nil && options.LegacyHWID == nil {
		options.HWID, options.LegacyHWID = MachineID, LegacyMachineID
	}
	s := &Service{options: options, certFile: filepath.Join(options.Home, "license.cert"),
		configFile: filepath.Join(options.Home, "client_config.json"),
		stateFile:  filepath.Join(options.Home, "license_state.json"),
		anchor: TimeAnchor{Value: NewClockValue("time_anchor", options.Home, options.Registry),
			RunsFile: filepath.Join(options.DataDir, "logs", "runs.jsonl")},
		trialClock: NewTrialClock(options.Home, options.Registry, options.Keys)}
	if options.Enforce {
		os.Setenv("STUDIO_LICENSE_CERT_PATH", s.certFile)
	}
	return s
}

// CertFile is the license.cert path.
func (s *Service) CertFile() string { return s.certFile }

// readObject is _read_json(path, {}) for callers that then use dict methods: a file that
// is valid JSON but not an object makes Python fail with an uncaught AttributeError.
func readObject(path string) (*pyjson.Object, error) {
	value, ok := pytext.ReadJSON(path)
	if !ok {
		return pyjson.NewObject(), nil
	}
	object, isObject := value.(*pyjson.Object)
	if !isObject {
		return nil, uncaught("'" + filepath.Base(path) + "' does not hold a JSON object")
	}
	return object, nil
}

// HWID returns this PC's machine ID (cached after the first success).
func (s *Service) HWID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hwidDone {
		id, err := s.options.HWID()
		if err != nil {
			return "", err
		}
		s.hwid, s.hwidDone = id, true
	}
	return s.hwid, nil
}

// LegacyHWID returns the pre-v2 machine ID, or "" when it cannot be read or is not wanted.
func (s *Service) LegacyHWID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.legacyDone {
		s.legacyDone = true
		if s.options.LegacyHWID != nil {
			if id, err := s.options.LegacyHWID(); err == nil {
				s.legacy = id
			}
		}
	}
	return s.legacy
}

// machineFor is _machine_for: the machine ID this certificate must match.
func (s *Service) machineFor(payload *pyjson.Object) (string, error) {
	current, err := s.HWID()
	if err != nil {
		return "", err
	}
	certificateHWID, _ := payload.Get("hwid")
	issuedAt, found := payload.Get("issued_at")
	if !found {
		issuedAt = int64(0)
	}
	bound, err := BoundHWID(certificateHWID, issuedAt, current, s.LegacyHWID())
	if err == errIntOverflow {
		return "", uncaught("cannot convert float infinity to integer")
	}
	return bound, nil
}

// ServerURL is the configured license server, or DefaultServerURL, without a trailing "/".
func (s *Service) ServerURL() (string, error) {
	config, err := readObject(s.configFile)
	if err != nil {
		return "", err
	}
	configured, _ := config.Get("server_url")
	if !truthy(configured) {
		return DefaultServerURL, nil
	}
	text, isText := configured.(string)
	if !isText {
		return "", uncaught("server_url is not text")
	}
	return strings.TrimRight(text, "/"), nil
}

func (s *Service) now() int64 { return int64(s.options.Clock()) }
