package runtimeapi

import (
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// sendGoose is POST /equipment/<node_id>/goose (Go-only, ADR-0112): an equipment with a GOOSE
// link sends {action, value?}; an IED component sends {gocb_ref, ref, value}. The answer holds
// the values the simulated IED then held and the correlation id of the logged events.
func (a Area) sendGoose(r *http.Request) (any, int, error) {
	data, ok := getJSONSilent(r).(*pyjson.Object)
	if !ok {
		return nil, 0, apperr.ValueError("A GOOSE send must be a JSON object")
	}
	result, err := a.Runtime.SendGoose(r.PathValue("node_id"), data)
	if err != nil {
		return nil, 0, err
	}
	return result, http.StatusOK, nil
}
