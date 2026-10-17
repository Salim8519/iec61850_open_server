package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestHelperRemoveMovesLicenseAsideNeverDeletes(t *testing.T) {
	folder := t.TempDir()
	write(t, filepath.Join(folder, "license.cert"), "cert")
	write(t, filepath.Join(folder, ".time_anchor"), "123")
	write(t, filepath.Join(folder, "client_config.json"), "{}")
	now := time.Date(2026, 10, 17, 9, 30, 0, 0, time.UTC)
	moved, err := moveAside(folder, licenseFiles, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"license.cert", ".time_anchor"} {
		if _, err := os.Stat(filepath.Join(moved, name)); err != nil {
			t.Errorf("%s was not kept in %s", name, moved)
		}
		if _, err := os.Stat(filepath.Join(folder, name)); err == nil {
			t.Errorf("%s is still active", name)
		}
	}
	if _, err := os.Stat(filepath.Join(folder, "client_config.json")); err != nil {
		t.Error("an unrelated file was touched")
	}
	write(t, filepath.Join(folder, "license.cert"), "again")
	second, err := moveAside(folder, licenseFiles, now)
	if err != nil || second == moved {
		t.Fatalf("a second removal in the same second reused %s (%v)", second, err)
	}
	if empty, err := moveAside(t.TempDir(), licenseFiles, now); empty != "" || err != nil {
		t.Fatalf("nothing to remove gave %q, %v", empty, err)
	}
}

func TestHelperInstallWritesStudioCertificateAndKeepsTheOldOne(t *testing.T) {
	folder := t.TempDir()
	write(t, filepath.Join(folder, "license.cert"), "old")
	write(t, filepath.Join(folder, ".time_anchor"), "123")
	certificate := map[string]any{"payload": `{"type":"lifetime","license_id":"SUBS-AAAA-BBBB-CCCC-DDDD",` +
		`"hwid":"ab","expires_at":0}`, "signature": "c2ln"}
	moved, err := installCertificate(folder, certificate, time.Now())
	if err != nil || moved == "" {
		t.Fatalf("install: %q, %v", moved, err)
	}
	payload, err := readPayload(folder)
	if err != nil || payload.LicenseID != "SUBS-AAAA-BBBB-CCCC-DDDD" || payload.Type != "lifetime" {
		t.Fatalf("readPayload: %+v, %v", payload, err)
	}
	if _, err := os.Stat(filepath.Join(folder, ".time_anchor")); err != nil {
		t.Error("the clock anchor must stay in place")
	}
	if _, err := installCertificate(folder, map[string]any{"error": "x"}, time.Now()); err == nil {
		t.Error("a missing certificate was installed")
	}
}
