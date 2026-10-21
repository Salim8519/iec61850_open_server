package licensing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"studio-engine/internal/licensing/hostkey"
)

const activateKey = "SUBS-2222-3333-4444-5555"

// hkServer is a license server that answers like the real one for activation: the certificate it
// signs names the host key the request carried (and none when there was none).
type hkServer struct {
	signer *hkSigner
	url    string
	mu     sync.Mutex
	bodies []map[string]any
}

func newHKServer(t *testing.T, signer *hkSigner) *hkServer {
	s := &hkServer{signer: signer}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/challenge":
			w.Write([]byte(`{"nonce":"n1","difficulty":1,"timestamp":1,"mac":"m"}`))
		case "/api/v1/activate":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			s.mu.Lock()
			s.bodies = append(s.bodies, body)
			s.mu.Unlock()
			fields := map[string]any{"hwid": body["hwid"], "license_id": body["license_key"]}
			if hostKey, _ := body["host_key"].(string); hostKey != "" {
				fields["host_key"] = hostKey
			}
			bundle := signer.sign(t, fields)
			json.NewEncoder(w).Encode(map[string]any{"certificate": map[string]string{"payload": bundle.Payload, "signature": bundle.Signature}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	s.url = server.URL
	return s
}

func (s *hkServer) lastBody() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bodies[len(s.bodies)-1]
}

// computer is an app (Enforce) with this server; protector nil means a build with no host key.
func (s *hkServer) computer(t *testing.T, protector hostkey.Protector) *Service {
	t.Helper()
	home := hkTempDir(t)
	hwid := strings.Repeat("a", 64)
	return New(Options{Home: home, DataDir: home, Enforce: true, Keys: s.signer.keys(), AllowTestServer: true,
		HWID: func() (string, error) { return hwid, nil }, Debugger: func() bool { return false }, Logf: t.Logf,
		Registry: newHKRegistry(), HostKey: HostKeyOptions{Protector: protector},
		ClientFactory: func(string) Client { return NewTestHTTPClient(s.url, http.DefaultClient) }})
}

func certificateOfHome(t *testing.T, service *Service) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(service.options.Home, "license.cert"))
	if err != nil {
		t.Fatal(err)
	}
	var bundle Bundle
	var file struct{ Payload, Signature string }
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	bundle = Bundle{file.Payload, file.Signature}
	var payload map[string]any
	json.Unmarshal([]byte(bundle.Payload), &payload)
	return payload
}

func TestTheFirstActivationMakesTheHostKeyAndSendsOnlyItsPublicHalf(t *testing.T) {
	server := newHKServer(t, newHKSigner(t))
	protector := hostkey.FakeProtector{Owner: "alice@HOST"}
	app := server.computer(t, protector)
	if _, err := os.Stat(filepath.Join(app.options.Home, hostkey.KeyFile)); err == nil {
		t.Fatal("a key exists before the first activation")
	}
	if _, err := app.Activate(activateKey); err != nil {
		t.Fatal(err)
	}
	sent, _ := server.lastBody()["host_key"].(string)
	if _, err := hostkey.DecodeKey(sent); err != nil {
		t.Fatalf("the activation request carried %q: %v", sent, err)
	}
	if certificate := certificateOfHome(t, app); certificate["host_key"] != sent {
		t.Fatalf("the stored certificate names %v, not the key that was sent", certificate["host_key"])
	}
	blob, _ := os.ReadFile(filepath.Join(app.options.Home, hostkey.KeyFile))
	private, err := (hostkey.KeyStore{Path: filepath.Join(app.options.Home, hostkey.KeyFile), Protector: protector}).Load()
	if err != nil || len(blob) == 0 || strings.Contains(string(server.lastBodyText()), string(private.Seed())) {
		t.Fatalf("the private key reached the server or was not kept: %v", err)
	}
	info := app.HostKeyInfo()
	if !info.Available || info.Key != hostkey.MaskKey(sent) {
		t.Fatalf("host key info after activation: %+v", info)
	}
	// Activating again on the same PC keeps the same key pair.
	if _, err := app.Activate(activateKey); err != nil || server.lastBody()["host_key"] != sent {
		t.Fatalf("the second activation sent another key: %v", err)
	}
}

func (s *hkServer) lastBodyText() []byte {
	data, _ := json.Marshal(s.lastBody())
	return data
}

func TestABuildWithoutAProtectorNeverMakesOrSendsAKey(t *testing.T) {
	server := newHKServer(t, newHKSigner(t))
	app := server.computer(t, nil)
	if _, err := app.Activate(activateKey); err != nil {
		t.Fatal(err)
	}
	if _, has := server.lastBody()["host_key"]; has {
		t.Fatalf("a build with no host key sent one: %v", server.lastBody())
	}
	if _, err := os.Stat(filepath.Join(app.options.Home, hostkey.KeyFile)); err == nil {
		t.Fatal("a key file was written")
	}
	if code, err := app.ActivationCode(); err != nil || code != strings.Repeat("a", 64) {
		t.Fatalf("activation code %q: %v", code, err)
	}
	if info := app.HostKeyInfo(); info.Available || info.Reason != NoHostKeyBuildText {
		t.Fatalf("info %+v", info)
	}
}

func TestTheOfflineActivationCodeIsTheMachineIDAndTheHostKeyAndWorksEndToEnd(t *testing.T) {
	server := newHKServer(t, newHKSigner(t))
	app := server.computer(t, hostkey.FakeProtector{Owner: "alice@HOST"})
	code, err := app.ActivationCode()
	if err != nil {
		t.Fatal(err)
	}
	hwid, key, found := strings.Cut(code, ".")
	if !found || hwid != strings.Repeat("a", 64) || len(key) != 43 || strings.Contains(key, ".") {
		t.Fatalf("activation code %q", code)
	}
	if again, _ := app.ActivationCode(); again != code {
		t.Fatalf("the code changed: %q then %q", code, again)
	}
	// The activation page sends the two parts; the server's certificate is installed offline.
	bundle := server.signer.sign(t, map[string]any{"hwid": hwid, "host_key": key, "license_id": activateKey})
	pasted, _ := json.Marshal(map[string]string{"payload": bundle.Payload, "signature": bundle.Signature})
	if _, err := app.ActivateOffline(string(pasted)); err != nil {
		t.Fatal(err)
	}
	if info := app.HostKeyInfo(); !info.Available {
		t.Fatalf("a host key from an offline activation does not work: %+v", info)
	}
}

func TestActivationCodesAreOnlyForTheAppAndNeverInDevelopment(t *testing.T) {
	server := newHKServer(t, newHKSigner(t))
	development := New(Options{Home: hkTempDir(t), DataDir: hkTempDir(t), Enforce: false,
		HostKey: HostKeyOptions{Protector: hostkey.FakeProtector{Owner: "x"}}})
	if _, err := development.ActivationCode(); err == nil || !strings.Contains(err.Error(), DevelopmentText) {
		t.Fatalf("development: %v", err)
	}
	if info := development.HostKeyInfo(); info.Available || info.Reason != DevelopmentText {
		t.Fatalf("development info %+v", info)
	}
	_ = server
}

func TestAfterAWindowsReinstallTheOldKeyIsGoneAndActivatingAgainMakesANewOne(t *testing.T) {
	server := newHKServer(t, newHKSigner(t))
	protector := hostkey.FakeProtector{Owner: "alice@HOST"}
	app := server.computer(t, protector)
	if _, err := app.Activate(activateKey); err != nil {
		t.Fatal(err)
	}
	oldKey := server.lastBody()["host_key"]
	reinstalled := hostkey.FakeProtector{Owner: "alice@HOST-after-reinstall"} // same files, a new Windows user
	app.options.HostKey.Protector = reinstalled
	if info := app.HostKeyInfo(); info.Available || info.Reason != HostKeyUnusableText {
		t.Fatalf("the old blob must not open for the new user: %+v", info)
	}
	if _, err := app.Activate(activateKey); err != nil {
		t.Fatal(err)
	}
	if newKey := server.lastBody()["host_key"]; newKey == oldKey || newKey == nil {
		t.Fatalf("the key was not replaced: %v", newKey)
	}
	if info := app.HostKeyInfo(); !info.Available {
		t.Fatalf("after activating again: %+v", info)
	}
}

func TestHostKeyInfoSaysInPlainWordsWhySharingIsNotPossible(t *testing.T) {
	signer := newHKSigner(t)
	host := newHKHost(t, signer)
	service := func() *Service {
		return New(Options{Home: host.home, DataDir: host.home, Enforce: true, Keys: signer.keys(),
			HWID: func() (string, error) { return host.hwid, nil }, Clock: func() float64 { return *host.now },
			Debugger: func() bool { return false }, Logf: t.Logf, Registry: host.registry,
			HostKey: HostKeyOptions{Protector: host.protector}})
	}
	if info := service().HostKeyInfo(); info.Available || info.Reason != HostNoCertificateText {
		t.Fatalf("no license: %+v", info)
	}
	host.install(map[string]any{"type": "trial", "expires_at": 1_760_000_000 + 86400})
	if info := service().HostKeyInfo(); info.Available || info.Reason != HostNotLifetimeText {
		t.Fatalf("a trial: %+v", info)
	}
	host.install(map[string]any{"host_key": nil})
	if info := service().HostKeyInfo(); info.Available || info.Reason != HostNoKeyInCertText {
		t.Fatalf("an old certificate: %+v", info)
	}
	host.install(nil)
	if info := service().HostKeyInfo(); !info.Available || info.Reason != "" || info.Key == "" {
		t.Fatalf("a good license: %+v", info)
	}
	for _, text := range []string{HostNoCertificateText, HostNotLifetimeText, HostNoKeyInCertText, HostKeyUnusableText,
		HostKeyMismatchText, HostIsVMText, NoHostKeyBuildText} {
		if strings.Contains(strings.ToLower(text), "dpapi") || strings.Contains(text, "Ed25519") || strings.Contains(text, "base64") {
			t.Errorf("jargon in %q", text)
		}
	}
}
