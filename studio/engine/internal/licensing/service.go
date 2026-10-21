package licensing

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/licensing/tpm"
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
	// HWID reads the machine ID (v3, ADR-0098); nil means this PC (MachineID).
	HWID          func() (string, error)
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
	// HostKey is the VM host key (ADR-0099): the DPAPI protector of the host key and the search a
	// virtual machine makes for its host. The zero value switches the feature off.
	HostKey HostKeyOptions
	// AllowTestServer is the explicit test-only seam: tests and the sandbox may talk to a
	// server on this PC. The release composition never sets it, so an enforcing build only
	// ever asks its built-in server and ignores client_config.json (ADR-0097).
	AllowTestServer bool
	// Nonce returns the 32 hex characters sent with every verify call; nil means RandomNonce.
	Nonce func() string
	// Chip is the computer's security chip (ADR-0098). The app composition passes
	// tpm.System(); nil means no chip, so every license is a lease and tests never touch
	// hardware.
	Chip tpm.Chip
}

// Service owns this computer's license state (service.LicenseService).
type Service struct {
	options    Options
	certFile   string
	configFile string
	stateFile  string
	akFile     string // the stored attestation key of a chip-bound license
	anchor     TimeAnchor
	trialClock TrialClock
	vm         vmHostState // the proof of a licensed host (vm_host.go)
	revoked    RevokedIDs
	lease      LeaseStore
	chip       chipState

	mu       sync.Mutex // cache and machine ID
	filesMu  sync.Mutex // serialises read-modify-write of the home files, like Python's RLock
	leaseMu  sync.Mutex // one lease check at a time: a check rotates the token
	cached   *Status
	cachedAt float64
	hwid     string
	hwidDone bool
	// leaseValidFrom is when a lease license was last confirmed (signed): the later of its
	// certificate's issue time and its newest signed verify answer.
	leaseValidFrom int64
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
		if options.AllowTestServer {
			options.ClientFactory = TestClientFactory
		}
	}
	if options.Nonce == nil {
		options.Nonce = RandomNonce
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
	if options.HWID == nil {
		options.HWID = MachineID
	}
	if options.Chip == nil {
		options.Chip = tpm.None{}
	}
	s := &Service{options: options, certFile: filepath.Join(options.Home, "license.cert"),
		configFile: filepath.Join(options.Home, "client_config.json"),
		stateFile:  filepath.Join(options.Home, "license_state.json"),
		akFile:     filepath.Join(options.Home, "ak.blob"),
		anchor: TimeAnchor{Value: NewClockValue("time_anchor", options.Home, options.Registry),
			RunsFile: filepath.Join(options.DataDir, "logs", "runs.jsonl")},
		trialClock: NewTrialClock(options.Home, options.Registry, options.Keys),
		revoked:    NewRevokedIDs(options.Home, options.Registry),
		lease:      NewLeaseStore(options.Home, options.Registry, options.Keys)}
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

// ServerURL is the built-in license server in an enforcing build; client_config.json is
// read only in development and for the test-only seam. It never ends with "/".
func (s *Service) ServerURL() (string, error) {
	if s.options.Enforce && !s.options.AllowTestServer {
		return DefaultServerURL, nil
	}
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
