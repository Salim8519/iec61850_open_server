package licensing

import (
	"io"
	"math/big"
	"os"
	"strings"
	"unicode/utf8"
)

// runsTailWindow is how much of the end of runs.jsonl time_anchor.py reads.
const runsTailWindow = 8192

// TimeAnchor is the latest wall-clock time ScadaBench has seen, to detect clock rollback
// (licensing/time_anchor.py): the anchor copies (Value, three places) plus the run log
// tail. Each copy holds decimal epoch seconds.
type TimeAnchor struct {
	Value    ClockValue
	RunsFile string // "" when there is no run log
}

// latestRunTimestamp reads only the tail of runs.jsonl; its "at" fields are ISO-8601 times.
func latestRunTimestamp(path string) int64 {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0
	}
	if _, err := file.Seek(max(0, size-runsTailWindow), io.SeekStart); err != nil {
		return 0
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return 0
	}
	tail := strings.ToValidUTF8(string(data), string(utf8.RuneError))
	var latest int64
	for _, marker := range strings.Split(tail, `"at":"`)[1:] {
		value, _, _ := strings.Cut(marker, `"`)
		if parsed, ok := parseISOTime(value); ok {
			latest = max(latest, parsed.timestamp())
		}
	}
	return latest
}

// MaxSeen is the newest of the anchor copies (0 without any) and the run log's latest
// "at" time.
func (a TimeAnchor) MaxSeen() int64 {
	var latest *big.Int
	for _, copy := range a.Value.ReadInts() {
		if latest == nil || copy.Cmp(latest) > 0 {
			latest = copy
		}
	}
	result := int64(0)
	if latest != nil {
		result = saturate(latest)
	}
	if a.RunsFile != "" {
		result = max(result, latestRunTimestamp(a.RunsFile))
	}
	return result
}

// Advance moves the anchor forward only, and restores any copy that was deleted or
// lowered; write failures are ignored.
func (a TimeAnchor) Advance(now int64) {
	copies := a.Value.ReadInts()
	target := big.NewInt(now)
	for _, copy := range copies {
		if copy.Cmp(target) > 0 {
			target = copy
		}
	}
	stale := len(copies) < a.Value.Places()
	for _, copy := range copies {
		stale = stale || copy.Cmp(target) != 0
	}
	if stale {
		a.Value.Write(target.String())
	}
}
