package desktop

import (
	"crypto/sha256"
	"encoding/hex"
)

// showEventPrefix names the per-data-folder "show yourself" event (single_instance.py).
const showEventPrefix = `Local\ScadaBench-Show-`

// ShowEventName is single_instance.event_name: `Local\ScadaBench-Show-` plus the first 24
// hex digits of SHA-256 over the UTF-8 of os.path.normcase(os.path.abspath(dir)). On
// Windows that is GetFullPathNameW of the cleaned path, "/" turned into "\" and
// LCMapStringEx(LOCALE_NAME_INVARIANT, LCMAP_LOWERCASE), exactly as CPython 3.12 does,
// so a Go and a Python ScadaBench on one folder find each other.
func ShowEventName(dir string) string {
	digest := sha256.Sum256([]byte(normcase(abspath(dir))))
	return showEventPrefix + hex.EncodeToString(digest[:])[:24]
}

// runShow calls onShow and survives its panic: a failed focus must never stop the app.
func runShow(onShow func(), log *Logger) {
	defer func() {
		if failure := recover(); failure != nil {
			log.Error(singleInstanceLogger, "Could not show the ScadaBench window: %v", failure)
		}
	}()
	onShow()
}
