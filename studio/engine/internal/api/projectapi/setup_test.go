package projectapi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"studio-engine/internal/testutil"
	"sync"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/store/settings"
)

// seedValidator answers with the XSD verdicts CPython recorded for the public corpus.
type seedValidator map[string]scl.SchemaVerdict

func (v seedValidator) Validate(source []byte, schema string) (scl.SchemaVerdict, error) {
	sum := sha256.Sum256(source)
	if verdict, ok := v[hex.EncodeToString(sum[:])+"/"+schema]; ok {
		return verdict, nil
	}
	return scl.SchemaVerdict{}, apperr.RuntimeError("no recorded XSD verdict")
}

func loadSeed(t *testing.T) seedValidator {
	data, err := os.ReadFile("../../scl/testdata/golden/xsd_seed.json")
	if err != nil {
		t.Skip("xsd seed missing")
	}
	decoded, _ := pyjson.Decode(data)
	entries := decoded.([]any)
	if golden, err := os.ReadFile("testdata/generated/http_golden.json"); err == nil {
		recorded, _ := pyjson.Decode(golden)
		more, _ := get(object(recorded), "xsd").([]any)
		entries = append(entries, more...)
	}
	seed := seedValidator{}
	for _, raw := range entries {
		item := raw.(*pyjson.Object)
		status, _ := get(item, "status").(string)
		verdict := scl.SchemaVerdict{Status: status}
		for _, e := range get(item, "errors").([]any) {
			line, _ := get(object(e), "line").(int64)
			message, _ := get(object(e), "message").(string)
			verdict.Errors = append(verdict.Errors, scl.SchemaError{Line: int(line), Message: message})
		}
		sum, _ := get(item, "sha256").(string)
		schema, _ := get(item, "schema").(string)
		seed[sum+"/"+schema] = verdict
	}
	return seed
}

// fakeRuntime is a stopped coordinator that records emitted events.
type fakeRuntime struct {
	lock   sync.Mutex
	events []string
}

func (f *fakeRuntime) State() string          { return "stopped" }
func (f *fakeRuntime) RunningIEDs() []string  { return nil }
func (f *fakeRuntime) CycleLock() sync.Locker { return &f.lock }
func (f *fakeRuntime) Emit(kind string, payload *pyjson.Object) {
	body, _ := pyjson.Dumps(payload)
	f.events = append(f.events, kind+" "+string(body))
}

type startupResult struct {
	result *pyjson.Object
	err    error
}

func (s startupResult) Result() (*pyjson.Object, error) { return s.result, s.err }

// counterIDs is the generator's deterministic uuid4().hex: 1, 2, 3... as 32 hex digits.
func counterIDs() func() string {
	var lock sync.Mutex
	count := 0
	return func() string {
		lock.Lock()
		defer lock.Unlock()
		count++
		return fmt.Sprintf("%032x", count)
	}
}

// newArea wires the routes like the app: one id counter for the service and importer,
// startup recovery run to completion before the first request.
func newArea(t *testing.T) (http.Handler, *fakeRuntime) {
	dir := testutil.TempDir(t)
	ids := counterIDs()
	projects, err := service.New(service.Config{DataDir: dir, NewID: ids,
		Importer: &scl.Importer{Validator: loadSeed(t), NewID: ids}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = projects.Close() })
	result, err := projects.RecoverStartup()
	runtime := &fakeRuntime{}
	area := New(Deps{Projects: projects, Runtime: runtime, Startup: startupResult{result, err},
		Preferences: settings.NewPreferences(dir)})
	mux := http.NewServeMux()
	for _, route := range area.Routes() {
		mux.Handle(route.Method+" "+route.Pattern, route.Handler)
	}
	return mux, runtime
}
