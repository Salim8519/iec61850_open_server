package liveapi

import (
	"net/http"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
)

// operate is POST /live/{key}/operate {ref, value, confirm}: one confirmed command to a real
// IED from Devices, only while its Allow commands is on (Go-only, ADR-0138).
func (s *Service) operate(w http.ResponseWriter, r *http.Request) {
	if s.Commands == nil {
		writeError(w, apperr.RuntimeError("Commands are not available in this copy of ScadaBench"))
		return
	}
	request, err := body(r)
	if err != nil {
		writeError(w, err)
		return
	}
	ref, _ := pyv.Get(request, "ref").(string)
	answer, err := s.Commands.Operate(r.PathValue("key"), ref, pyv.Get(request, "value"),
		pyv.Get(request, "confirm") == true)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, answer)
}
