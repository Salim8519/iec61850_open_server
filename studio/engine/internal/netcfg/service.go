package netcfg

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Projects is what the network workflow needs from the open project (projects service).
type Projects interface {
	// SimulatedIEDs lists the IEDs with role "simulated" (port defaults to 102).
	SimulatedIEDs() ([]PlanIED, error)
	// EnterLabMode stores lab_ip = row.Proposed per row and switches the network to lab mode
	// on the adapter (update_network {"mode": "lab", "adapter_guid", "adapter_label"}).
	EnterLabMode(rows []Row, adapter Adapter) error
	// LeaveLabMode switches the network back to loopback when it is in lab mode.
	LeaveLabMode() error
}

type storedPreview struct {
	guid     string
	strategy string
	manual   any
}

// NetworkService is service.NetworkService: preview, apply through the elevated helper, and
// clean up. Every Windows access goes through the injected interfaces.
type NetworkService struct {
	Projects   Projects
	Directory  string // data folder
	Program    string // str(Path(native IED worker))
	Executable string // the app exe started with --network-helper
	Journal    *Journal
	Source     AdapterSource
	Rules      func() ([]string, error)                                              // MMS rule names present
	Elevator   Elevator                                                              // UAC launch of the helper
	Probe      Prober                                                                // ARP reachability
	Firewall   func(unblock []string) (gateway, disabled map[string]bool, err error) // gateway_state
	Clock      func() time.Time                                                      // monotonic clock for the cache
	NewID      func() string                                                         // uuid4().hex for request files
	SNMP       SNMPProjects                                                          // nil: no SNMP devices (Python)
	SNMPRules  func() ([]string, error)                                              // SNMP rule names present
	Progress   PreviewProgress                                                       // live state of the address check

	mu       sync.Mutex
	stamp    time.Time
	cached   []Adapter
	previews map[string]storedPreview
}

// NewNetworkService wires the real Windows readers; tests replace the fields with fakes.
func NewNetworkService(projects Projects, directory, program, executable string) *NetworkService {
	shell := SystemExecutor{}
	return &NetworkService{Projects: projects, Directory: directory, Program: program, Executable: executable,
		Journal: NewJournal(directory), Source: SystemAdapters{Shell: shell},
		Rules: func() ([]string, error) {
			return RuleNames(studioRulePrefix, func() ([]string, error) { return ListStudioRules(shell) })
		},
		SNMPRules: func() ([]string, error) {
			return RuleNames(SNMPRulePrefix, func() ([]string, error) { return ListSNMPRules(shell) })
		},
		Elevator: UACElevator{}, Probe: ARPProber{},
		Firewall: func(names []string) (map[string]bool, map[string]bool, error) { return GatewayState(names, shell) },
		Clock:    time.Now, NewID: randomHex}
}

func randomHex() string {
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	raw[6], raw[8] = raw[6]&0x0F|0x40, raw[8]&0x3F|0x80 // uuid4 version and variant
	return hex.EncodeToString(raw[:])
}

func (s *NetworkService) clearCache() {
	s.mu.Lock()
	s.cached = nil
	s.mu.Unlock()
}

// Adapters is service.adapters: the list, cached for 10 seconds unless refresh is set.
func (s *NetworkService) Adapters(refresh bool) ([]Adapter, error) {
	s.mu.Lock()
	cached, stamp := s.cached, s.stamp
	s.mu.Unlock()
	now := s.Clock()
	if !refresh && cached != nil && now.Sub(stamp) <= 10*time.Second {
		return cached, nil
	}
	value, err := s.Source.Adapters()
	if err != nil {
		return nil, err
	}
	if value == nil {
		value = []Adapter{}
	}
	s.mu.Lock()
	s.cached, s.stamp = value, s.Clock()
	s.mu.Unlock()
	return value, nil
}

// Adapter is service.adapter: the adapter with this GUID or a plain error.
func (s *NetworkService) Adapter(guid any) (Adapter, error) {
	adapters, err := s.Adapters(false)
	if err != nil {
		return Adapter{}, err
	}
	wanted := strings.ToUpper(pyStr(guid))
	for _, adapter := range adapters {
		if adapter.GUID == wanted {
			return adapter, nil
		}
	}
	return Adapter{}, apperr.New("That network adapter is no longer available. Refresh the adapter list.")
}

// withDescriptions is [{**item, "description": label(item)} for item in items].
func withDescriptions(items []*pyjson.Object, label func(*pyjson.Object) (string, error)) ([]*pyjson.Object, error) {
	described := make([]*pyjson.Object, len(items))
	for index, item := range items {
		text, err := label(item)
		if err != nil {
			return nil, err
		}
		described[index] = copyObject(item).Set("description", text)
	}
	return described, nil
}

func copyObject(item *pyjson.Object) *pyjson.Object {
	copied := pyjson.NewObject()
	for _, key := range item.Keys() {
		value, _ := item.Get(key)
		copied.Set(key, value)
	}
	return copied
}

// isTrue is "value is True".
func isTrue(value any) bool {
	flag, ok := value.(bool)
	return ok && flag
}
