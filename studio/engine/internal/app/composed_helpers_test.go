package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/events"
	"studio-engine/internal/native"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
)

// muxNative returns STUDIO_NATIVE_BIN when its studio_control has control_mux_v1 (the
// composed tests never run workers from the owner's build folder by default).
func muxNative(t testing.TB) string {
	t.Helper()
	folder := os.Getenv("STUDIO_NATIVE_BIN")
	if folder == "" {
		t.Skip("set STUDIO_NATIVE_BIN to a copy of the native workers to run the composed click path")
	}
	folder, _ = filepath.Abs(folder)
	process, err := native.StartControl(filepath.Join(folder, "studio_control.exe"))
	if err != nil {
		t.Skipf("studio_control unavailable in %s: %v", folder, err)
	}
	defer process.Close()
	hello, err := process.Hello()
	if err != nil || !native.HasCapabilities(hello, []string{native.MuxCapability}) {
		t.Skipf("studio_control in %s lacks %s", folder, native.MuxCapability)
	}
	return folder
}

// composeAt builds the engine on dataDir with the given native folder.
func composeAt(t testing.TB, dataDir, nativeDir string) *Engine {
	t.Helper()
	schemas, _ := filepath.Abs(filepath.Join("..", "..", "..", "schemas"))
	// studio_xsd.exe is not built yet: Python's verdicts for the template and corpus files.
	seed, _ := filepath.Abs(filepath.Join("..", "scl", "testdata", "golden", "xsd_seed.json"))
	engine, err := Compose(Options{DataDir: dataDir, NativeDir: nativeDir, LicenseHome: testutil.TempDir(t),
		SchemasDir: schemas, XSDSeed: []string{seed}, Token: "token", SecretKey: "secret", Version: "0.0.0+dev",
		NoBackground: true, Logf: t.Logf})
	if err != nil {
		t.Fatal(err)
	}
	if !engine.Recovery.Wait(30 * time.Second) {
		t.Fatal("startup recovery did not finish")
	}
	return engine
}

// call sends one authenticated loopback request through the composed handler and decodes
// the JSON answer (nil for a non-JSON body).
func call(t testing.TB, engine *Engine, method, path string, body any) (int, any) {
	t.Helper()
	var payload []byte
	if body != nil {
		encoded, err := pyjson.Flask(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = encoded
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.RemoteAddr, request.Host = "127.0.0.1:50000", "127.0.0.1:8765"
	request.Header.Set("Authorization", "Bearer token")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	engine.Handler.ServeHTTP(recorder, request)
	decoded, err := pyjson.Decode(recorder.Body.Bytes())
	if err != nil {
		decoded = nil
	}
	return recorder.Code, decoded
}

// mustCall is call that fails the test on any status but 200.
func mustCall(t testing.TB, engine *Engine, method, path string, body any) any {
	t.Helper()
	status, decoded := call(t, engine, method, path, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", method, path, status, pyjsonText(decoded))
	}
	return decoded
}

func pyjsonText(value any) string {
	encoded, err := pyjson.Dumps(value)
	if err != nil {
		return "?"
	}
	return string(encoded)
}

func obj(pairs ...any) *pyjson.Object {
	result := pyjson.NewObject()
	for index := 0; index < len(pairs); index += 2 {
		result.Set(pairs[index].(string), pairs[index+1])
	}
	return result
}

func field(value any, keys ...string) any {
	for _, key := range keys {
		object, ok := value.(*pyjson.Object)
		if !ok {
			return nil
		}
		value, _ = object.Get(key)
	}
	return value
}

// waitEvent waits for the first event after cursor that matches, returning it and its
// arrival time.
func waitEvent(bus *events.Bus, cursor int64, timeout time.Duration, match func(*events.Event) bool) (*events.Event, time.Time) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, event := range bus.After(cursor, 50*time.Millisecond, nil) {
			cursor = event.ID
			if match(event) {
				return event, time.Now()
			}
		}
	}
	return nil, time.Time{}
}
