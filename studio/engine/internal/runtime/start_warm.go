package runtime

import (
	"context"
	"time"

	"studio-engine/internal/memtrim"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// StartWarmDelay is the quiet time after the project opened or changed: edits in a row give one
// warm-up. Reading one text at a time keeps one processor busy, so the page loads alongside.
const StartWarmDelay = 500 * time.Millisecond

// StartWarmer reads, while the app is idle, what a Run start request takes from each IED's SCL
// text: the typed controls and writable deadbands (ADR-0133). The first Start after opening read
// all of it while seven IED processes were loading the same processors, and one text that reads
// in 0.12 s alone took up to 7 s there. The warmer fills the same caches startRequest reads, one
// text at a time, so the values sent to the workers are the ones the Run would have read itself.
type StartWarmer struct {
	project func() *pyjson.Object
	delay   time.Duration
	changed chan struct{}
	warmed  map[warmKey]string // SCL text per IED read by the last warm-up
}

// warmKey is one IED as the start-data caches key it (with its SCL text).
type warmKey struct{ id, name string }

// NewStartWarmer warms the IEDs of project() delay after each change.
func NewStartWarmer(project func() *pyjson.Object, delay time.Duration) *StartWarmer {
	return &StartWarmer{project: project, delay: delay, changed: make(chan struct{}, 1)}
}

// Changed notes a project change. It never blocks (the project service calls it under its
// mutation lock); several changes before the delay ends give one warm-up.
func (w *StartWarmer) Changed() {
	select {
	case w.changed <- struct{}{}:
	default:
	}
}

// Run warms after each change until ctx ends.
func (w *StartWarmer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.changed:
		}
		if !w.settle(ctx) {
			return
		}
		w.warm(ctx, w.project())
	}
}

// settle waits until no change came for the delay; false when ctx ended.
func (w *StartWarmer) settle(ctx context.Context) bool {
	timer := time.NewTimer(w.delay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-w.changed:
			timer.Reset(w.delay)
		case <-timer.C:
			return true
		}
	}
}

// warm reads the start-request SCL data of every IED of project, one after another, skipping
// IEDs whose text the last warm-up already read (a Run or an edit elsewhere changes nothing here,
// and asks for no memory trim). A text that cannot be read is left to the Run, which
// starts that IED as before (SPC/DPC controls only, nothing writable).
func (w *StartWarmer) warm(ctx context.Context, project *pyjson.Object) {
	if project == nil {
		return
	}
	warmed := map[warmKey]string{}
	read := false
	for _, item := range pyval.List(project, "ieds") {
		if ctx.Err() != nil {
			return // closing: nothing is left half done
		}
		ied, ok := item.(*pyjson.Object)
		if !ok {
			continue
		}
		key, source := warmKey{pyval.Str(ied, "id"), pyval.Str(ied, "name")}, pyval.Str(ied, "source")
		warmed[key] = source
		// The same text (usually the same string, compared without reading it) was read already.
		if previous, found := w.warmed[key]; source == "" || (found && previous == source) {
			continue
		}
		model := &iedModel{id: key.id, name: key.name, raw: ied}
		model.extendedControls()
		model.writableDeadbands()
		read = true
	}
	w.warmed = warmed
	if read {
		memtrim.Soon() // the parse buffers go back to Windows, as after opening a project
	}
}
