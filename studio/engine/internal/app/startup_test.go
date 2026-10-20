package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/recovery"
	"studio-engine/internal/testutil"
)

// TestStartupGateHoldsUntilRecovery: engineering requests wait while recovery runs and are
// answered once it finished; /api/v1/preferences is never held; a recovery that stopped to
// protect the autosave answers 503 startup_recovery_failed with its text.
func TestStartupGateHoldsUntilRecovery(t *testing.T) {
	release := make(chan struct{})
	held := recovery.StartBackground(func() (*pyjson.Object, error) {
		<-release
		return nil, nil
	}, nil)
	schemas, _ := filepath.Abs(filepath.Join("..", "..", "..", "schemas"))
	engine, err := Compose(Options{DataDir: testutil.TempDir(t), NativeDir: testutil.TempDir(t),
		LicenseHome: testutil.TempDir(t), SchemasDir: schemas, Token: "token", SecretKey: "secret",
		Version: "0.0.0+dev", NoBackground: true, Logf: t.Logf, Startup: startupGate{held}})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	answered := make(chan int, 1)
	go func() {
		status, _ := call(t, engine, "GET", "/api/v1/project", nil)
		answered <- status
	}()
	if status, _ := call(t, engine, "GET", "/api/v1/preferences", nil); status != http.StatusOK {
		t.Fatalf("preferences must not wait for recovery: %d", status)
	}
	select {
	case status := <-answered:
		t.Fatalf("GET /project answered %d while recovery was still running", status)
	case <-time.After(300 * time.Millisecond):
	}
	close(release)
	select {
	case status := <-answered:
		if status != http.StatusOK {
			t.Fatalf("GET /project after recovery: %d", status)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GET /project still waiting after recovery finished")
	}

	broken := testutil.TempDir(t)
	if err := os.Mkdir(filepath.Join(broken, "autosave.subsim"), 0o755); err != nil {
		t.Fatal(err)
	}
	failed := composeAt(t, broken, testutil.TempDir(t))
	defer failed.Close()
	status, body := call(t, failed, "GET", "/api/v1/project", nil)
	text := pyjsonText(body)
	if status != http.StatusServiceUnavailable || field(body, "code") != "startup_recovery_failed" ||
		!strings.Contains(text, "Autosave is not a regular local file") {
		t.Fatalf("failed recovery: %d %s", status, text)
	}
}
