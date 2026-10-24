// Package memtrim gives memory back to Windows after a large one-off job: opening or
// importing a project, Run start, Stop, removing IEDs (ADR-0116). Go keeps freed memory for
// reuse until its next collection; after these jobs nothing allocates for a while, so the
// process kept showing its peak. One collection a moment after the job returns it.
package memtrim

import (
	"runtime/debug"
	"sync"
	"time"
)

// quiet is how long after the last job the memory is returned: jobs in a row (open, then
// Run) cost one collection, and none runs while a job is still working.
const quiet = 2 * time.Second

var (
	mu    sync.Mutex
	timer *time.Timer
)

// Soon returns freed memory to the system once no job has asked for quiet seconds.
func Soon() {
	mu.Lock()
	defer mu.Unlock()
	if timer != nil {
		timer.Reset(quiet)
		return
	}
	timer = time.AfterFunc(quiet, func() {
		mu.Lock()
		timer = nil
		mu.Unlock()
		debug.FreeOSMemory()
	})
}
