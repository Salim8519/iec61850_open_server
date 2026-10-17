package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// The buttons. Make and Remove check the password on every click and decrypt the owner
// key only for that moment; the key is wiped right after, so nothing stays unlocked.

// unlockSigner opens the vault with the password typed in the window.
func unlockSigner(vault []byte, password string) (*Signer, error) {
	if password == "" {
		return nil, errors.New("type your password first")
	}
	secrets, err := openSecrets(vault, password)
	if err != nil {
		return nil, err
	}
	return newSigner(secrets)
}

// makeLicenseHere licenses this PC forever. It never buries a license that still
// works here (for example a bought key): the owner presses Remove first on purpose.
func makeLicenseHere(vault []byte, password string, now time.Time) (string, error) {
	signer, err := unlockSigner(vault, password)
	if err != nil {
		return "", err
	}
	defer signer.wipe()
	hwid, err := machineID()
	if err != nil {
		return "", err
	}
	if existing, err := readPayload(licenseHome()); err == nil && existing.HWID == hwid &&
		(existing.ExpiresAt == 0 || now.Unix() < existing.ExpiresAt) {
		return "This PC already has a working license (" + existing.LicenseID + ").\n" +
			"Nothing was changed. To replace it, press Remove first.", nil
	}
	bundle, payload, err := signer.licenseFor(hwid, 0, now)
	if err != nil {
		return "", err
	}
	if _, err := installCertificate(licenseHome(), bundle, now); err != nil {
		return "", err
	}
	return "Done: this PC is licensed forever (" + payload.LicenseID + ").\n" +
		"Reopen ScadaBench, or refresh its page.", nil
}

// removeLicenseHere takes the license off this PC; the files are kept, not deleted.
func removeLicenseHere(vault []byte, password string, now time.Time) (string, error) {
	signer, err := unlockSigner(vault, password)
	if err != nil {
		return "", err
	}
	signer.wipe()
	moved, err := moveAside(licenseHome(), licenseFiles, now)
	if err != nil {
		return "", err
	}
	if moved == "" {
		return "There was no license on this PC. Nothing was changed.", nil
	}
	return "Done: the license was removed from this PC.\n" +
		"A backup copy was kept in its " + filepath.Base(moved) + " folder.", nil
}

// licenseText returns this PC's license file text for the Copy button. A license is
// bound to one Machine ID and holds no secret, so copying needs no password.
func licenseText() (string, error) {
	raw, err := os.ReadFile(filepath.Join(licenseHome(), "license.cert"))
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("this PC has no license to copy; press Make first")
	}
	return string(raw), err
}

// statusText describes this PC's license in plain words for the top of the window.
func statusText() string {
	hwid, err := machineID()
	if err != nil {
		return "This PC: the Machine ID cannot be read (" + err.Error() + ")"
	}
	payload, err := readPayload(licenseHome())
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "This PC: no license."
	case err != nil:
		return "This PC: " + err.Error()
	case payload.HWID != hwid:
		return "This PC: has a license for ANOTHER PC, so ScadaBench refuses it."
	case payload.ExpiresAt == 0:
		return "This PC: licensed forever (" + payload.LicenseID + ")."
	}
	left := time.Until(time.Unix(payload.ExpiresAt, 0))
	if left <= 0 {
		return "This PC: the trial license has expired."
	}
	return fmt.Sprintf("This PC: trial license, %.1f days left.", left.Hours()/24)
}
