package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/testutil"
)

// The page carries the offline policy: everything from this engine, nothing from the internet.
func TestPageForbidsLoadingFromTheInternet(t *testing.T) {
	folder := testutil.TempDir(t)
	if err := os.WriteFile(filepath.Join(folder, "index.html"), []byte("<!doctype html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	front, err := LoadFrontend(folder)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Token: "tok", SecretKey: "secret", Frontend: front})
	if err != nil {
		t.Fatal(err)
	}
	status, _, header := call(s, "GET", "/")
	policy := header.Get("Content-Security-Policy")
	if status != 200 || policy != OfflinePolicy {
		t.Fatalf("GET / = %d, policy %q", status, policy)
	}
	for _, rule := range []string{"default-src 'self'", "script-src 'self'", "connect-src 'self'", "font-src 'self' data:"} {
		if !strings.Contains(policy, rule) {
			t.Errorf("policy lacks %q", rule)
		}
	}
	if strings.Contains(policy, "http") || strings.Contains(policy, "*") {
		t.Errorf("policy allows an outside address: %q", policy)
	}
}
