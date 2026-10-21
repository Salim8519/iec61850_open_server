// Package projectfile owns the names of ScadaBench project files (ADR-0096): the .scadabench
// extension of project archives, the autosave and recovery copies the app writes itself, the
// refusal of the old .subsim extension and the startup rename of the app's own old files. It
// mirrors studio_app.projects.project_files and extension_migration byte for byte.
package projectfile

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// Names of project files. Only AutosaveName and the recovery folder are app-owned; DownloadName
// is what the browser saves when the user downloads the project.
const (
	Extension    = ".scadabench"
	OldExtension = ".subsim"
	AutosaveName = "autosave" + Extension
	DownloadName = "substation" + Extension
	// OldProjectCode is the error code of the refusal; the message is plain English for the user.
	OldProjectCode    = "old_project_file"
	OldProjectMessage = "This is an old project file (.subsim). ScadaBench now uses .scadabench project files."
)

// IsOldName reports whether a user-picked file name carries the old .subsim extension.
func IsOldName(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), OldExtension)
}

// UniqueName returns prefix + uuid4().hex + Extension, the name of a recovery backup.
func UniqueName(prefix string) string { return prefix + uuid4Hex() + Extension }

// uuid4Hex is uuid4().hex: 32 lowercase hex digits with the version and variant bits set.
func uuid4Hex() string {
	var random [16]byte
	_, _ = rand.Read(random[:])
	random[6] = random[6]&0x0f | 0x40 // version 4
	random[8] = random[8]&0x3f | 0x80 // RFC 4122 variant
	return hex.EncodeToString(random[:])
}
