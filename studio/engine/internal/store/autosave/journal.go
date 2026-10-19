package autosave

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
)

// Journal is <data>/autosave.journal: one fsynced line per committed mutation, so no edit
// is lost when the process dies before the background snapshot. The file starts with a
// header naming the autosave it extends (its SHA-256); records then describe each
// mutation (see records.go). A snapshot writer marks a snapshot before renaming it in and
// afterwards rewrites the journal to the records newer than that snapshot. Fold applies a
// leftover journal at the next start (and in the launcher before Python starts).
//
// The first record of a journal is a full document unless the caller's previous document
// is exactly the autosave on disk (same project.json CRC-32 and length).
type Journal struct {
	dataDir string
	path    string
	enc     *archive.Encoder

	mu      sync.Mutex
	file    *os.File
	size    int64
	lastRev uint64
	records []line // records newer than the journal's base, for rewriting
}

type line struct {
	rev  uint64
	data []byte
}

// ErrNotFolded is returned by OpenJournal when a journal from an earlier run still exists.
var ErrNotFolded = errors.New("an autosave journal from an earlier run must be folded first")

// OpenJournal prepares the journal of dataDir; the file is created at the first Commit.
// enc should be the snapshot writer's Encoder so their caches are shared.
func OpenJournal(dataDir string, enc *archive.Encoder) (*Journal, error) {
	path := filepath.Join(dataDir, JournalName)
	if _, err := os.Lstat(path); err == nil {
		return nil, ErrNotFolded
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if enc == nil {
		enc = archive.NewEncoder()
	}
	return &Journal{dataDir: dataDir, path: path, enc: enc}, nil
}

// Commit durably records that the project changed from prev to next (compact documents,
// never mutated afterwards) as commit number rev, which must increase with every call.
// prev must be the document of the previous Commit, or the restored document for the first
// one; nil makes the record a full document. Non-finite floats are refused with the
// ValueError json.dumps(allow_nan=False) raises, as a save would.
func (j *Journal) Commit(rev uint64, prev, next *pyjson.Object) error {
	if err := archive.CheckFinite(next); err != nil {
		return err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if rev <= j.lastRev {
		return fmt.Errorf("autosave journal: commit %d after %d", rev, j.lastRev)
	}
	var lines [][]byte
	if j.file == nil {
		payload, base, err := autosaveDigest(j.dataDir)
		if err != nil {
			return err
		}
		if prev != nil && !j.matches(payload, prev) {
			prev = nil
		}
		// A new journal is self-contained: its base is the autosave as it is now.
		j.records = nil
		lines = append(lines, encodeLine(headerRecord(base)))
	}
	record := fullRecord(rev, next)
	if prev != nil {
		record = changeRecord(rev, prev, next)
	}
	data := encodeLine(record)
	if err := j.append(append(lines, data)...); err != nil {
		return err
	}
	j.lastRev = rev
	j.records = append(j.records, line{rev: rev, data: data})
	return nil
}

// matches reports whether the autosave holds exactly pyjson.Dumps(prev).
func (j *Journal) matches(payload []byte, prev *pyjson.Object) bool {
	if payload == nil {
		return false
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil || len(reader.File) != 1 {
		return false
	}
	crc, size, err := j.enc.Checksum(prev)
	entry := reader.File[0]
	return err == nil && entry.CRC32 == crc && entry.UncompressedSize64 == uint64(size)
}

func encodeLine(record *pyjson.Object) []byte {
	data, _ := pyjson.Compact(record) // records hold only pyjson values
	return append(data, '\n')
}

// append writes and fsyncs lines, creating the file first when needed; a failed write is
// cut off again so a refused mutation never reappears at the next start.
func (j *Journal) append(lines ...[]byte) error {
	if j.file == nil {
		file, err := os.OpenFile(j.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o666)
		if err != nil {
			return err
		}
		j.file, j.size = file, 0
	}
	data := bytes.Join(lines, nil)
	_, err := j.file.Write(data)
	if err == nil {
		err = j.file.Sync()
	}
	if err != nil {
		_ = j.file.Truncate(j.size)
		_, _ = j.file.Seek(j.size, 0)
		return err
	}
	j.size += int64(len(data))
	return nil
}

// markSnapshot records that snapshot rev with sha is about to replace the autosave.
func (j *Journal) markSnapshot(rev uint64, sha string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	return j.append(encodeLine(snapRecord(rev, sha)))
}

// snapshotted rewrites the journal after snapshot rev (sha) was renamed in: only the
// records newer than rev remain, behind a header naming the new autosave.
func (j *Journal) snapshotted(rev uint64, sha string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	kept := j.records[:0:0]
	lines := [][]byte{encodeLine(headerRecord(sha))}
	for _, record := range j.records {
		if record.rev > rev {
			kept = append(kept, record)
			lines = append(lines, record.data)
		}
	}
	temporary := j.path + ".tmp"
	data := bytes.Join(lines, nil)
	if err := writeSynced(temporary, data); err != nil {
		return err
	}
	if err := j.file.Close(); err != nil {
		return err
	}
	j.file = nil
	if err := os.Rename(temporary, j.path); err != nil {
		return err
	}
	file, err := os.OpenFile(j.path, os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return err
	}
	j.file, j.size, j.records = file, int64(len(data)), kept
	return nil
}

func writeSynced(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o666)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

// Close closes the journal and removes it when no record is newer than the autosave
// (after a final SnapshotWriter.Flush); otherwise it stays for Fold.
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.file == nil {
		return nil
	}
	err := j.file.Close()
	j.file = nil
	if err == nil && len(j.records) == 0 {
		err = os.Remove(j.path)
	}
	return err
}
