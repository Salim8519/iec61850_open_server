package licensing

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestFromISOFormatMatchesPython(t *testing.T) {
	failures := 0
	for _, c := range loadUnits(t).ISO {
		parsed, ok := parseISOTime(c.Text)
		got := []int{parsed.year, parsed.month, parsed.day, parsed.hour, parsed.minute, parsed.second, parsed.micro}
		switch {
		case c.Error && ok:
			t.Errorf("%q: parsed %+v, Python refuses it", c.Text, parsed)
		case !c.Error && !ok:
			t.Errorf("%q: refused, Python parses it", c.Text)
		case c.TS != nil && (!parsed.aware || parsed.timestamp() != *c.TS):
			t.Errorf("%q: timestamp %d (aware %v), want %d", c.Text, parsed.timestamp(), parsed.aware, *c.TS)
		case c.Naive != nil && (parsed.aware || fmt.Sprint(got) != fmt.Sprint(c.Naive)):
			t.Errorf("%q: naive %v, want %v", c.Text, got, c.Naive)
		default:
			continue
		}
		if failures++; failures > 40 {
			t.Fatal("too many differences")
		}
	}
}

// saturated is Python's arbitrary-size int as the int64 the Go anchor keeps: values
// beyond int64 compare the same way against any real clock.
func saturated(t *testing.T, number json.Number) int64 {
	value, ok := new(big.Int).SetString(number.String(), 10)
	if !ok {
		t.Fatalf("bad number %s", number)
	}
	return saturate(value)
}

func writeOrRemove(t *testing.T, path string, data *string) {
	t.Helper()
	os.Remove(path)
	if data != nil {
		if err := os.WriteFile(path, fromB64(t, *data), 0o666); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTimeAnchorMatchesPython(t *testing.T) {
	folder := t.TempDir()
	value := NewClockValue("time_anchor", folder, nil)
	anchor := TimeAnchor{Value: value, RunsFile: filepath.Join(folder, "runs.jsonl")}
	for index, c := range loadUnits(t).Anchor {
		runs := c.Runs
		writeOrRemove(t, anchor.RunsFile, &runs)
		writeOrRemove(t, value.File, c.Anchor)
		writeOrRemove(t, value.StateFile, c.State)
		if got := anchor.MaxSeen(); got != saturated(t, c.MaxSeen) {
			t.Errorf("case %d: max_seen %d, want %s", index, got, c.MaxSeen)
		}
		if got := (TimeAnchor{Value: value}).MaxSeen(); got != saturated(t, c.AnchorOnly) {
			t.Errorf("case %d: anchor only %d, want %s", index, got, c.AnchorOnly)
		}
		for now, want := range c.Advance {
			// A fresh folder each time: Windows keeps a just-deleted file "delete pending"
			// while a scanner holds it, and replacing onto that name then fails.
			home := filepath.Join(folder, fmt.Sprintf("%d-%s", index, now))
			os.MkdirAll(home, 0o777)
			fresh := TimeAnchor{Value: NewClockValue("time_anchor", home, nil)}
			writeOrRemove(t, fresh.Value.File, c.Anchor)
			writeOrRemove(t, fresh.Value.StateFile, c.State)
			moment, _ := strconv.ParseInt(now, 10, 64)
			fresh.Advance(moment)
			for place, path := range []string{fresh.Value.File, fresh.Value.StateFile} {
				if got := readOptional(path); !sameText(got, want[place]) {
					t.Errorf("case %d: advance(%s) left %s = %v, want %v", index, now, filepath.Base(path),
						got, want[place])
				}
			}
		}
	}
}

// readOptional is a file's bytes as base64, or nil when it does not exist.
func readOptional(path string) *string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	text := b64Text(data)
	return &text
}
