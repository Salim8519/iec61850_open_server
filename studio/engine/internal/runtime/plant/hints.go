package plant

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Switching hints (domain/interlock_hints.py): advice only, never a block.
var (
	passThrough = map[string]bool{"JUNCTION": true, "CT": true}
	powered     = map[string]bool{"live": true, "maybe_live": true, "earth_fault": true}
)

func hint(code string, related []any, text string) *pyjson.Object {
	if related == nil {
		related = []any{}
	}
	return pyval.Obj("code", code, "related", related, "text", text)
}

func (s *Sections) neighbours() map[string][]string {
	links := map[string][]string{}
	for _, wire := range s.Wires {
		a, b := wire.Ends[0], wire.Ends[1]
		links[a] = append(links[a], b)
		links[b] = append(links[b], a)
	}
	return links
}

func (s *Sections) switchesNear(links map[string][]string, id string) []string {
	var found []string
	seen := map[string]bool{}
	queue := []string{}
	for _, point := range s.Points(id) {
		if !seen[point] {
			seen[point] = true
			queue = append(queue, point)
		}
	}
	for len(queue) > 0 {
		current := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		for _, point := range links[current] {
			if seen[point] {
				continue
			}
			seen[point] = true
			owner := strings.SplitN(point, ":", 2)[0]
			kind := s.byID[owner].Kind
			if passThrough[kind] {
				queue = append(queue, point)
			} else if Switches[kind] && owner != id && !contains(found, owner) {
				found = append(found, owner)
			}
		}
	}
	return found
}

func contains(items []string, wanted string) bool {
	for _, item := range items {
		if item == wanted {
			return true
		}
	}
	return false
}

func (s *Sections) label(id string) string { return pyval.Str(s.byID[id].Data, "label") }

func (s *Sections) breakerHints(links map[string][]string, id string) []any {
	hints := []any{}
	for _, other := range s.switchesNear(links, id) {
		if s.byID[other].Kind != "CB" {
			continue
		}
		name, position := s.label(other), s.Position(other)
		if position == closedPosition {
			hints = append(hints, hint("load", []any{other}, "Open "+name+" first: a disconnector must not switch load current."))
		} else if position != openPosition {
			hints = append(hints, hint("unknown", []any{other}, "The position of "+name+" is unknown. Check that it is open first."))
		}
	}
	return hints
}

func (s *Sections) closingHints(id string) []any {
	var sides []string
	distinct := map[string]bool{}
	for _, point := range s.Points(id) {
		state := s.State(point)
		sides = append(sides, state)
		distinct[state] = true
	}
	if s.Position(id) == closedPosition || len(distinct) < 2 {
		return []any{}
	}
	anyPowered := false
	for _, state := range sides {
		anyPowered = anyPowered || powered[state]
	}
	if anyPowered && distinct["earthed"] {
		return []any{hint("earth", nil, "One side is earthed and the other side has power. Closing causes an earth fault. "+
			"Open the earth switch first.")}
	}
	return []any{}
}

// SwitchingHints is switching_hints: {node: {"open": [...], "close": [...]}} for switches that need one.
func (s *Sections) SwitchingHints() *pyjson.Object {
	links := s.neighbours()
	result := pyjson.NewObject()
	for _, node := range s.Nodes {
		if !Switches[node.Kind] {
			continue
		}
		points := s.Points(node.ID)
		wiredAny := false
		for _, point := range points {
			_, wired := links[point]
			wiredAny = wiredAny || wired
		}
		if !wiredAny {
			continue
		}
		open, closing := []any{}, []any{}
		if node.Kind == "DS" {
			open = s.breakerHints(links, node.ID)
			closing = s.breakerHints(links, node.ID)
		}
		if node.Kind == "ES" {
			hot := false
			for _, point := range points {
				if _, wired := links[point]; wired && powered[s.State(point)] {
					hot = true
				}
			}
			if s.Position(node.ID) != closedPosition && hot {
				closing = append(closing, hint("earth", nil, "This part may have power. Closing the earth switch causes an earth fault. "+
					"Open the switches that feed it first."))
			}
		} else {
			closing = append(closing, s.closingHints(node.ID)...)
		}
		if len(open) > 0 || len(closing) > 0 {
			result.Set(node.ID, pyval.Obj("open", open, "close", closing))
		}
	}
	return result
}
