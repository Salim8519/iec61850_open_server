package runtimeapi

import (
	"net/http"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// fields reads request.get_json() and the named keys like data["key"] in order.
func fields(r *http.Request, keys ...string) ([]any, any, error) {
	data, err := getJSON(r)
	if err != nil {
		return nil, nil, err
	}
	values := make([]any, len(keys))
	for i, key := range keys {
		if values[i], err = field(data, key); err != nil {
			return nil, nil, err
		}
	}
	return values, data, nil
}

// update is POST /signals/update: {"value": actual}.
func (a Area) update(r *http.Request) (any, int, error) {
	values, data, err := fields(r, "ied_id", "ref", "value")
	if err != nil {
		return nil, 0, err
	}
	quality, err := optional(data, "quality", nil)
	if err != nil {
		return nil, 0, err
	}
	actual, err := a.Runtime.Update(text(values[0]), text(values[1]), values[2], quality, "manual")
	if err != nil {
		return nil, 0, err
	}
	return pyval.Obj("value", actual), http.StatusOK, nil
}

// source is POST /signals/source.
func (a Area) source(r *http.Request) (any, int, error) {
	values, _, err := fields(r, "ied_id", "ref", "spec")
	if err != nil {
		return nil, 0, err
	}
	if err := a.Runtime.ConfigureSource(text(values[0]), text(values[1]), values[2]); err != nil {
		return nil, 0, err
	}
	return pyval.Obj("ok", true), http.StatusOK, nil
}

// pulse is POST /signals/pulse: {"id", "value"}.
func (a Area) pulse(r *http.Request) (any, int, error) {
	values, data, err := fields(r, "ied_id", "ref")
	if err != nil {
		return nil, 0, err
	}
	active, err := optional(data, "active_value", true)
	if err != nil {
		return nil, 0, err
	}
	duration, err := optional(data, "duration_ms", int64(1000))
	if err != nil {
		return nil, 0, err
	}
	result, err := a.Runtime.Pulse(text(values[0]), text(values[1]), active, duration)
	if err != nil {
		return nil, 0, err
	}
	return result, http.StatusOK, nil
}

// release is POST /signals/release-override.
func (a Area) release(r *http.Request) (any, int, error) {
	values, _, err := fields(r, "ied_id", "ref")
	if err != nil {
		return nil, 0, err
	}
	if err := a.Runtime.ReleaseOverride(text(values[0]), text(values[1])); err != nil {
		return nil, 0, err
	}
	return pyval.Obj("ok", true), http.StatusOK, nil
}

// controlOperate is POST /controls/operate: the probe's JSON result.
func (a Area) controlOperate(r *http.Request) (any, int, error) {
	values, data, err := fields(r, "ied_id")
	if err != nil {
		return nil, 0, err
	}
	if err := a.Runtime.KnownIed(text(values[0])); err != nil {
		return nil, 0, err
	}
	if typedRequest(data) { // Go-only (ADR-0104); {ied_id, ref, value: bool} stays the old path
		return a.typedOperate(data)
	}
	ref, err := field(data, "ref")
	if err != nil {
		return nil, 0, err
	}
	value, err := field(data, "value")
	if err != nil {
		return nil, 0, err
	}
	result, err := a.Runtime.OperateControl(text(values[0]), text(ref), value)
	if err != nil {
		return nil, 0, err
	}
	return result, http.StatusOK, nil
}

// testPanel is GET /project/test-panel: every entry with its signal and source state.
func (a Area) testPanel(*http.Request) (any, int, error) {
	items := []any{}
	for _, item := range pyval.List(a.Project.Current(), "test_panel") {
		entry, _ := item.(*pyjson.Object)
		iedID, ref := pyval.Str(entry, "ied_id"), pyval.Str(entry, "ref")
		view, err := a.Runtime.Signal(iedID, ref)
		if err != nil {
			return nil, 0, err
		}
		overridden, automatic := a.Runtime.SourceState(iedID, ref)
		row := pyjson.NewObject()
		for _, key := range entry.Keys() {
			value, _ := entry.Get(key)
			row.Set(key, value)
		}
		row.Set("signal", view).Set("overridden", overridden).Set("automatic", automatic)
		items = append(items, row)
	}
	return pyval.Obj("items", items), http.StatusOK, nil
}
