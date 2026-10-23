package runtime

import (
	"strings"

	"studio-engine/internal/runtime/pyval"
)

// stampStartTimes mirrors what studio_ied did at start (ADR-0115): every timestamp "t" that
// holds no time yet reads the Run start time, so the values the app shows match the wire. A
// time restored from the previous run (startKinds) is kept. A worker that reports no
// started_at (an older build) changes nothing.
func stampStartTimes(item *running, startedAt any) {
	if _, ok := startedAt.(int64); !ok {
		return
	}
	previous := item.table.startTime()
	for _, entry := range item.model.signals {
		if entry.Kind != "Timestamp" || !strings.HasSuffix(entry.Ref, ".t") {
			continue
		}
		value, _ := item.table.get(entry.Ref)
		if !pyval.Truthy(value) || (previous != nil && pyval.Equal(value, previous)) {
			item.table.set(entry.Ref, startedAt)
		}
	}
	item.table.mu.Lock()
	item.table.started = startedAt
	item.table.mu.Unlock()
}

// startTime is the Run start time last stamped into this table (nil before the first Run).
func (t *valueTable) startTime() any {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.started
}
