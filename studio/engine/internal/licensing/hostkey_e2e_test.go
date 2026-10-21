package licensing

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/licensing/hostkey"
)

// End to end on this PC: the REAL license server binary signs the certificate, the REAL DPAPI keeps
// the host key, the REAL host helper (scadabench_hostkey.exe, built with the test seam so it binds a
// loopback address) answers, a virtual machine (a second license home) becomes licensed in vm-host
// mode, and the native guard's own verifier accepts the proof the Go backend wrote. No VM, no
// firewall and no registry value is touched; the helper's only test input is a loopback address and
// the test server's public key.

// testSeed is the seed of the test release key (license_signer.py TEST_SEED), so the enforced native
// build, which embeds that key, trusts what the real server signs here.
var testSeed = bytes.Repeat([]byte{0x42}, 32)

type hostE2E struct {
	t        *testing.T
	binary   string
	env      []string
	url      string
	public   ed25519.PublicKey
	keys     map[string][]byte
	localApp string
}

func engineDir() string { return filepath.Join("..", "..") }
func repoDir() string   { return filepath.Join("..", "..", "..", "..") }

// buildTool runs "go build -o output [flags] package" in dir; a failing build fails the test.
func buildTool(t *testing.T, dir, output, pkg string, flags ...string) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("Go toolchain not installed")
	}
	command := exec.Command("go", append(append([]string{"build", "-o", output}, flags...), pkg)...)
	command.Dir = dir
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("cannot build %s: %v\n%s", pkg, err, out)
	}
}

func freeAddress(t *testing.T) string {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func startHostServer(t *testing.T) *hostE2E {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("the end-to-end test uses Windows DPAPI")
	}
	tools := hkTempDir(t)
	binary := filepath.Join(tools, "license-server.exe")
	buildTool(t, filepath.Join(repoDir(), "license-server"), binary, "./cmd/license-server")
	data := filepath.Join(hkTempDir(t), "data")
	if err := os.MkdirAll(data, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "signing.key"), []byte(hex.EncodeToString(testSeed)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	public := ed25519.NewKeyFromSeed(testSeed).Public().(ed25519.PublicKey)
	address := freeAddress(t)
	env := append(os.Environ(), "LICENSE_DATA_DIR="+data, "LICENSE_LISTEN_ADDR="+address, "LICENSE_POW_DIFFICULTY=10",
		"LICENSE_RATE_SENSITIVE=1000", "LICENSE_RATE_GENERAL=10000")
	process := exec.Command(binary, "serve")
	process.Env = env
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { process.Process.Kill(); process.Wait() })
	url := "http://" + address
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		if response, err := http.Get(url + "/api/v1/health"); err == nil {
			response.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the license server did not start")
		}
	}
	return &hostE2E{t: t, binary: binary, env: env, url: url, public: public, keys: map[string][]byte{KeyID(public): public},
		localApp: hkTempDir(t)}
}

func (h *hostE2E) issueKey() string {
	command := exec.Command(h.binary, "issue", "--email", "buyer@example.com")
	command.Env = h.env
	var out bytes.Buffer
	command.Stdout = &out
	if err := command.Run(); err != nil {
		h.t.Fatalf("issue: %v", err)
	}
	var issued struct{ License struct{ Key string } }
	if err := json.Unmarshal(out.Bytes(), &issued); err != nil || issued.License.Key == "" {
		h.t.Fatalf("issue answer %q: %v", out.String(), err)
	}
	return issued.License.Key
}

// hostHome is %LOCALAPPDATA%\ScadaBench of the helper process.
func (h *hostE2E) hostHome() string { return filepath.Join(h.localApp, "ScadaBench") }

func (h *hostE2E) service(home string, vm *VMOptions) *Service {
	return New(Options{Home: home, DataDir: home, Enforce: true, Keys: h.keys, AllowTestServer: true,
		Debugger: func() bool { return false }, Logf: h.t.Logf, Registry: newHKRegistry(),
		HostKey: HostKeyOptions{Protector: hostkey.System(), VM: vm}})
}

// startHelper runs the real helper program and waits until it answers on bind.
func (h *hostE2E) startHelper(bind string) *exec.Cmd {
	h.t.Helper()
	exe := filepath.Join(hkTempDir(h.t), "scadabench_hostkey.exe")
	buildTool(h.t, engineDir(), exe, "./cmd/scadabench_hostkey", "-tags", "hostkeytest")
	command := exec.Command(exe)
	command.Env = append(os.Environ(), "LOCALAPPDATA="+h.localApp, "SCADABENCH_HOSTKEY_TEST_BIND="+bind,
		"SCADABENCH_HOSTKEY_TEST_PUBKEY="+hex.EncodeToString(h.public))
	if err := command.Start(); err != nil {
		h.t.Fatal(err)
	}
	h.t.Cleanup(func() { command.Process.Kill(); command.Wait() })
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if conn, err := net.DialTimeout("tcp4", bind, 200*time.Millisecond); err == nil {
			conn.Close()
			return command
		}
	}
	h.t.Fatalf("the helper never listened on %s (log: %s)", bind, readLog(h.hostHome()))
	return nil
}

func readLog(home string) string {
	data, _ := os.ReadFile(filepath.Join(home, "vm_host.log"))
	return string(data)
}

func TestRealServerRealDPAPIAndTheRealHelperLicenseAVirtualMachine(t *testing.T) {
	h := startHostServer(t)
	hostHome := h.hostHome()
	host := h.service(hostHome, nil)
	if _, err := host.SetServerURL(h.url); err != nil {
		t.Fatal(err)
	}
	key := h.issueKey()
	if _, err := host.Activate(key); err != nil {
		t.Fatalf("activation with a host key: %v", err)
	}
	if status, _ := host.Current(true); !status.Licensed || status.Mode != "lifetime" {
		t.Fatalf("the host's own status: %+v", status)
	}
	if info := host.HostKeyInfo(); !info.Available {
		t.Fatalf("this PC cannot share its license: %+v", info)
	}
	if blob, err := os.ReadFile(filepath.Join(hostHome, hostkey.KeyFile)); err != nil || len(blob) < 50 {
		t.Fatalf("host_key.dpapi: %v (%d bytes)", err, len(blob))
	}
	code, err := host.ActivationCode()
	if err != nil || !strings.Contains(code, ".") {
		t.Fatalf("activation code %q: %v", code, err)
	}

	bind := freeAddress(t)
	helper := h.startHelper(bind)

	vmHome := hkTempDir(t)
	vm := h.service(vmHome, &VMOptions{Hosts: func() []string { return []string{bind} }, Timeout: 5 * time.Second})
	if status, _ := vm.Current(true); status.Licensed {
		t.Fatalf("a computer with no license of its own was licensed before asking: %+v", status)
	}
	if err := vm.RefreshVMHost(); err != nil {
		t.Fatalf("the VM was not confirmed by the real helper: %v\nhelper log: %s", err, readLog(hostHome))
	}
	status, err := vm.Current(true)
	if err != nil || !status.Licensed || status.Mode != ModeVMHost || status.Plan != "lifetime" || status.LicenseID != key {
		t.Fatalf("VM status %+v (%v)", status, err)
	}
	proofPath := filepath.Join(vmHome, VMProofFile)
	if _, err := os.Stat(proofPath); err != nil {
		t.Fatalf("no proof file for the native workers: %v", err)
	}
	checkNativeGuard(t, h, proofPath, vmHome)

	// Cancelling the host's license stops the helper from answering, even though license.cert is still there.
	if err := os.WriteFile(filepath.Join(hostHome, ".revoked_ids"), []byte(key), 0o666); err != nil {
		t.Fatal(err)
	}
	second := h.service(hkTempDir(t), &VMOptions{Hosts: func() []string { return []string{bind} }, Timeout: 5 * time.Second})
	if err := second.RefreshVMHost(); err == nil || !strings.Contains(err.Error(), RevokedText) {
		t.Fatalf("a cancelled host still licensed a VM: %v", err)
	}
	// Ending the helper (as turning sharing off does) leaves nobody to ask.
	helper.Process.Kill()
	helper.Wait()
	third := h.service(hkTempDir(t), &VMOptions{Hosts: func() []string { return []string{bind} }, Timeout: 2 * time.Second})
	if err := third.RefreshVMHost(); err == nil {
		t.Fatal("a VM was licensed although the helper is gone")
	}
}

// checkNativeGuard asks the native guard's own verifier (studio_license_check verify-vm) about the
// proof file the Go backend wrote; it skips when no native build is present.
func checkNativeGuard(t *testing.T, h *hostE2E, proofPath, vmHome string) {
	t.Helper()
	folder := os.Getenv("STUDIO_ENFORCED_NATIVE_BIN")
	if folder == "" {
		folder = filepath.Join(repoDir(), "studio", "native", "build", "license-enforced", "Release")
	}
	tool := filepath.Join(folder, "studio_license_check.exe")
	if _, err := os.Stat(tool); err != nil {
		t.Logf("native guard not checked: %s is not built", tool)
		return
	}
	hwid, err := MachineID()
	if err != nil {
		t.Fatal(err)
	}
	run := func(proof, machine string, now time.Time) string {
		command := exec.Command(tool, "verify-vm", proof, hex.EncodeToString(h.public), machine, fmt.Sprint(now.Unix()), vmHome)
		out, _ := command.CombinedOutput()
		return strings.TrimSpace(string(out))
	}
	if got := run(proofPath, hwid, time.Now()); got != "ok" {
		t.Fatalf("the native guard refused the proof the Go backend wrote: %q", got)
	}
	if got := run(proofPath, hwid, time.Now().Add(25*time.Hour)); !strings.Contains(got, "out of date") {
		t.Fatalf("the native guard accepted a proof older than 24 hours: %q", got)
	}
	if got := run(proofPath, strings.Repeat("0", 64), time.Now()); !strings.Contains(got, "another computer") {
		t.Fatalf("the native guard accepted the proof on another machine: %q", got)
	}
}
