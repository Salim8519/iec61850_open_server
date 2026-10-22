package runtimeapi

import (
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// selectEquipment is POST /equipment/<node_id>/select: {"action":"open"|"close","options":{...}}.
func (a Area) selectEquipment(r *http.Request) (any, int, error) {
	data, ok := getJSONSilent(r).(*pyjson.Object)
	if !ok {
		return nil, 0, apperr.ValueError("Equipment selection must be a JSON object")
	}
	action := ""
	if value, present := data.Get("action"); present && value != nil {
		action = text(value)
	}
	result, err := a.Runtime.EquipmentSelect(r.PathValue("node_id"), action, pyval.Get(data, "options"))
	if err != nil {
		return nil, 0, err
	}
	return result, http.StatusOK, nil
}

// cancelEquipment is POST /equipment/<node_id>/cancel (any body).
func (a Area) cancelEquipment(r *http.Request) (any, int, error) {
	result, err := a.Runtime.EquipmentCancel(r.PathValue("node_id"))
	if err != nil {
		return nil, 0, err
	}
	return result, http.StatusOK, nil
}
