package licensing

import (
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"studio-engine/internal/licensing/pytext"
)

// runsTailWindow is how much of the end of runs.jsonl time_anchor.py reads.
const runsTailWindow = 8192

// TimeAnchor remembers the latest wall-clock time ScadaBench has seen, to detect clock
// rollback (licensing/time_anchor.py). The file holds ASCII decimal epoch seconds.
type TimeAnchor struct {
	AnchorFile string
	RunsFile   string // "" when there is no run log
}

// readAnchorInt is time_anchor._read_int: ASCII text, stripped, int(); 0 on any error.
func readAnchorInt(path string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	for _, c := range data {
		if c >= 0x80 {
			return 0
		}
	}
	text := pytext.Strip(string(data))
	if text == "" {
		return 0
	}
	value, err := pytext.ParseInt(text)
	if err != nil {
		return 0
	}
	return saturate(value)
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

// MaxSeen is the newest of the anchor and the run log's latest "at" time.
func (a TimeAnchor) MaxSeen() int64 {
	latest := readAnchorInt(a.AnchorFile)
	if a.RunsFile != "" {
		latest = max(latest, latestRunTimestamp(a.RunsFile))
	}
	return latest
}

// Advance moves the anchor forward only, never backwards; write failures are ignored.
func (a TimeAnchor) Advance(now int64) {
	if now <= readAnchorInt(a.AnchorFile) {
		return
	}
	_ = pytext.WriteReplace(a.AnchorFile, []byte(strconv.FormatInt(now, 10)))
}
