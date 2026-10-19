package licensing

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

// Tamper texts of licensing/integrity.py.
const (
	Tampered = "ScadaBench program files were changed or damaged. Reinstall ScadaBench."
	Debugger = "ScadaBench cannot run while a debugger is attached."
)

// FileDigest is the SHA-256 hex of a file.
func FileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// IntegrityProblem is integrity.problem: a user-readable reason, or "" when the
// installation is intact. expected maps shipped native file names to SHA-256 digests
// (empty in development); debugger reports an attached debugger.
func IntegrityProblem(nativeFolder string, expected map[string]string, debugger func() bool) string {
	if debugger() {
		return Debugger
	}
	for name, wanted := range expected {
		digest, err := FileDigest(filepath.Join(nativeFolder, name))
		if err != nil || digest != wanted {
			return Tampered
		}
	}
	return ""
}
