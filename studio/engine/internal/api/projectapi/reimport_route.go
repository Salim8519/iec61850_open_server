package projectapi

import (
	"net/http"

	"studio-engine/internal/apperr"
)

// reimport is POST /ieds/{key}/reimport (multipart: file, apply "1" to change the project): a newer
// CID of the IED replaces its model while its id, settings and the links it still has stay
// (Go-only, ADR-0148). Without apply it only answers what would change.
func (a *Area) reimport(w http.ResponseWriter, r *http.Request) error {
	if err := a.requireStopped(); err != nil {
		return err
	}
	parsed, err := parseForm(r)
	if err != nil {
		return err
	}
	source, filename, err := parsed.file("file")
	if err != nil {
		return err
	}
	if len(source) == 0 {
		return apperr.New("Choose the new CID file of this IED")
	}
	if filename == "" {
		filename = "import.cid"
	}
	answer, err := a.deps.Projects.Reimport(r.PathValue("key"), source, filename, parsed.values["apply"] == "1")
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, answer)
	return nil
}
