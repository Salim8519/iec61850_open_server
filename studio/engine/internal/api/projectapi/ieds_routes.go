package projectapi

import (
	"net/http"

	"studio-engine/internal/pyjson"
)

// lockCycle holds coordinator.cycle_lock for IED removal; unlock is a no-op without one.
func (a *Area) lockCycle() (unlock func()) {
	if a.deps.Runtime == nil {
		return func() {}
	}
	lock := a.deps.Runtime.CycleLock()
	if lock == nil {
		return func() {}
	}
	lock.Lock()
	return lock.Unlock
}

// removalPreview is GET /ieds/{key}/removal-preview.
func (a *Area) removalPreview(w http.ResponseWriter, r *http.Request) error {
	if err := a.requireStopped(); err != nil {
		return err
	}
	preview, err := a.deps.Projects.RemovalPreview(r.PathValue("key"))
	if err != nil {
		return err
	}
	writeJSON(w, 200, preview)
	return nil
}

// removeIED is DELETE /ieds/{key} {"confirm": true}.
func (a *Area) removeIED(w http.ResponseWriter, r *http.Request) error {
	defer a.lockCycle()()
	if err := a.requireStopped(); err != nil {
		return err
	}
	raw, _ := getJSON(r, true)
	if object(raw) == nil && truthy(raw) {
		return errNotDict
	}
	if !confirmed(raw) {
		writeJSON(w, 400, pyjson.NewObject().Set("error", "Confirm deletion after reviewing affected bindings and automation"))
		return nil
	}
	result, err := a.deps.Projects.RemoveIED(r.PathValue("key"))
	if err != nil {
		return err
	}
	writeJSON(w, 200, result)
	return nil
}

// truthy is Python's bool() of decoded JSON.
func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	}
	return true
}

// confirmed is `data.get("confirm") is True` for a dict.
func confirmed(raw any) bool {
	data := object(raw)
	if data == nil {
		return false
	}
	value, _ := data.Get("confirm")
	flag, ok := value.(bool)
	return ok && flag
}

// bulkPreview is POST /ieds/removal-preview {"ids"}.
func (a *Area) bulkPreview(w http.ResponseWriter, r *http.Request) error {
	defer a.lockCycle()()
	if err := a.requireStopped(); err != nil {
		return err
	}
	raw, err := getJSON(r, false)
	if err != nil {
		return err
	}
	data := object(raw)
	if data == nil {
		return errNotDict
	}
	preview, err := a.deps.Projects.BulkPreview(get(data, "ids"))
	if err != nil {
		return err
	}
	writeJSON(w, 200, preview)
	return nil
}

// bulkRemove is POST /ieds/remove {"ids", "fingerprint", "confirm": true}; it emits
// ieds_removed after the removal commits.
func (a *Area) bulkRemove(w http.ResponseWriter, r *http.Request) error {
	defer a.lockCycle()()
	if err := a.requireStopped(); err != nil {
		return err
	}
	raw, err := getJSON(r, false)
	if err != nil {
		return err
	}
	data := object(raw)
	if data == nil {
		return errNotDict
	}
	if !confirmed(data) {
		writeJSON(w, 400, pyjson.NewObject().Set("error", "Confirm the deletion preview first"))
		return nil
	}
	ids := get(data, "ids")
	result, err := a.deps.Projects.BulkRemove(ids, get(data, "fingerprint"))
	if err != nil {
		return err
	}
	if a.deps.Runtime != nil {
		list, _ := ids.([]any)
		a.deps.Runtime.Emit("ieds_removed", pyjson.NewObject().Set("ied_ids", ids).Set("count", len(list)))
	}
	writeJSON(w, 200, result)
	return nil
}
