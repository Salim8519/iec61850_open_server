package acquisition

import (
	"strings"

	"studio-engine/internal/acquisition/livedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

var liveSources = map[string]bool{"live_report": true, "live_poll": true, "live_goose": true}

// LiveValues applies values received from a real IED to the runtime value table
// (acquisition/live_values.py). Outward `signal` events are exactly Python's: one per
// received point (or its owner), even when the value did not change.
type LiveValues struct {
	deps Deps
	refs refIndex
}

// NewLiveValues returns the writer.
func NewLiveValues(deps Deps) *LiveValues { return &LiveValues{deps: deps} }

// Apply stores raw device values {ref: device JSON}; unknown refs are ignored and bad
// values reported. It returns {"applied": n, "errors": {ref: message}}.
func (l *LiveValues) Apply(iedID string, raw *pyjson.Object, source string) (*pyjson.Object, error) {
	if !liveSources[source] {
		return nil, apperr.ValueError("Unknown live source")
	}
	var index map[string]*pyjson.Object
	if l.deps.Values.Has(iedID) {
		if ied := findIED(l.deps.Projects.Current(), iedID); ied != nil {
			index = l.refs.For(ied)
		}
	}
	changed := pyjson.NewObject()
	errors := pyjson.NewObject()
	var order []string
	if raw != nil {
		for _, ref := range raw.Keys() {
			signal := index[ref]
			if signal == nil {
				continue
			}
			value, _ := raw.Get(ref)
			converted, err := livedom.Convert(pyv.GetStr(signal, "kind"), value)
			if err != nil {
				errors.Set(ref, err.Error())
				continue
			}
			if !pyv.Has(changed, ref) {
				order = append(order, ref)
			}
			changed.Set(ref, converted)
		}
	}
	updates := make([]RefValue, 0, len(order))
	for _, ref := range order {
		value, _ := changed.Get(ref)
		updates = append(updates, RefValue{Ref: ref, Value: value})
	}
	if len(updates) > 0 {
		l.deps.Values.SetLive(iedID, updates)
	}
	l.emitSignals(iedID, index, changed, order, source)
	for _, ref := range errors.Keys() {
		message, _ := errors.Get(ref)
		l.deps.Events.Emit("live_read_error", pyv.O("ied_id", iedID, "ref", ref, "message", message))
	}
	keys := make([]PointKey, 0, len(order))
	for _, ref := range order {
		keys = append(keys, PointKey{IED: iedID, Ref: ref})
	}
	l.deps.Equipment.PublishChanged(keys)
	return pyv.O("applied", len(order), "errors", errors), nil
}

func (l *LiveValues) current(iedID string, ref any) (any, bool) {
	text, ok := ref.(string)
	if !ok {
		return nil, false
	}
	value, _, found := l.deps.Values.Get(iedID, text)
	return value, found
}

func (l *LiveValues) emitSignals(iedID string, index map[string]*pyjson.Object, changed *pyjson.Object,
	order []string, source string) {
	emitted := map[string]bool{}
	for _, ref := range order {
		signal := index[ref]
		target := signal
		if kind := pyv.GetStr(signal, "kind"); kind == "Quality" || kind == "Timestamp" {
			if owner := companionOwner(index, signal); owner != nil {
				target = owner
			}
		}
		targetRef := pyv.GetStr(target, "ref")
		if emitted[targetRef] {
			continue
		}
		emitted[targetRef] = true
		updates := pyjson.NewObject()
		for _, item := range []any{targetRef, pyv.Get(target, "quality_ref"), pyv.Get(target, "timestamp_ref")} {
			text, _ := item.(string)
			if text != "" && index[text] != nil && pyv.Has(changed, text) && !pyv.Has(updates, text) {
				value, _ := l.current(iedID, text)
				updates.Set(text, value)
			}
		}
		quality := any(int64(0))
		if index[pyv.GetStr(target, "quality_ref")] != nil {
			quality, _ = l.current(iedID, pyv.Get(target, "quality_ref"))
		}
		stamp := any(int64(0))
		if index[pyv.GetStr(target, "timestamp_ref")] != nil {
			stamp, _ = l.current(iedID, pyv.Get(target, "timestamp_ref"))
		}
		value, _ := l.current(iedID, targetRef)
		l.deps.Events.Emit("signal", pyv.O("ied_id", iedID, "ref", targetRef, "value", value, "source", source,
			"quality", quality, "source_timestamp", stamp, "updates", updates, "live_state", "fresh"))
	}
}

// companionOwner finds the data attribute whose quality or time point companion is.
func companionOwner(index map[string]*pyjson.Object, companion *pyjson.Object) *pyjson.Object {
	ref := pyv.GetStr(companion, "ref")
	base := ref
	if cut := strings.LastIndex(ref, "."); cut >= 0 {
		base = ref[:cut]
	}
	for _, name := range []string{"stVal", "mag.f", "mag.i", "general", "cVal.mag.f", "setVal"} {
		candidate := index[base+"."+name]
		if candidate != nil && (pyv.Equal(pyv.Get(candidate, "quality_ref"), ref) ||
			pyv.Equal(pyv.Get(candidate, "timestamp_ref"), ref)) {
			return candidate
		}
	}
	return nil
}

// MarkStale keeps the last values visible but never presents them as fresh.
func (l *LiveValues) MarkStale(iedID, reason string) {
	l.deps.Values.MarkStale(iedID)
	l.deps.Events.Emit("live_stale", pyv.O("ied_id", iedID, "message", reason))
}
