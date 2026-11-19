package recovery

import (
	"os"
	"path/filepath"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/autosave"
	"studio-engine/internal/store/projectfile"
)

// Texts of the notice shown when the autosave journal of the last Go run could not be folded.
const (
	journalKeptMessage = "The record of your latest unsaved edits could not be read. " +
		"The last autosave was opened without them; the record was moved to the recovery folder."
	journalEmptyMessage = "The record of your latest unsaved edits could not be read. " +
		"ScadaBench started with an empty substation; the record was moved to the recovery folder."
)

// JournalNotice describes an unreadable journal moved aside by QuarantineJournal.
type JournalNotice struct {
	details string // the fold error
	backup  string // where the journal now is
}

// QuarantineJournal moves <data>/autosave.journal, which Fold could not apply (cause), to
// recovery/unreadable-journal-<uuid>.journal and copies autosave.scadabench (when present) to
// recovery/unreadable-journal-<uuid>.scadabench, so the pair stays together for inspection.
// The autosave itself stays in place and nothing is deleted. On an error nothing was moved
// and startup must stop as before.
func QuarantineJournal(dataDir string, cause error) (*JournalNotice, error) {
	folder := filepath.Join(dataDir, FolderName)
	if isSymlink(folder) {
		return nil, runtimeError("Recovery folder is a link; cannot safely preserve the unreadable autosave journal")
	}
	if err := makeFolder(folder); err != nil {
		return nil, err
	}
	copyName := filepath.Join(folder, uniqueName("unreadable-journal-"))
	journalName := strings.TrimSuffix(copyName, projectfile.Extension) + ".journal"
	saved := filepath.Join(dataDir, projectfile.AutosaveName)
	if _, err := os.Stat(saved); err == nil {
		if err := copyFile(saved, copyName); err != nil {
			return nil, err
		}
	}
	if err := os.Rename(filepath.Join(dataDir, autosave.JournalName), journalName); err != nil {
		return nil, err
	}
	return &JournalNotice{details: cause.Error(), backup: journalName}, nil
}

// Apply returns the startup recovery result with the journal notice: the notice replaces
// "no autosave" and "restored" (one notice; the restored project lacks the journal's
// edits). An unreadable autosave keeps its own notice, which already names the problem.
func (n *JournalNotice) Apply(result *pyjson.Object) *pyjson.Object {
	if n == nil {
		return result
	}
	message := journalEmptyMessage
	if result != nil {
		if status, _ := result.Get("status"); status != "restored" {
			return result
		}
		message = journalKeptMessage
	}
	return pyjson.NewObject().Set("status", "unreadable").Set("message", message).
		Set("details", n.details).Set("backup", n.backup)
}
