package api

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/release"
	"substation-license-server/internal/security"
)

func sampleRelease() release.Info {
	return release.Info{Version: "0.1.14", Date: "2026-10-20", Notes: "Faster SLD.",
		Page:      "https://scadabench.com/download",
		Installer: release.Download{URL: "https://scadabench.com/download/ScadaBench-Setup-0.1.14.exe", SHA256: sha64("a")}}
}

func sha64(c string) string {
	out := ""
	for len(out) < 64 {
		out += c
	}
	return out
}

func getRelease(s *Server, etag string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/release", nil)
	request.RemoteAddr = "203.0.113.9:4000"
	if etag != "" {
		request.Header.Set("If-None-Match", etag)
	}
	recorder := httptest.NewRecorder()
	s.Handler().ServeHTTP(recorder, request)
	return recorder
}

func TestReleaseIsSignedWithADomainPrefixAndCached(t *testing.T) {
	s := newTestServer(t)
	if got := getRelease(s, ""); got.Code != http.StatusNotFound {
		t.Fatalf("no release yet: want 404, got %d", got.Code)
	}
	if err := release.Save(s.Config.DataDir, sampleRelease()); err != nil {
		t.Fatal(err)
	}
	if err := s.LoadRelease(); err != nil {
		t.Fatal(err)
	}
	got := getRelease(s, "")
	if got.Code != http.StatusOK || got.Header().Get("ETag") == "" || got.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("want a cacheable 200 without CORS, got %d %v", got.Code, got.Header())
	}
	var body struct{ Payload, Signature, Kid string }
	json.Unmarshal(got.Body.Bytes(), &body)
	signature, _ := base64.RawURLEncoding.DecodeString(body.Signature)
	if !ed25519.Verify(s.Signer.Public(), []byte(certs.ReleaseDomain+body.Payload), signature) {
		t.Fatal("release signature must cover the domain prefix and payload")
	}
	if ed25519.Verify(s.Signer.Public(), []byte(body.Payload), signature) {
		t.Fatal("a release signature must never verify as a plain (license) payload")
	}
	if again := getRelease(s, got.Header().Get("ETag")); again.Code != http.StatusNotModified || again.Body.Len() != 0 {
		t.Fatalf("matching ETag: want empty 304, got %d", again.Code)
	}
}

func TestReleaseCheckIsRateLimitedPerAddress(t *testing.T) {
	s := newTestServer(t)
	s.Limiter = security.NewLimiter(security.LimiterConfig{GeneralPerMinute: 2, SensitivePerMinute: 1,
		MaxFailures: 3, BanDuration: time.Minute, MaxEntries: 100})
	release.Save(s.Config.DataDir, sampleRelease())
	s.LoadRelease()
	codes := []int{getRelease(s, "").Code, getRelease(s, "").Code, getRelease(s, "").Code}
	if codes[2] != http.StatusTooManyRequests {
		t.Fatalf("third check within a minute must be refused, got %v", codes)
	}
}

func TestReleaseRecordRefusesForeignHostsAndBadValues(t *testing.T) {
	bad := []func(*release.Info){
		func(i *release.Info) { i.Installer.URL = "https://evil.example/ScadaBench.exe" },
		func(i *release.Info) { i.Installer.URL = "http://scadabench.com/x.exe" },
		func(i *release.Info) { i.Version = "1.0" },
		func(i *release.Info) { i.Installer.SHA256 = "ABC" },
		func(i *release.Info) { i.Page = "https://user@scadabench.com/" },
	}
	for index, change := range bad {
		info := sampleRelease()
		change(&info)
		if info.Validate() == nil {
			t.Fatalf("case %d must be refused", index)
		}
	}
	if _, err := release.Parse([]byte(`{"version":"0.1.14","extra":1}`)); err == nil {
		t.Fatal("unknown fields must be refused")
	}
}
