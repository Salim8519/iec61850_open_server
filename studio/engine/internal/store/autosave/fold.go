package autosave

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
)

// ErrJournalMismatch means the autosave on disk is neither the journal's base nor one of
// the snapshots it marked (for example another program saved it); both files are kept.
var ErrJournalMismatch = errors.New("autosave journal does not belong to autosave.scadabench; both files were kept")

// Fold applies a leftover <data>/autosave.journal to autosave.scadabench and removes the
// journal only after the new autosave was renamed in. It returns the number of records
// applied (0 and no error when there is no journal). A damaged or partial last line is a
// torn write and is ignored; a damaged earlier line is an error and keeps both files.
// enc may be nil.
func Fold(dataDir string, enc *archive.Encoder) (int, error) {
	path := filepath.Join(dataDir, JournalName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	parsed, err := parseJournal(data)
	if err != nil {
		return 0, err
	}
	payload, current, err := autosaveDigest(dataDir)
	if err != nil {
		return 0, err
	}
	floor, accepted := parsed.floor(current)
	if !accepted && len(parsed.records) > 0 {
		return 0, ErrJournalMismatch
	}
	pending := parsed.after(floor)
	if len(pending) == 0 {
		return 0, os.Remove(path)
	}
	var doc *pyjson.Object
	if _, full := pending[0].record.Get("doc"); !full {
		if payload == nil {
			return 0, fmt.Errorf("%w (no autosave.scadabench)", errMissing)
		}
		if doc, err = archive.Decode(payload); err != nil {
			return 0, fmt.Errorf("autosave journal cannot be folded into an unreadable autosave: %w", err)
		}
	}
	pool := map[string]string{}
	if doc != nil {
		addTexts(pool, doc)
	}
	for _, item := range pending {
		if doc, err = apply(doc, item.record, pool); err != nil {
			return 0, fmt.Errorf("autosave journal record %d: %w", item.rev, err)
		}
	}
	if enc == nil {
		enc = archive.NewEncoder()
	}
	encoded, err := enc.Encode(doc)
	if err != nil {
		return 0, err
	}
	if err := WriteAutosave(dataDir, encoded); err != nil {
		return 0, err
	}
	return len(pending), os.Remove(path)
}

type parsedRecord struct {
	rev    uint64
	record *pyjson.Object
}

type parsedJournal struct {
	base    string
	hasBase bool
	snaps   map[string]uint64 // snapshot sha256 -> highest revision marked with it
	records []parsedRecord
}

// parseJournal reads every line; only the last line may be damaged (torn write).
func parseJournal(data []byte) (*parsedJournal, error) {
	lines := bytes.Split(data, []byte{'\n'})
	if len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	parsed := &parsedJournal{snaps: map[string]uint64{}}
	for index, text := range lines {
		value, err := pyjson.Decode(text)
		record, isObject := value.(*pyjson.Object)
		if err != nil || !isObject || !parsed.add(index, record) {
			if index == len(lines)-1 {
				break // torn final write
			}
			return nil, fmt.Errorf("autosave journal line %d is damaged; the journal and the autosave were kept", index+1)
		}
	}
	return parsed, nil
}

func (p *parsedJournal) add(index int, record *pyjson.Object) bool {
	if index == 0 {
		base, ok := record.Get("base")
		text, isText := base.(string)
		p.base, p.hasBase = text, ok && isText
		return p.hasBase
	}
	if sha, ok := record.Get("sha256"); ok {
		text, isText := sha.(string)
		rev, okRev := revision(record, "snap")
		if isText && okRev && rev > p.snaps[text] {
			p.snaps[text] = rev
		}
		return isText && okRev
	}
	rev, ok := revision(record, "rev")
	if ok {
		p.records = append(p.records, parsedRecord{rev: rev, record: record})
	}
	return ok
}

func revision(record *pyjson.Object, key string) (uint64, bool) {
	value, _ := record.Get(key)
	number, ok := value.(int64)
	return uint64(number), ok && number >= 0
}

// floor returns the revision the autosave with digest current already contains: 0 for the
// journal's base, N for a snapshot marked N.
func (p *parsedJournal) floor(current string) (uint64, bool) {
	rev, isSnap := p.snaps[current]
	if isSnap {
		return rev, true
	}
	return 0, p.hasBase && p.base == current
}

func (p *parsedJournal) after(floor uint64) []parsedRecord {
	var pending []parsedRecord
	for _, item := range p.records {
		if item.rev > floor {
			pending = append(pending, item)
		}
	}
	return pending
}
