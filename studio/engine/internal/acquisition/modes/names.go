// Package modes holds the pure IEC 61850 mode rules of ScadaBench: which Mod/Beh/Sim
// points the station health check reads, the test-mode settings of simulated IEDs and the
// mode badges (ports of domain/health_reads.py, test_mode.py and mode_badges.py).
//
// Inputs and outputs are pyjson trees with Python dict/list semantics (see package pyv).
package modes

import (
	"math"

	"studio-engine/internal/acquisition/pyv"
)

// GocbAttributes returns GOCB_ATTRIBUTES: the GOOSE control block attributes health reads.
func GocbAttributes() [4]string {
	return [4]string{"GoEna", "GoID", "DatSet", "ConfRev"}
}

// ModeNames returns MODE_NAMES in order: index 0 is mode 1 ("on"), index 4 is mode 5 ("off").
func ModeNames() [5]string {
	return [5]string{"on", "on-blocked", "test", "test/blocked", "off"}
}

// ModeName is mode_name: the name of an integer mode 1..5. Booleans, floats and other
// values report false, as Python's isinstance(value, int) check excludes them.
func ModeName(value any) (string, bool) {
	if !pyv.IsInt(value) {
		return "", false
	}
	number, ok := pyv.Int64(value)
	if !ok || number < 1 || number > 5 {
		return "", false
	}
	return ModeNames()[number-1], true
}

// modeKey is Python's `value in MODE_NAMES` (and MODE_NAMES[value]): dict lookup uses hash
// equality, so True is mode 1 and 3.0 is mode 3. It returns the integer mode.
func modeKey(value any) (int64, bool) {
	var number int64
	switch v := value.(type) {
	case bool:
		if !v {
			return 0, false
		}
		number = 1
	case float64:
		if math.IsNaN(v) || v != math.Trunc(v) || v < 1 || v > 5 {
			return 0, false
		}
		number = int64(v)
	default:
		converted, ok := pyv.Int64(value)
		if !ok {
			return 0, false
		}
		number = converted
	}
	if number < 1 || number > 5 {
		return 0, false
	}
	return number, true
}

// modeRank is the position of a mode name in MODE_NAMES (higher is further from normal
// service), or -1 for any other text.
func modeRank(name string) int {
	for index, known := range ModeNames() {
		if known == name {
			return index
		}
	}
	return -1
}
