package app

import (
	"sort"
	"strings"
	"sync"
)

// IED addresses stay on their card for the session (ADR-0141, trial N10): Stop keeps them, so the
// next Run adds nothing. They come off when the project's addresses change while nothing runs
// (another project opened, New, Close, This PC only, an IED moved or removed), at exit, and after
// a kill at the next launch.

// labFollow remembers the project's lab addresses so only a real change starts a Release.
type labFollow struct {
	mu      sync.Mutex
	last    string
	pending bool
}

// labProjectChanged runs on every project_changed event; it must not block (push Listen).
func (e *Engine) labProjectChanged() {
	if e.Lab == nil || e.Runtime == nil || e.Projects == nil {
		return
	}
	key := labKey(labProjects{service: e.Projects})
	e.labFollow.mu.Lock()
	changed := key != e.labFollow.last
	e.labFollow.last = key
	start := changed && !e.labFollow.pending
	e.labFollow.pending = e.labFollow.pending || start
	e.labFollow.mu.Unlock()
	if start {
		go e.labReleaseAfterChange()
	}
}

// labReleaseAfterChange releases once nothing runs; a Run in progress releases at its Stop.
func (e *Engine) labReleaseAfterChange() {
	e.labFollow.mu.Lock()
	e.labFollow.pending = false
	e.labFollow.mu.Unlock()
	_ = e.Runtime.Stopped(func() error { e.labRelease(); return nil })
}

// labKey is the project's network and its simulated IEDs' lab addresses, in one string.
func labKey(projects labProjects) string {
	mode, guid := projects.Network()
	ieds, _ := projects.SimulatedIEDs()
	addresses := make([]string, 0, len(ieds))
	for _, ied := range ieds {
		addresses = append(addresses, ied.LabIP)
	}
	sort.Strings(addresses)
	return mode + " " + strings.ToUpper(guid) + " " + strings.Join(addresses, ",")
}

// labEndSession takes every address this copy put on a card off at exit (EndSession); what stays
// is logged, the event stream is already closed.
func (e *Engine) labEndSession() {
	if e.Lab == nil {
		return
	}
	if err := e.Lab.EndSession(); err != nil {
		e.options.Logf("Lab addresses left at exit: %v", err)
	}
}
