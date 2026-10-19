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
	anchor := TimeAnchor{AnchorFile: filepath.Join(folder, ".time_anchor"), RunsFile: filepath.Join(folder, "runs.jsonl")}
	for index, c := range loadUnits(t).Anchor {
		runs := c.Runs
		writeOrRemove(t, anchor.RunsFile, &runs)
		writeOrRemove(t, anchor.AnchorFile, c.Anchor)
		if got := anchor.MaxSeen(); got != saturated(t, c.MaxSeen) {
			t.Errorf("case %d: max_seen %d, want %s", index, got, c.MaxSeen)
		}
		if got := (TimeAnchor{AnchorFile: anchor.AnchorFile}).MaxSeen(); got != saturated(t, c.AnchorOnly) {
			t.Errorf("case %d: anchor only %d, want %s", index, got, c.AnchorOnly)
		}
		for now, want := range c.Advance {
			// A fresh folder each time: Windows keeps a just-deleted file "delete pending"
			// while a scanner holds it, and replacing onto that name then fails.
			fresh := TimeAnchor{AnchorFile: filepath.Join(folder, fmt.Sprintf("%d-%s", index, now), ".time_anchor")}
			if c.Anchor != nil {
				os.MkdirAll(filepath.Dir(fresh.AnchorFile), 0o777)
				writeOrRemove(t, fresh.AnchorFile, c.Anchor)
			}
			value, _ := strconv.ParseInt(now, 10, 64)
			fresh.Advance(value)
			data, err := os.ReadFile(fresh.AnchorFile)
			got := (*string)(nil)
			if err == nil {
				text := b64Text(data)
				got = &text
			}
			if !sameText(got, want) {
				t.Errorf("case %d: advance(%s) left %v, want %v", index, now, got, want)
			}
		}
	}
}
