package licensing

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/licensing/tpm"
)

// These tests run the Go license service against the REAL license server (license-server/,
// built from this repository or the binary STUDIO_LICENSE_SERVER_BIN). They are skipped while
// the server predates ADR-0098 (no POST /api/v1/activate/tpm): the orchestrator runs them
// after merging the server half.

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
	server := &realServer{process: process, url: url, binary: binary, env: env, keys: map[string][]byte{KeyID(public): public}}
	response, err := http.Post(url+"/api/v1/activate/tpm", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusMethodNotAllowed {
		t.Skip("this license server has no /api/v1/activate/tpm yet (ADR-0098 server half not merged)")
	}
	return server
}

func (s *realServer) issueKey(t *testing.T) string {
	command := exec.Command(s.binary, "issue", "--email", "buyer@example.com")
	command.Env = s.env
	var out bytes.Buffer
	command.Stdout = &out
	if err := command.Run(); err != nil {
		t.Fatalf("issue: %v", err)
	}
	var issued struct{ License struct{ Key string } }
	if err := json.Unmarshal(out.Bytes(), &issued); err != nil || issued.License.Key == "" {
		t.Fatalf("issue answer %q: %v", out.String(), err)
	}
	return issued.License.Key
}

// computer is a Go license service pointed at the real server.
func (s *realServer) computer(t *testing.T, chip tpm.Chip, hwid string) *Service {
	t.Helper()
	t.Setenv("STUDIO_LICENSE_CERT_PATH", "")
	home := scenarioDir(t)
	service := New(Options{Home: home, DataDir: home, Enforce: true, Keys: s.keys, AllowTestServer: true,
		HWID: func() (string, error) { return hwid, nil }, Debugger: func() bool { return false },
		Logf: t.Logf, Registry: memoryRegistry{}, Chip: chip})
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

func TestRealServerLeaseTokenRotationAndAClonedCopy(t *testing.T) {
	server := startRealServer(t)
	key := server.issueKey(t)
	hwid := strings.Repeat("e5", 32)
	original := server.computer(t, tpm.None{}, hwid)
	if _, err := original.Activate(key); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if got := status(t, original); !got.Licensed || got.Binding != BindingLease {
		t.Fatalf("after activation: %+v", got)
	}
	first := original.lease.CurrentToken()
	if first == "" {
		t.Fatal("the activation answer carried no lease token")
	}
	// A copy of the license files on a second computer (same Machine ID, e.g. a cloned disk).
	clone := server.computer(t, tpm.None{}, hwid)
	for _, name := range []string{"license.cert", "license_state.json", ".lease_token", ".lease_answer"} {
		if data, err := os.ReadFile(filepath.Join(original.options.Home, name)); err == nil {
			os.WriteFile(filepath.Join(clone.options.Home, name), data, 0o666)
		}
	}
	if _, err := original.CheckOnlineNow(); err != nil {
		t.Fatalf("check online (original): %v", err)
	}
	if second := original.lease.CurrentToken(); second == "" || second == first {
		t.Fatalf("the token did not rotate: %q -> %q", first, second)
	}
	if _, err := clone.CheckOnlineNow(); err != nil {
		t.Fatalf("check online (copy): %v", err)
	}
	if got := status(t, clone); got.Licensed || !clone.revoked.Contains(key) {
		t.Fatalf("the copy kept its license: %+v", got)
	}
	if got := status(t, original); !got.Licensed {
		t.Fatalf("the original lost its license: %+v", got)
	}
}

func TestRealServerWithThisComputersRealChip(t *testing.T) {
	if os.Getenv("STUDIO_REAL_TPM") != "1" {
		t.Skip("set STUDIO_REAL_TPM=1 to activate with this computer's real security chip")
	}
	server := startRealServer(t)
	key := server.issueKey(t)
	computer := server.computer(t, tpm.System(), strings.Repeat("e6", 32))
	if _, err := computer.Activate(key); err != nil {
		t.Fatalf("activate: %v", err)
	}
	got := status(t, computer)
	if !got.Licensed || (got.Binding != BindingTPM && got.Binding != BindingLease) {
		t.Fatalf("status %+v", got)
	}
	t.Logf("this computer's license is tied by: %s", got.Binding)
	if got.Binding == BindingTPM {
		server.stop(t) // a chip license never needs the server again
		if again := status(t, computer); !again.Licensed {
			t.Fatalf("the chip license needed the server: %+v", again)
		}
	}
}

// stop ends the server (a second Kill from the test cleanup is harmless).
func (s *realServer) stop(t *testing.T) {
	t.Helper()
	s.process.Process.Kill()
	s.process.Wait()
}
