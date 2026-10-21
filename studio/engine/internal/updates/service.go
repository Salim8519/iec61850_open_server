package updates

import (
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"studio-engine/internal/licensing"
	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// Constants of updates/service.py (ADR-0071).
const (
	ReleaseDomain  = "scadabench-release-v1\n"
	Day            = 24 * 3600
	TimeoutSeconds = 6
	ManualGap      = 60 // "Check now" pressed again within a minute reuses the last answer
	StartDelayMin  = 20 // seconds after start: never slows the launch, spreads server load
	StartDelayMax  = 120
	MaxBody        = 16 << 10
)

// Opener sends the one GET of the update check; *http.Client satisfies it.
type Opener interface {
	Do(request *http.Request) (*http.Response, error)
}

// Options are UpdateService's constructor arguments.
type Options struct {
	Directory string                 // the data folder holding update_check.json
	ServerURL func() (string, error) // the license server (licensing.Service.ServerURL)
	Current   string                 // this app's version, e.g. "0.1.14+dev"
	Keys      map[string][]byte      // nil means licensing.PublicKeys()
	Opener    Opener                 // nil means an http.Client with the 6 s timeout
	Clock     func() float64         // nil means time.time()
	Logf      func(string, ...any)   // background failures; nil means discard
}

// Service is updates.UpdateService: a quiet daily check for a newer ScadaBench that never
// downloads or installs anything.
type Service struct {
	options  Options
	path     string
	mu       sync.Mutex // serialises read-modify-write of update_check.json, like Python's lock
	checking atomic.Bool
}

// New returns the update service.
func New(options Options) *Service {
	if options.Keys == nil {
		options.Keys = licensing.PublicKeys()
	}
	if options.Opener == nil {
		options.Opener = &http.Client{Timeout: TimeoutSeconds * time.Second}
	}
	if options.Clock == nil {
		options.Clock = func() float64 { return float64(time.Now().UnixNano()) / 1e9 }
	}
	if options.Logf == nil {
		options.Logf = func(string, ...any) {}
	}
	return &Service{options: options, path: filepath.Join(options.Directory, "update_check.json")}
}

// read is UpdateService._read: the stored object, or {} when missing, unreadable or not
// an object.
func (s *Service) read() *pyjson.Object {
	value, _ := pytext.ReadJSON(s.path)
	if object, ok := value.(*pyjson.Object); ok {
		return object
	}
	return pyjson.NewObject()
}

// write is UpdateService._write: json.dumps(state, indent=2), text mode, then replace.
func (s *Service) write(state *pyjson.Object) error {
	text, err := pytext.DumpsIndent(state)
	if err != nil {
		return err
	}
	return pytext.WriteReplace(s.path, text)
}

func get(object *pyjson.Object, key string) any {
	value, _ := object.Get(key)
	return value
}

// Status is UpdateService.status() in Python's key order. An error stands for the
// KeyError/TypeError Python raises on a hand-edited file (HTTP 500).
func (s *Service) Status() (*pyjson.Object, error) {
	state := s.read()
	latest, _ := get(state, "release").(*pyjson.Object)
	available, important, dismissed := false, false, false
	if latest != nil && latest.Len() > 0 {
		version, found := latest.Get("version")
		if !found {
			return nil, &licensing.UncaughtError{Reason: "'version'"}
		}
		var err error
		if available, err = IsNewer(version, s.options.Current); err != nil {
			return nil, err
		}
		if minimum := get(latest, "min_version"); available && licensing.Truthy(minimum) {
			if important, err = IsNewer(minimum, s.options.Current); err != nil {
				return nil, err
			}
		}
		text, isText := get(state, "dismissed").(string)
		dismissed = available && isText && text == version && !important
	}
	auto, found := state.Get("auto")
	autoOn := !found || auto != any(false)
	var shown any
	if available {
		shown = latest
	}
	return pyjson.NewObject().Set("current", s.options.Current).Set("auto", autoOn).
		Set("checked_at", get(state, "checked_at")).Set("error", get(state, "error")).
		Set("checking", s.checking.Load()).Set("available", available).Set("important", important).
		Set("dismissed", dismissed).Set("latest", shown), nil
}

// ConfigError is the ValueError of configure(): the API answers 400 with
// {"error": Message, "code": "invalid"}.
type ConfigError struct{ Message string }

func (e *ConfigError) Error() string { return e.Message }

// Body is the Flask body object for the 400 answer.
func (e *ConfigError) Body() *pyjson.Object {
	return pyjson.NewObject().Set("error", e.Message).Set("code", "invalid")
}

// Configure is UpdateService.configure: only "auto" (bool) and "dismissed" (a version or
// null) can be changed; changes is the request's decoded JSON body (nil when unreadable).
func (s *Service) Configure(changes any) (*pyjson.Object, error) {
	object, isObject := changes.(*pyjson.Object)
	if isObject {
		for _, key := range object.Keys() {
			if key != "auto" && key != "dismissed" {
				isObject = false
			}
		}
	}
	if !isObject {
		return nil, &ConfigError{"Only auto and dismissed can be changed"}
	}
	s.mu.Lock()
	err := s.applyChanges(object)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return s.Status()
}

func (s *Service) applyChanges(changes *pyjson.Object) error {
	state := s.read()
	if auto, found := changes.Get("auto"); found {
		if _, isBool := auto.(bool); !isBool {
			return &ConfigError{"auto must be true or false"}
		}
		state.Set("auto", auto)
	}
	if dismissed, found := changes.Get("dismissed"); found {
		if dismissed != nil {
			_, ok, err := Parse(dismissed)
			if err != nil {
				return err
			}
			if !ok {
				return &ConfigError{"dismissed must be a version"}
			}
		}
		state.Set("dismissed", dismissed)
	}
	return s.write(state)
}
