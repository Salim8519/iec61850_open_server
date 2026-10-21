package licensing

import (
	"sort"
	"strings"

	"studio-engine/internal/licensing/pytext"
)

// RevokedIDs remembers which licenses the server cancelled, so a renamed certificate can
// never come back (licensing/revoked_ids.py, ADR-0097). When the server's signed verdict is
// "revoked" (or "moved"), the license ID goes into three places beside the trial clock: the
// dot-file ".revoked_ids", the string "RevokedLicenses" under HKCU\Software\ScadaBench and
// the "revoked_ids" field of license_state.json. Each copy is the IDs sorted and joined by
// ",". Evaluation refuses any certificate whose ID is in ANY copy and restores the copies
// that were deleted from the ones that remain. Nothing here ever removes an ID.
// Native twin: license_clock.cpp (read_revoked_ids).
type RevokedIDs struct{ Value ClockValue }

// NewRevokedIDs returns the cancelled-ID record kept under home.
func NewRevokedIDs(home string, registry Registry) RevokedIDs {
	return RevokedIDs{Value: NewClockValue("revoked_ids", home, registry)}
}

// parseIDs is RevokedIds.parse: the non-empty, trimmed parts of a comma-separated copy.
func parseIDs(text string) []string {
	var ids []string
	for _, part := range strings.Split(text, ",") {
		if id := pytext.Strip(part); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// All is the union of every copy that can be read.
func (r RevokedIDs) All() map[string]bool {
	found := map[string]bool{}
	for _, copy := range r.Value.ReadAll() {
		for _, id := range parseIDs(copy) {
			found[id] = true
		}
	}
	return found
}

// Contains reports whether licenseID (any decoded JSON value; only text can match) is cancelled.
func (r RevokedIDs) Contains(licenseID any) bool {
	text, isText := licenseID.(string)
	return isText && r.All()[pytext.Strip(text)]
}

func joinIDs(ids map[string]bool) string {
	names := make([]string, 0, len(ids))
	for id := range ids {
		names = append(names, id)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

// Add remembers a cancelled license everywhere (an ID that is not text is ignored).
func (r RevokedIDs) Add(licenseID any) {
	text, isText := licenseID.(string)
	if !isText || len(parseIDs(text)) == 0 || strings.Contains(text, ",") {
		return
	}
	ids := r.All()
	ids[pytext.Strip(text)] = true
	r.Value.Write(joinIDs(ids))
}

// Restore rewrites any copy that is missing, empty or behind the union of the others.
func (r RevokedIDs) Restore() {
	ids := r.All()
	if len(ids) == 0 {
		return
	}
	text := joinIDs(ids)
	copies := r.Value.ReadAll()
	stale := len(copies) < r.Value.Places()
	for _, copy := range copies {
		stale = stale || copy != text
	}
	if stale {
		r.Value.Write(text)
	}
}
