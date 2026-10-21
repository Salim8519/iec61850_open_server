package projectfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Kinds of journal lines, in extension_migration.py.
const (
	KindRenamed      = "project_file_renamed"
	KindKeptBoth     = "project_file_kept_both"
	KindLeftAlone    = "project_file_left_alone"
	KindRenameFailed = "project_file_rename_failed"
)

// stopped is extension_migration.STOPPED: the text of the error that stops startup when the
// autosave cannot be renamed (the OS detail is in the journal, which differs per engine).
const stopped = "ScadaBench could not rename your saved project to the new file name (%s to %s). " +
	"Nothing was deleted. Close other programs that use the data folder, then start ScadaBench " +
	"again. Details are in logs/" + JournalName + "."

// move is one app-owned file to rename. Protected files stop startup when they cannot be renamed.
type move struct {
	old, new  string
	protected bool
}

// Migrate renames the app's own old-extension files in dataDir: autosave.subsim and
// recovery/*.subsim become .scadabench. Nothing is deleted or overwritten: when the new name
// exists both files stay and the fact is journaled. Every outcome is appended to
// logs/project-file-migrations.jsonl. It returns the journal records; an error only when the
// autosave itself could not be renamed (startup then stops to protect it). Call it with the
// data-folder lease held, before the autosave journal is folded.
func Migrate(dataDir string) ([]Record, error) {
	journal := newJournal(dataDir)
	var records []Record
	for _, item := range plan(dataDir) {
		record, ok := rename(journal, item)
		if !ok {
			continue
		}
		records = append(records, record)
		if item.protected && record.Kind == KindRenameFailed {
			return records, fmt.Errorf(stopped, filepath.Base(item.old), filepath.Base(item.new))
		}
	}
	return records, nil
}

// plan lists the autosave and every recovery/*.subsim (any letter case), in name order.
func plan(dataDir string) []move {
	items := []move{{filepath.Join(dataDir, "autosave"+OldExtension), filepath.Join(dataDir, AutosaveName), true}}
	folder := filepath.Join(dataDir, "recovery")
	if !isRealFolder(folder) {
		return items
	}
	entries, err := os.ReadDir(folder)
	if err != nil {
		return items
	}
	var names []string
	for _, entry := range entries {
		if strings.HasSuffix(strings.ToLower(entry.Name()), OldExtension) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		items = append(items, move{filepath.Join(folder, name),
			filepath.Join(folder, name[:len(name)-len(OldExtension)]+Extension), false})
	}
	return items
}

// isRealFolder is recovery.is_dir() and not recovery.is_symlink().
func isRealFolder(path string) bool {
	linked, err := os.Lstat(path)
	if err != nil || linked.Mode()&fs.ModeSymlink != 0 {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// rename moves one file; ok is false when there was nothing to move.
func rename(journal *journal, item move) (Record, bool) {
	info, err := os.Lstat(item.old)
	if err != nil {
		return Record{}, false
	}
	if !info.Mode().IsRegular() {
		return journal.write(KindLeftAlone, item.old, item.new, "not a regular file", ""), true
	}
	if _, err := os.Lstat(item.new); err == nil {
		return journal.write(KindKeptBoth, item.old, item.new,
			"the new name already exists; both files were kept", ""), true
	}
	if err := os.Rename(item.old, item.new); err != nil {
		var detail *fs.PathError
		text := err.Error()
		if errors.As(err, &detail) {
			text = detail.Err.Error()
		}
		return journal.write(KindRenameFailed, item.old, item.new, "", text), true
	}
	return journal.write(KindRenamed, item.old, item.new, "", ""), true
}
