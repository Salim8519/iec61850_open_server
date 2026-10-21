package modes

import (
	"strings"
	"unicode"
)

// The Python module matches mode references with regular expressions. These functions
// match exactly the same texts: `[^/]+` is everything before the first "/", `\d` is any
// Unicode decimal digit (category Nd, as Python's str patterns), and `$` also matches just
// before one trailing "\n", as Python's re does without MULTILINE.

// endsPattern reports whether rest is tail, or tail followed by a single "\n".
func endsPattern(rest, tail string) bool {
	return rest == tail || rest == tail+"\n"
}

// splitDomain returns the text before and after the first "/"; the part before must not
// be empty.
func splitDomain(ref string) (string, string, bool) {
	slash := strings.IndexByte(ref, '/')
	if slash <= 0 {
		return "", "", false
	}
	return ref[:slash], ref[slash+1:], true
}

// isSimulationRef matches ^([^/]+)/LPHD\d*\.Sim\.stVal$ (LPHD.Sim of a logical device).
func isSimulationRef(ref string) bool {
	_, rest, ok := splitDomain(ref)
	if !ok || !strings.HasPrefix(rest, "LPHD") {
		return false
	}
	return endsPattern(strings.TrimLeftFunc(rest[len("LPHD"):], unicode.IsDigit), ".Sim.stVal")
}

// behaviourRef matches ^([^/]+)/([^.]+)\.Beh\.stVal$ and returns both groups: the logical
// device and the logical node.
func behaviourRef(ref string) (string, string, bool) {
	domain, rest, ok := splitDomain(ref)
	if !ok {
		return "", "", false
	}
	dot := strings.IndexByte(rest, '.')
	if dot <= 0 || !endsPattern(rest[dot+1:], "Beh.stVal") {
		return "", "", false
	}
	return domain, rest[:dot], true
}

// isModeSignal is ref.endswith(MODE_SUFFIXES).
func isModeSignal(ref string) bool {
	for _, suffix := range []string{".Mod.stVal", ".Beh.stVal", ".Sim.stVal"} {
		if strings.HasSuffix(ref, suffix) {
			return true
		}
	}
	return false
}
