package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const sealedName = "ScadaBench-License-Helper.exe"

// createHelper runs once, on the owner's PC. It creates a new owner signing key, locks
// it with the password inside a portable copy of this program (written to outDir),
// and saves only the PUBLIC key for the Studio release. Nothing goes over the network.
// The password never expires; it stays the helper's password for good.
func createHelper(program []byte, first, second, outDir, publicFile string) (string, error) {
	if len(first) < minPasswordLength {
		return "", fmt.Errorf("the password is too short: use at least %d characters", minPasswordLength)
	}
	if first != second {
		return "", errors.New("the two passwords are different; type the same one in both boxes")
	}
	// Create makes a NEW key that ScadaBench does not trust yet; never let it silently
	// replace the owner's existing helper (and password).
	out := filepath.Join(outDir, sealedName)
	if _, err := os.Stat(out); err == nil {
		return "", errors.New("you already have a helper on the Desktop. Press \"Update my helper\" " +
			"instead: it keeps your password and key")
	}
	secrets, public, err := newOwnerKey()
	if err != nil {
		return "", err
	}
	vault, err := sealSecrets(secrets, first, defaultKDF)
	if err != nil {
		return "", err
	}
	if _, err := openSecrets(vault, first); err != nil {
		return "", fmt.Errorf("self-check of the locked key failed, nothing was written: %w", err)
	}
	if err := os.WriteFile(out, appendVault(program, vault), 0o700); err != nil {
		return "", fmt.Errorf("cannot write %s (close it if it is open): %w", out, err)
	}
	text := fmt.Sprintf("%s\nkid %s\n", hex.EncodeToString(public), secrets.KeyID)
	if err := os.WriteFile(publicFile, []byte(text), 0o600); err != nil {
		return "", err
	}
	return "Done! Your helper is on the Desktop:\n" + out + "\n" +
		"Copy it anywhere and double-click it. Your password works forever.\n" +
		"Last step: tell Claude \"helper setup done\".", nil
}

// updateHelper moves an existing sealed helper onto this program version and keeps its
// vault, so the password and owner key stay exactly the same. The old file is kept.
func updateHelper(program []byte, helperPath, backupDir string, now time.Time) (string, error) {
	old, err := os.ReadFile(helperPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("there is no helper on the Desktop to update; use Create instead")
	}
	if err != nil {
		return "", err
	}
	vault := splitVault(old)
	var box sealed
	if vault == nil || json.Unmarshal(vault, &box) != nil || box.Version != 1 {
		return "", errors.New("the file on the Desktop is not a working helper")
	}
	if bytes.Equal(stripVault(old), stripVault(program)) {
		return "Your helper is already this version. Nothing was changed.", nil
	}
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", err
	}
	backup := filepath.Join(backupDir, "helper-"+now.Format("20060102-150405")+".exe")
	if err := os.Rename(helperPath, backup); err != nil {
		return "", fmt.Errorf("cannot move the old helper aside: %w", err)
	}
	if err := os.WriteFile(helperPath, appendVault(program, vault), 0o700); err != nil {
		os.Rename(backup, helperPath)
		return "", err
	}
	return "Done: your helper is updated. Your password is the same as before.\n" +
		"Close the old helper window if it is open, then open the helper again.", nil
}

// ownerPublicFile is kept with the other owner-only release state on this PC.
func ownerPublicFile() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}
	folder := filepath.Join(base, "ScadaBench-deploy")
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(folder, "owner-signing-key.txt"), nil
}
