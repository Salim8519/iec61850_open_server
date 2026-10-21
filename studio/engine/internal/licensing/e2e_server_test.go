package licensing

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests run the Go license service against the REAL license server (license-server/,
// built from this repository or the binary STUDIO_LICENSE_SERVER_BIN): activate once, the
// same Machine ID can be issued again (a Windows reinstall), any other Machine ID is refused,
// the weekly check is a signed answer and never stops an offline computer (ADR-0097, ADR-0098).

type realServer struct {
	process     *exec.Cmd
	url, binary string
	env         []string
	keys        map[string][]byte
}

func freePort(t *testing.T) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func buildServerBinary(t *testing.T) string {
	if prebuilt := os.Getenv("STUDIO_LICENSE_SERVER_BIN"); prebuilt != "" {
		return prebuilt
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("Go toolchain not installed")
	}
	binary := filepath.Join(t.TempDir(), "license-server.exe")
	build := exec.Command("go", "build", "-o", binary, "./cmd/license-server")
	build.Dir = filepath.Join("..", "..", "..", "..", "license-server")
	if output, err := build.CombinedOutput(); err != nil {
		t.Skipf("the license server does not build here: %v\n%s", err, output)
	}
	return binary
}

func startRealServer(t *testing.T) *realServer {
	t.Helper()
	binary := buildServerBinary(t)
	port := freePort(t)
	env := append(os.Environ(), "LICENSE_DATA_DIR="+filepath.Join(t.TempDir(), "data"),
		fmt.Sprintf("LICENSE_LISTEN_ADDR=127.0.0.1:%d", port), "LICENSE_POW_DIFFICULTY=10",
		"LICENSE_RATE_SENSITIVE=1000", "LICENSE_RATE_GENERAL=10000")
	keygen := exec.Command(binary, "keygen")
	keygen.Env = env
	created, err := keygen.Output()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	fields := strings.Fields(string(created))
	public, err := hex.DecodeString(fields[len(fields)-1])
	if err != nil || len(public) != 32 {
		t.Fatalf("public key %q: %v", fields[len(fields)-1], err)
	}
	process := exec.Command(binary, "serve")
	process.Env = env
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { process.Process.Kill(); process.Wait() })
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		if response, err := http.Get(url + "/api/v1/health"); err == nil {
			response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the license server did not start")
		}
	}
	return &realServer{process: process, url: url, binary: binary, env: env, keys: map[string][]byte{KeyID(public): public}}
}

func (s *realServer) admin(t *testing.T, args ...string) []byte {
	t.Helper()
	command := exec.Command(s.binary, args...)
	command.Env = s.env
	var out bytes.Buffer
	command.Stdout = &out
	if err := command.Run(); err != nil {
		t.Fatalf("%s: %v", args[0], err)
	}
	return out.Bytes()
}

func (s *realServer) issueKey(t *testing.T) string {
	out := s.admin(t, "issue", "--email", "buyer@example.com")
	var issued struct{ License struct{ Key string } }
	if err := json.Unmarshal(out, &issued); err != nil || issued.License.Key == "" {
		t.Fatalf("issue answer %q: %v", out, err)
	}
	return issued.License.Key
}

// stop ends the server (a second Kill from the test cleanup is harmless).
func (s *realServer) stop() {
	s.process.Process.Kill()
	s.process.Wait()
}

// computer is a Go license service pointed at the real server; its clock is the real time
// plus *offset seconds, so a test can let weeks pass.
func (s *realServer) computer(t *testing.T, hwid string, offset *float64) *Service {
	t.Helper()
	t.Setenv("STUDIO_LICENSE_CERT_PATH", "")
	home := scenarioDir(t)
	service := New(Options{Home: home, DataDir: home, Enforce: true, Keys: s.keys, AllowTestServer: true,
		HWID: func() (string, error) { return hwid, nil }, Debugger: func() bool { return false },
		Clock: func() float64 { return float64(time.Now().Unix()) + *offset },
		Logf:  t.Logf, Registry: memoryRegistry{}})
	if _, err := service.SetServerURL(s.url); err != nil {
		t.Fatal(err)
	}
	return service
}

func status(t *testing.T, service *Service) Status {
	t.Helper()
	current, err := service.Current(true)
	if err != nil {
		t.Fatal(err)
	}
	return current
}

func TestRealServerActivatesOnceAndKeepsTheSameIDForLife(t *testing.T) {
	server := startRealServer(t)
	key := server.issueKey(t)
	hwid, offset := strings.Repeat("e5", 32), 0.0
	first := server.computer(t, hwid, &offset)
	if _, err := first.Activate(key); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if got := status(t, first); !got.Licensed || got.Plan != "lifetime" {
		t.Fatalf("after activation: %+v", got)
	}
	// Windows is reinstalled: the license home is gone, the Machine ID v3 is the same.
	reinstalled := server.computer(t, hwid, &offset)
	if got := status(t, reinstalled); got.Licensed {
		t.Fatalf("a new license home must start without a license: %+v", got)
	}
	if _, err := reinstalled.Activate(key); err != nil {
		t.Fatalf("the same key on the same Machine ID is issued again: %v", err)
	}
	if got := status(t, reinstalled); !got.Licensed {
		t.Fatalf("after the reinstall: %+v", got)
	}
	// Any other Machine ID is refused for good: no moves, no resets.
	other := server.computer(t, strings.Repeat("a7", 32), &offset)
	_, err := other.Activate(key)
	var refused *ServerError
	if !errors.As(err, &refused) || refused.Code != any("bound_elsewhere") {
		t.Fatalf("another Machine ID got %v, want bound_elsewhere", err)
	}
	if got := status(t, other); got.Licensed {
		t.Fatalf("the refused computer is licensed: %+v", got)
	}
	// Activated once, works forever: no server, years later.
	server.stop()
	offset = 5 * 365 * 86400
	if got := status(t, first); !got.Licensed {
		t.Fatalf("the license needed the server: %+v", got)
	}
}

func TestRealServerWeeklyCheckIsSignedAndNeverBlocksAnOfflineComputer(t *testing.T) {
	server := startRealServer(t)
	key := server.issueKey(t)
	offset := 0.0
	computer := server.computer(t, strings.Repeat("e6", 32), &offset)
	if _, err := computer.Activate(key); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if verdict, err := computer.VerifyOnlineIfDue(false); err != nil || verdict != nil {
		t.Fatalf("a fresh license is not asked again: %v %v", verdict, err)
	}
	offset = 8 * 86400
	verdict, err := computer.VerifyOnlineIfDue(false)
	if err != nil || verdict != any("valid") {
		t.Fatalf("the weekly check: %v %v", verdict, err)
	}
	if got := status(t, computer); !got.Licensed {
		t.Fatalf("after a valid check: %+v", got)
	}
	server.stop() // the server is gone: the next weekly check is skipped quietly
	offset = 16 * 86400
	verdict, err = computer.VerifyOnlineIfDue(false)
	if err != nil || verdict != nil {
		t.Fatalf("an unreachable server must not matter: %v %v", verdict, err)
	}
	if got := status(t, computer); !got.Licensed {
		t.Fatalf("an offline computer lost its license: %+v", got)
	}
}

func TestRealServerCancelledKeyStopsAtTheNextWeeklyCheckAndStaysCancelled(t *testing.T) {
	server := startRealServer(t)
	key := server.issueKey(t)
	offset := 0.0
	computer := server.computer(t, strings.Repeat("e7", 32), &offset)
	if _, err := computer.Activate(key); err != nil {
		t.Fatalf("activate: %v", err)
	}
	server.admin(t, "revoke", "--key", key, "--reason", "refund")
	offset = 8 * 86400
	verdict, err := computer.VerifyOnlineIfDue(false)
	if err != nil || verdict != any("revoked") {
		t.Fatalf("the weekly check after a refund: %v %v", verdict, err)
	}
	if got := status(t, computer); got.Licensed || !computer.revoked.Contains(key) {
		t.Fatalf("the cancelled license still works: %+v", got)
	}
	if _, err := computer.Activate(key); err == nil {
		t.Fatal("a cancelled key activated again")
	}
	if got := status(t, computer); got.Licensed {
		t.Fatalf("cancelled stays cancelled: %+v", got)
	}
}
