package native

import (
	"errors"
	"fmt"
	"testing"

	"studio-engine/internal/apperr"
)

func TestTimeoutErrorIsRuntimeAndTimeout(t *testing.T) {
	var err error = newTimeout("read")
	if err.Error() != "Native read request timed out" {
		t.Fatalf("text %q", err.Error())
	}
	var app *apperr.Error
	if !errors.As(err, &app) || app.Kind != apperr.Runtime || app.Message != err.Error() {
		t.Fatalf("not an apperr Runtime error: %#v", app)
	}
	timeout, ok := err.(interface{ Timeout() bool })
	if !ok || !timeout.Timeout() || !IsTimeout(err) {
		t.Fatal("Timeout() not reported")
	}
	if !IsTimeout(fmt.Errorf("wrapped: %w", err)) {
		t.Fatal("wrapped timeout not recognised")
	}
	if IsTimeout(apperr.RuntimeError("Native worker exited: 1")) || IsTimeout(nil) {
		t.Fatal("non-timeouts reported as timeouts")
	}
}
