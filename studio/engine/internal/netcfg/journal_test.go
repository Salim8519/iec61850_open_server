package netcfg

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

type journalVectors struct {
	Times     []string `json:"times"`
	Scenarios []struct {
		Name    string            `json:"name"`
		Steps   []json.RawMessage `json:"steps"`
		Results []outcome         `json:"results"`
		File    string            `json:"file"`
	} `json:"scenarios"`
	Raw []struct {
		Name        string  `json:"name"`
		File        string  `json:"file"`
		Outstanding outcome `json:"outstanding"`
	} `json:"raw"`
	Keys []struct {
		Item json.RawMessage `json:"item"`
		Key  outcome         `json:"key"`
	} `json:"keys"`
}

// fixedClock returns the vector times in order, like the generator's FixedClock.
func fixedClock(t *testing.T, texts []string) func() time.Time {
	times := []time.Time{}
	for _, text := range texts {
		at, err := time.Parse("2006-01-02T15:04:05.999999-07:00", text)
		if err != nil {
			t.Fatal(err)
		}
		times = append(times, at)
	}
	calls := 0
	return func() time.Time {
		at := times[calls%len(times)]
		calls++
		return at
	}
}

func objectsText(t *testing.T, items []*pyjson.Object) string {
	return dumpsText(t, itemsAny(items))
}

func runJournalStep(t *testing.T, journal *Journal, step []any) (string, error) {
	switch step[0] {
	case "append":
		item, _ := step[2].(*pyjson.Object)
		record, err := journal.Append(pyStr(step[1]), item, pyStr(step[3]), step[4])
		if err != nil {
			return "", err
		}
		return dumpsText(t, record), nil
	case "outstanding":
		items, err := journal.Outstanding()
		if err != nil {
			return "", err
		}
		return objectsText(t, items), nil
	}
	kinds := map[string]bool{}
	for _, kind := range stringList(step[1]) {
		kinds[kind] = true
	}
	items, err := journal.Leftovers(func(item *pyjson.Object) (bool, error) {
		return kinds[pyStr(get(item, "kind"))], nil
	})
	if err != nil {
		return "", err
	}
	return objectsText(t, items), nil
}

// sameJournalBytes compares file bytes; Python wrote them on Windows (CRLF lines).
func sameJournalBytes(got, want []byte) bool {
	if runtime.GOOS != "windows" {
		want = []byte(strings.ReplaceAll(string(want), "\r\n", "\n"))
	}
	return string(got) == string(want)
}

func TestJournalMatchesPython(t *testing.T) {
	var v journalVectors
	loadVectors(t, "journal.json", &v)
	for _, scenario := range v.Scenarios {
		journal := NewJournal(t.TempDir())
		journal.Now = fixedClock(t, v.Times)
		for index, raw := range scenario.Steps {
			step, _ := py(t, raw).([]any)
			got, err := runJournalStep(t, journal, step)
			checkOutcome(t, fmt.Sprintf("%s step %d", scenario.Name, index), got, err, scenario.Results[index], true)
		}
		want, _ := base64.StdEncoding.DecodeString(scenario.File)
		got, err := os.ReadFile(journal.Path)
		if err != nil || !sameJournalBytes(got, want) {
			t.Errorf("%s: journal bytes differ (%v)\n got %q\nwant %q", scenario.Name, err, got, want)
		}
	}
	for _, raw := range v.Raw {
		journal := NewJournal(t.TempDir())
		data, _ := base64.StdEncoding.DecodeString(raw.File)
		os.MkdirAll(filepath.Dir(journal.Path), 0o777)
		os.WriteFile(journal.Path, data, 0o666)
		items, err := journal.Outstanding()
		got := ""
		if err == nil {
			got = objectsText(t, items)
		}
		checkOutcome(t, "raw "+raw.Name, got, err, raw.Outstanding, false)
	}
	for index, c := range v.Keys {
		key, err := ItemKey(pyObject(t, c.Item))
		checkOutcome(t, fmt.Sprintf("item_key %d", index), key, err, c.Key, true)
	}
}

func TestPythonISO(t *testing.T) {
	at := time.Date(2026, 10, 20, 10, 15, 30, 0, time.FixedZone("x", 7200))
	if got := PythonISO(at); got != "2026-10-20T08:15:30+00:00" {
		t.Fatal(got)
	}
	if got := PythonISO(at.Add(1500 * time.Nanosecond)); got != "2026-10-20T08:15:30.000001+00:00" {
		t.Fatal(got)
	}
}
