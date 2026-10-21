package licensing

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// TestVerifiedStatusMatchesPython replays generated/answers.json (make_answers.py): a server
// answer counts only when it is signed for the request, with Python's exact verdicts.
func TestVerifiedStatusMatchesPython(t *testing.T) {
	var recorded struct {
		Cases []struct {
			Name    string          `json:"name"`
			Answer  json.RawMessage `json:"answer"`
			Request struct {
				Nonce     string          `json:"nonce"`
				LicenseID json.RawMessage `json:"license_id"`
				HWID      string          `json:"hwid"`
			} `json:"request"`
			Status *string `json:"status"`
		} `json:"cases"`
	}
	readVectors(t, "answers.json", &recorded)
	keys := testKeys(t)
	genuine := 0
	for _, c := range recorded.Cases {
		answer := mustDecode(t, string(c.Answer))
		licenseID := mustDecode(t, string(c.Request.LicenseID))
		got, ok := VerifiedStatus(answer, c.Request.Nonce, licenseID, c.Request.HWID, keys)
		switch {
		case c.Status == nil && ok:
			t.Errorf("%s: Go accepted %q, Python refused", c.Name, got)
		case c.Status != nil && (!ok || got != *c.Status):
			t.Errorf("%s: got %q %v, want %q", c.Name, got, ok, *c.Status)
		}
		if ok {
			genuine++
		}
	}
	if genuine == 0 || genuine == len(recorded.Cases) {
		t.Fatalf("%d of %d answers were genuine; the vectors must hold both kinds", genuine, len(recorded.Cases))
	}
}

func newRevoked(t *testing.T, registry Registry) (RevokedIDs, string) {
	home := scenarioDir(t)
	return NewRevokedIDs(home, registry), home
}

func TestACancelledIDLivesInThreePlacesAndIsRestoredFromTheOthers(t *testing.T) {
	registry := memoryRegistry{}
	revoked, home := newRevoked(t, registry)
	revoked.Add("SUBS-AAAA-BBBB-CCCC-DDDD")
	revoked.Add("SUBS-EEEE-FFFF-GGGG-HHHH")
	revoked.Add(5)       // not text: ignored
	revoked.Add("a,b")   // would corrupt the list: ignored
	revoked.Add("  \t ") // nothing to remember
	want := "SUBS-AAAA-BBBB-CCCC-DDDD,SUBS-EEEE-FFFF-GGGG-HHHH"
	file := filepath.Join(home, ".revoked_ids")
	if data, _ := os.ReadFile(file); string(data) != want || registry["RevokedLicenses"] != want {
		t.Fatalf("file %q registry %q, want %q", data, registry["RevokedLicenses"], want)
	}
	state, _ := os.ReadFile(filepath.Join(home, "license_state.json"))
	if !strings.Contains(string(state), `"revoked_ids": "`+want+`"`) {
		t.Fatalf("state %s", state)
	}
	for _, drop := range []func(){
		func() { os.Remove(file) },
		func() { delete(registry, "RevokedLicenses") },
		func() { os.Remove(filepath.Join(home, "license_state.json")) },
		func() { os.Remove(file); delete(registry, "RevokedLicenses") },
	} {
		revoked.Restore()
		drop()
		if !revoked.Contains("SUBS-EEEE-FFFF-GGGG-HHHH") || !revoked.Contains(" SUBS-AAAA-BBBB-CCCC-DDDD ") {
			t.Fatal("a deleted copy freed a cancelled license")
		}
		revoked.Restore()
		if data, _ := os.ReadFile(file); string(data) != want || registry["RevokedLicenses"] != want {
			t.Fatalf("copies not restored: file %q registry %q", data, registry["RevokedLicenses"])
		}
	}
	if revoked.Contains("SUBS-OTHR-OTHR-OTHR-OTHR") || revoked.Contains(nil) || revoked.Contains(int64(5)) {
		t.Fatal("an unrelated license counted as cancelled")
	}
}

func TestAnEnforcingServiceIgnoresTheConfiguredServerAndRefusesNewOnes(t *testing.T) {
	t.Setenv("STUDIO_LICENSE_CERT_PATH", "")
	home := scenarioDir(t)
	os.WriteFile(filepath.Join(home, "client_config.json"), []byte(`{"server_url": "http://127.0.0.1:9"}`), 0o666)
	service := New(Options{Home: home, DataDir: home, Enforce: true, Keys: testKeys(t),
		HWID: func() (string, error) { return "ab", nil }, Logf: t.Logf})
	if url, err := service.ServerURL(); err != nil || url != DefaultServerURL {
		t.Fatalf("server %q %v, want %q", url, err, DefaultServerURL)
	}
	for _, address := range []any{"http://127.0.0.1:9", "https://evil.example.com", "", nil} {
		_, err := service.SetServerURL(address)
		var refused *ActionError
		if !errors.As(err, &refused) || refused.Code != "bad_url" || refused.Message != ServerFixedText {
			t.Fatalf("address %v: %v", address, err)
		}
	}
	seam := New(Options{Home: home, DataDir: home, Enforce: true, Keys: testKeys(t), AllowTestServer: true,
		HWID: func() (string, error) { return "ab", nil }, Logf: t.Logf})
	if url, _ := seam.ServerURL(); url != "http://127.0.0.1:9" {
		t.Fatalf("the test-only seam must read the configured server, got %q", url)
	}
}

type recordingDoer struct{ urls []string }

func (d *recordingDoer) Do(request *http.Request) (*http.Response, error) {
	d.urls = append(d.urls, request.URL.String())
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader([]byte("{}")))}, nil
}

func TestTheProductClientOnlyTalksToTheBuiltInServer(t *testing.T) {
	doer := &recordingDoer{}
	if _, err := NewHTTPClient(DefaultServerURL+"/", doer).Verify("SUBS-X", "ab", strings.Repeat("0", 32)); err != nil {
		t.Fatal(err)
	}
	if len(doer.urls) != 1 || doer.urls[0] != DefaultServerURL+"/api/v1/verify" {
		t.Fatalf("urls %v", doer.urls)
	}
	for _, address := range []string{"http://127.0.0.1:9", "http://localhost:8090", "https://evil.example.com",
		DefaultServerURL + ".evil.example", ""} {
		_, err := NewHTTPClient(address, doer).Verify("SUBS-X", "ab", "0")
		var server *ServerError
		if !errors.As(err, &server) || server.Code != "no_server" {
			t.Fatalf("%q: %v", address, err)
		}
	}
	if len(doer.urls) != 1 {
		t.Fatalf("a refused address still reached the network: %v", doer.urls)
	}
}

func TestTheVerifyBodyCarriesTheNonce(t *testing.T) {
	var sent string
	doer := doerFunc(func(request *http.Request) (*http.Response, error) {
		data, _ := io.ReadAll(request.Body)
		sent = string(data)
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})
	nonce := RandomNonce()
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(nonce) || nonce == RandomNonce() {
		t.Fatalf("nonce %q must be 32 fresh lowercase hex characters", nonce)
	}
	NewHTTPClient(DefaultServerURL, doer).Verify("SUBS-X", "ab", nonce)
	if sent != `{"license_key": "SUBS-X", "hwid": "ab", "nonce": "`+nonce+`"}` {
		t.Fatalf("body %s", sent)
	}
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(request *http.Request) (*http.Response, error) { return f(request) }

// TestNoReleaseCodeEnablesTheTestSeam walks the engine's sources: only tests and the
// -tags parity build may set AllowTestServer or build a test-seam client (ADR-0097).
func TestNoReleaseCodeEnablesTheTestSeam(t *testing.T) {
	seam := regexp.MustCompile(`AllowTestServer\s*:|\.AllowTestServer\s*=|TestClientFactory|NewTestHTTPClient`)
	root := filepath.Join("..", "..")
	checked := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		checked++
		inLicensing := strings.Contains(filepath.ToSlash(path), "/internal/licensing/")
		if seam.Match(data) && !inLicensing && !strings.Contains(string(data), "//go:build parity") {
			t.Errorf("%s sets the test-only license server seam in release code", path)
		}
		return nil
	})
	if err != nil || checked < 100 {
		t.Fatalf("walked %d files: %v", checked, err)
	}
}

func TestAnswersAreObjectsOrRefused(t *testing.T) {
	for _, answer := range []any{nil, "valid", int64(1), []any{}, pyjson.NewObject()} {
		if status, ok := VerifiedStatus(answer, "n", "SUBS-X", "ab", testKeys(t)); ok || status != "" {
			t.Fatalf("%v counted as genuine", answer)
		}
	}
}
