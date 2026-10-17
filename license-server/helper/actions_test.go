package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testPassword = "eight888"

func testVault(t *testing.T) []byte {
	t.Helper()
	secrets, _, err := newOwnerKey()
	if err != nil {
		t.Fatal(err)
	}
	vault, err := sealSecrets(secrets, testPassword, fastKDF)
	if err != nil {
		t.Fatal(err)
	}
	return vault
}

func TestHelperButtonsNeedThePasswordEveryTime(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	vault, now := testVault(t), time.Now()
	cert := filepath.Join(licenseHome(), "license.cert")
	for _, password := range []string{"", "wrong-password"} {
		if _, err := makeLicenseHere(vault, password, now); err == nil {
			t.Fatalf("password %q made a license", password)
		}
		if _, err := os.Stat(cert); err == nil {
			t.Fatal("a refused click still wrote a license")
		}
	}
	if _, err := makeLicenseHere(vault, "wrong-password", now); !errors.Is(err, errWrongPassword) {
		t.Fatalf("wrong password gave %v", err)
	}
	if _, err := removeLicenseHere(vault, "wrong-password", now); !errors.Is(err, errWrongPassword) {
		t.Fatalf("remove with a wrong password gave %v", err)
	}
}

func TestHelperMakeThenRemoveOnThisPC(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	vault, now := testVault(t), time.Now()
	if text := statusText(); text != "This PC: no license." {
		t.Fatalf("status before: %q", text)
	}
	text, err := makeLicenseHere(vault, testPassword, now)
	if err != nil || !strings.Contains(text, "licensed forever") {
		t.Fatalf("make: %q, %v", text, err)
	}
	payload, err := readPayload(licenseHome())
	mine, _ := machineID()
	if err != nil || payload.HWID != mine || payload.ExpiresAt != 0 || !strings.HasPrefix(payload.LicenseID, "OWNER-") {
		t.Fatalf("installed license: %+v, %v", payload, err)
	}
	if text := statusText(); !strings.HasPrefix(text, "This PC: licensed forever (OWNER-") {
		t.Fatalf("status after make: %q", text)
	}
	again, err := makeLicenseHere(vault, testPassword, now)
	if err != nil || !strings.Contains(again, "already has a working license") {
		t.Fatalf("second make: %q, %v", again, err)
	}
	if same, _ := readPayload(licenseHome()); same.LicenseID != payload.LicenseID {
		t.Fatal("a second Make replaced a working license")
	}
	if folders, _ := filepath.Glob(filepath.Join(licenseHome(), "removed-*")); len(folders) != 0 {
		t.Fatalf("a second Make moved files aside: %v", folders)
	}
	copied, err := licenseText()
	if file, _ := os.ReadFile(filepath.Join(licenseHome(), "license.cert")); err != nil || copied != string(file) {
		t.Fatalf("license text: %v", err)
	}
	text, err = removeLicenseHere(vault, testPassword, now)
	if err != nil || !strings.Contains(text, "removed") {
		t.Fatalf("remove: %q, %v", text, err)
	}
	kept, _ := filepath.Glob(filepath.Join(licenseHome(), "removed-*", "license.cert"))
	if len(kept) != 1 || statusText() != "This PC: no license." {
		t.Fatalf("after remove: kept %v, status %q", kept, statusText())
	}
	if text, err := removeLicenseHere(vault, testPassword, now); err != nil || !strings.Contains(text, "no license") {
		t.Fatalf("second remove: %q, %v", text, err)
	}
	if _, err := licenseText(); err == nil {
		t.Fatal("Copy found a license after removal")
	}
}

func TestHelperUpdateKeepsThePasswordAndKey(t *testing.T) {
	desktop, backups := t.TempDir(), filepath.Join(t.TempDir(), "helper-backups")
	helper, vault := filepath.Join(desktop, sealedName), testVault(t)
	if _, err := updateHelper([]byte("MZ new"), helper, backups, time.Now()); err == nil {
		t.Fatal("update without a helper on the Desktop succeeded")
	}
	os.WriteFile(helper, []byte("MZ not sealed"), 0o700)
	if _, err := updateHelper([]byte("MZ new"), helper, backups, time.Now()); err == nil {
		t.Fatal("an unsealed file was treated as a helper")
	}
	os.WriteFile(helper, appendVault([]byte("MZ old"), vault), 0o700)
	text, err := updateHelper([]byte("MZ new"), helper, backups, time.Now())
	if err != nil || !strings.Contains(text, "same as before") {
		t.Fatalf("update: %q, %v", text, err)
	}
	updated, _ := os.ReadFile(helper)
	if !strings.HasPrefix(string(updated), "MZ new") || string(splitVault(updated)) != string(vault) {
		t.Fatal("the updated helper lost its vault or kept the old program")
	}
	if _, err := openSecrets(splitVault(updated), testPassword); err != nil {
		t.Fatal("the same password no longer opens the updated helper")
	}
	kept, _ := filepath.Glob(filepath.Join(backups, "helper-*.exe"))
	if len(kept) != 1 {
		t.Fatalf("the old helper was not kept: %v", kept)
	}
	if text, _ := updateHelper([]byte("MZ new"), helper, backups, time.Now()); !strings.Contains(text, "already this version") {
		t.Fatalf("second update: %q", text)
	}
}

func TestHelperSetupCreatesAPortableHelperWithAPermanentPassword(t *testing.T) {
	out, public := t.TempDir(), filepath.Join(t.TempDir(), "owner-signing-key.txt")
	program := []byte("MZ program")
	if _, err := createHelper(program, "seven77", "seven77", out, public); err == nil {
		t.Fatal("a 7-character password was accepted")
	}
	if _, err := createHelper(program, testPassword, "eight889", out, public); err == nil {
		t.Fatal("two different passwords were accepted")
	}
	if text, err := createHelper(program, testPassword, testPassword, out, public); err != nil || !strings.Contains(text, "Done") {
		t.Fatalf("create: %q, %v", text, err)
	}
	sealed, err := os.ReadFile(filepath.Join(out, sealedName))
	if err != nil || !strings.HasPrefix(string(sealed), "MZ program") {
		t.Fatalf("sealed helper: %v", err)
	}
	secrets, err := openSecrets(splitVault(sealed), testPassword)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(public)
	if _, err := createHelper([]byte("MZ newer"), "another-pass", "another-pass", out, public); err == nil ||
		!strings.Contains(err.Error(), "Update my helper") {
		t.Fatalf("Create replaced an existing helper: %v", err)
	}
	if after, _ := os.ReadFile(filepath.Join(out, sealedName)); string(after) != string(sealed) {
		t.Fatal("the existing helper file was changed")
	}
	key, _ := os.ReadFile(public)
	if string(key) != string(before) {
		t.Fatal("a refused Create changed the owner key file")
	}
	if !strings.Contains(string(key), "kid "+secrets.KeyID) || strings.Contains(string(key), secrets.Seed) {
		t.Fatalf("public key file is wrong or leaks the private seed: %s", key)
	}
}
