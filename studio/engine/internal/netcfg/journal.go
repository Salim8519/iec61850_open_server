package netcfg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// journalEvents are the events journal.append accepts.
var journalEvents = map[string]bool{"applied": true, "removed": true, "failed": true, "skipped": true}

// Journal is the append-only record of Windows network changes ScadaBench itself made
// (<data>/network/journal.jsonl): one json.dumps(record, sort_keys=True) per line, written in
// text mode (CRLF on Windows), never rewritten.
type Journal struct {
	Path string
	Now  func() time.Time
	mu   sync.Mutex
}

// NewJournal returns the journal of a data folder.
func NewJournal(directory string) *Journal {
	return &Journal{Path: filepath.Join(directory, "network", "journal.jsonl"), Now: time.Now}
}

// PythonISO is datetime.isoformat() of an aware UTC time: microseconds only when non-zero.
func PythonISO(at time.Time) string {
	at = at.UTC()
	text := at.Format("2006-01-02T15:04:05")
	if micro := at.Nanosecond() / 1000; micro != 0 {
		text += fmt.Sprintf(".%06d", micro)
	}
	return text + "+00:00"
}

// textNewline is what "\n" becomes in a Python text-mode file on this system.
func textNewline() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

// Append writes one record and returns it (journal.append).
func (j *Journal) Append(event string, item *pyjson.Object, fingerprint string, result any) (*pyjson.Object, error) {
	if !journalEvents[event] {
		return nil, apperr.ValueError("Unknown journal event")
	}
	record := pyjson.NewObject().Set("at", PythonISO(j.Now())).Set("event", event).
		Set("fingerprint", fingerprint).Set("item", item).Set("result", cutRunes(pyStr(result), 500))
	line, err := dumpsSorted(record)
	if err != nil {
		return nil, err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(j.Path), 0o777); err != nil {
		return nil, err
	}
	stream, err := os.OpenFile(j.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		return nil, err
	}
	if _, err = stream.Write(append(line, textNewline()...)); err != nil {
		stream.Close()
		return nil, err
	}
	return record, stream.Close()
}

// Records reads every record (journal.records); a corrupt line is an error.
func (j *Journal) Records() ([]*pyjson.Object, error) {
	j.mu.Lock()
	data, err := os.ReadFile(j.Path)
	j.mu.Unlock()
	if errors.Is(err, os.ErrNotExist) {
		return []*pyjson.Object{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(data) {
		return nil, apperr.ValueError("'utf-8' codec can't decode the network journal")
	}
	text := strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n", "\n"), "\r", "\n")
	records := []*pyjson.Object{}
	for _, line := range pySplitlines(text) {
		if pyStrip(line) == "" {
			continue
		}
		value, err := pyjson.Decode([]byte(line))
		if err != nil {
			return nil, apperr.ValueError(err.Error())
		}
		record, ok := value.(*pyjson.Object)
		if !ok {
			return nil, apperr.ValueError("journal record is not an object")
		}
		records = append(records, record)
	}
	return records, nil
}

// Outstanding is journal.outstanding: owned items applied and not removed yet, newest state
// wins, in the order their keys first appeared.
func (j *Journal) Outstanding() ([]*pyjson.Object, error) {
	records, err := j.Records()
	if err != nil {
		return nil, err
	}
	state := pyjson.NewObject() // ordered like a Python dict
	for _, record := range records {
		raw, found := record.Get("item")
		if !found {
			return nil, apperr.KeyError("item")
		}
		item, _ := raw.(*pyjson.Object)
		if item == nil {
			return nil, apperr.ValueError("journal item is not an object")
		}
		if owned, _ := item.Get("owned"); !truthy(owned) {
			continue
		}
		event, found := record.Get("event")
		if !found {
			return nil, apperr.KeyError("event")
		}
		if event != "applied" && event != "removed" {
			continue
		}
		key, err := ItemKey(item)
		if err != nil {
			return nil, err
		}
		if event == "applied" {
			state.Set(key, item)
		} else {
			state.Delete(key)
		}
	}
	items := []*pyjson.Object{}
	for _, key := range state.Keys() {
		value, _ := state.Get(key)
		items = append(items, value.(*pyjson.Object))
	}
	return items, nil
}

// Leftovers is journal.leftovers: outstanding items that still exist; reported to the user,
// never removed automatically.
func (j *Journal) Leftovers(isPresent func(item *pyjson.Object) (bool, error)) ([]*pyjson.Object, error) {
	outstanding, err := j.Outstanding()
	if err != nil {
		return nil, err
	}
	found := []*pyjson.Object{}
	for _, item := range outstanding {
		present, err := isPresent(item)
		if err != nil {
			return nil, err
		}
		if present {
			found = append(found, item)
		}
	}
	return found, nil
}
