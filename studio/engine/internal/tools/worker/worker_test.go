package worker

import (
	"errors"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/nativetest"
)

func TestFrameEncoding(t *testing.T) {
	message := pyjson.NewObject().Set("version", int64(1)).Set("op", "x").Set("text", "\xc3\xa9<")
	frame, err := encodeFrame(message)
	if err != nil || string(frame[4:]) != `{"version":1,"op":"x","text":"\`+`u00e9<"}` || frame[0] != byte(len(frame)-4) {
		t.Fatalf("frame %q %v", frame, err)
	}
	_, err = encodeFrame(pyjson.NewObject().Set("v", []any{math.NaN()}))
	var refused *apperr.Error
	if !errors.As(err, &refused) || refused.Kind != apperr.Value {
		t.Fatalf("NaN must be refused like allow_nan=False: %v", err)
	}
}

func TestControlHelloAndUnknownOperation(t *testing.T) {
	program := nativetest.Program(t, "studio_control.exe")
	var mu sync.Mutex
	var events []string
	process, err := Spawn(program, func(event *pyjson.Object) {
		kind, _ := event.Get("event")
		mu.Lock()
		events = append(events, kind.(string))
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	hello, err := process.Request("hello", DefaultTimeout, nil)
	if err != nil {
		t.Fatal(err)
	}
	capabilities, _ := hello.(*pyjson.Object).Get("capabilities")
	encoded, _ := pyjson.Dumps(capabilities)
	if !strings.Contains(string(encoded), `"control_operate"`) {
		t.Fatalf("capabilities %s", encoded)
	}
	_, err = process.Request("nonsense", DefaultTimeout, nil)
	var refused *apperr.Error
	if !errors.As(err, &refused) || refused.Kind != apperr.Runtime || !strings.Contains(err.Error(), "nonsense") {
		t.Fatalf("unknown operation: %v", err)
	}
	process.Close()
	if process.Alive() {
		t.Fatal("still alive after Close")
	}
	_, err = process.Request("hello", DefaultTimeout, nil)
	if err == nil || !strings.HasPrefix(err.Error(), "Native worker exited: ") {
		t.Fatalf("request after close: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	for _, kind := range events {
		if kind == "worker_failed" {
			t.Fatal("worker_failed must not be delivered while closing")
		}
	}
}

func TestTimeoutIsRuntimeForHTTPButDistinct(t *testing.T) {
	err := error(&Timeout{Op: "probe"})
	var refused *apperr.Error
	var timeout *Timeout
	if !errors.As(err, &refused) || refused.Kind != apperr.Runtime || !errors.As(err, &timeout) ||
		err.Error() != "Native probe request timed out" {
		t.Fatal(err)
	}
	_ = time.Second
}
