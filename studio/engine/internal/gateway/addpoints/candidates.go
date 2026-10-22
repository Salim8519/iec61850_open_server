package addpoints

import (
	"sort"
	"strings"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// MaxCandidates is how many fitting signals one answer lists; a search narrows the rest.
const MaxCandidates = 300

// Candidate is one IED signal or controllable object that fits a gateway type.
type Candidate struct {
	Ref         string `json:"ref"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ValueKind   string `json:"value_kind"`
	FC          string `json:"fc"`
	CDC         string `json:"cdc,omitempty"`
	StatusRef   string `json:"status_ref,omitempty"`
	Taken       bool   `json:"taken"`
	// Suggested is what Propose from IEDs would pick for this type (listed first).
	Suggested bool `json:"suggested"`
}

// Fits reports whether a signal can be served as a status or measured point of kind: a
// status (ST) or measured (MX) value whose IEC 61850 type the kind carries unchanged.
func Fits(signal *pyjson.Object, kind string) bool {
	fc := text(signal, "fc")
	if gateway.IsCommandKind(kind) || (fc != "ST" && fc != "MX") {
		return false
	}
	return contains(gateway.AllowedKinds(text(signal, "kind")), kind)
}

// CommandFits reports whether a controllable object is operated by a command of kind: DPC by
// a double command, BSC/ISC by a step, APC/INC by a set point (ADR-0107), SPC by a single command.
func CommandFits(control *pyjson.Object, kind string) bool {
	switch text(control, "cdc") {
	case "DPC":
		return kind == "command_double"
	case "BSC", "ISC":
		return kind == gateway.KindStep
	case "APC", "INC":
		return kind == gateway.KindSetpoint
	case "ENC", "BAC":
		return false // no gateway kind carries a list choice or a binary step yet
	}
	return kind == "command_single" // SPC and any other class, as Propose maps them
}

// Candidates lists what of one IED fits kind, matching every word of query in the reference
// or description. taken are point ids already in the list. It answers at most MaxCandidates
// and the total that matched.
func Candidates(ied *IED, kind, query string, taken map[string]bool) ([]Candidate, int) {
	words := strings.Fields(strings.ToLower(query))
	found := []Candidate{}
	if gateway.IsCommandKind(kind) {
		for _, control := range ied.controls {
			if !CommandFits(control, kind) {
				continue
			}
			ref := text(control, "ref")
			status, _ := ied.StatusOf(control)
			found = append(found, ied.candidate(ref, kind, "", "CO", text(control, "cdc"), status, taken, true))
		}
	} else {
		for _, signal := range ied.signals {
			if Fits(signal, kind) {
				suggested := gateway.SmartKind(signal) == kind
				found = append(found, ied.candidate(text(signal, "ref"), kind, text(signal, "kind"), text(signal, "fc"),
					"", "", taken, suggested))
			}
		}
	}
	matched := found[:0]
	for _, item := range found {
		if matches(item, words) {
			matched = append(matched, item)
		}
	}
	sort.SliceStable(matched, func(i, j int) bool { return matched[i].Suggested && !matched[j].Suggested })
	total := len(matched)
	if total > MaxCandidates {
		matched = matched[:MaxCandidates]
	}
	return matched, total
}

func (ied *IED) candidate(ref, kind, valueKind, fc, cdc, status string, taken map[string]bool, suggested bool) Candidate {
	description := ied.Description(ref)
	name := description
	if name == "" {
		name = ShortName(ref)
	}
	return Candidate{Ref: ref, Name: name, Description: description, ValueKind: valueKind, FC: fc, CDC: cdc,
		StatusRef: status, Taken: taken[gateway.PointID(ied.ID, ref, kind)], Suggested: suggested}
}

func matches(item Candidate, words []string) bool {
	haystack := strings.ToLower(item.Ref + " " + item.Description)
	for _, word := range words {
		if !strings.Contains(haystack, word) {
			return false
		}
	}
	return true
}

// FittingRefs are the references of every signal or command of the IED that fits kind.
func FittingRefs(ied *IED, kind string) []string {
	refs := []string{}
	if gateway.IsCommandKind(kind) {
		for _, control := range ied.controls {
			if CommandFits(control, kind) {
				refs = append(refs, text(control, "ref"))
			}
		}
		return refs
	}
	for _, signal := range ied.signals {
		if Fits(signal, kind) {
			refs = append(refs, text(signal, "ref"))
		}
	}
	return refs
}

// TakenIDs are the point ids of a list.
func TakenIDs(points []any) map[string]bool {
	taken := map[string]bool{}
	for _, raw := range points {
		if point, ok := raw.(*pyjson.Object); ok {
			if id := text(point, "id"); id != "" {
				taken[id] = true
			} else if text(point, "ied") != "" {
				taken[gateway.PointID(text(point, "ied"), text(point, "ref"), text(point, "kind"))] = true
			}
		}
	}
	return taken
}
