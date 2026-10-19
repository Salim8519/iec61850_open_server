package plant

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Energisation of the drawing (domain/energization.py): which parts are live, dead or earthed.
var (
	joining = map[string]bool{"JUNCTION": true, "CT": true, "INFEED": true}
	power   = map[string]bool{"CB": true, "DS": true, "ES": true, "JUNCTION": true, "CT": true, "INFEED": true, "VT": true}
	order   = []string{"earth_fault", "live", "maybe_live", "earthed", "dead"}
)

const (
	closedPosition = int64(2)
	openPosition   = int64(1)
)

type sets map[string]string

func (s sets) find(item string) string {
	if _, ok := s[item]; !ok {
		s[item] = item
	}
	for s[item] != item {
		s[item] = s[s[item]]
		item = s[item]
	}
	return item
}

func (s sets) join(a, b string) { s[s.find(a)] = s.find(b) }

// Node is one diagram node: id and data.
type Node struct {
	ID   string
	Data *pyjson.Object
	Kind string
}

// Wire is one diagram edge between two electrical points.
type Wire struct {
	ID   string
	Ends [2]string
}

// Sections are the electrical sections of a diagram for one set of switch positions.
type Sections struct {
	Nodes     []Node
	byID      map[string]Node
	positions map[string]int64 // 0 = unknown
	certain   sets
	possible  sets
	Wires     []Wire
	live      map[string]bool
	maybe     map[string]bool
	earthed   map[string]bool
}

func terminal(node Node, handle any, fallback string) string {
	if joining[node.Kind] {
		return node.ID
	}
	side, _ := handle.(string)
	if side == "" {
		side = fallback
	}
	if side == "north" || side == "west" {
		return node.ID + ":a"
	}
	return node.ID + ":b"
}

// NewSections is Sections(diagram, positions); positions maps node id to evaluated state.
func NewSections(diagram *pyjson.Object, positions map[string]any) *Sections {
	s := &Sections{byID: map[string]Node{}, positions: map[string]int64{}, certain: sets{}, possible: sets{},
		live: map[string]bool{}, maybe: map[string]bool{}, earthed: map[string]bool{}}
	for _, item := range pyval.List(diagram, "nodes") {
		object := item.(*pyjson.Object)
		data := pyval.Map(object, "data")
		node := Node{pyval.Str(object, "id"), data, pyval.Str(data, "kind")}
		if _, seen := s.byID[node.ID]; !seen {
			s.Nodes = append(s.Nodes, node)
		} else {
			for i := range s.Nodes {
				if s.Nodes[i].ID == node.ID {
					s.Nodes[i] = node
				}
			}
		}
		s.byID[node.ID] = node
	}
	for _, node := range s.Nodes {
		if Switches[node.Kind] {
			value := positions[node.ID]
			if pyval.IsInt(value) && (pyval.Equal(value, openPosition) || pyval.Equal(value, closedPosition)) {
				s.positions[node.ID], _ = value.(int64)
			} else if pyval.Equal(value, closedPosition) || pyval.Equal(value, openPosition) {
				f, _ := pyval.Float(value) // True == 1 and 2.0 == 2 in Python's `in (1, 2)`
				s.positions[node.ID] = int64(f)
			} else {
				s.positions[node.ID] = 0
			}
		}
	}
	for _, item := range pyval.List(diagram, "edges") {
		edge := item.(*pyjson.Object)
		source, okSource := s.byID[pyval.Str(edge, "source")]
		target, okTarget := s.byID[pyval.Str(edge, "target")]
		if !okSource || !okTarget || !power[source.Kind] || !power[target.Kind] {
			continue
		}
		ends := [2]string{terminal(source, pyval.Get(edge, "sourceHandle"), "east"),
			terminal(target, pyval.Get(edge, "targetHandle"), "north")}
		s.addWire(pyval.Str(edge, "id"), ends)
		s.certain.join(ends[0], ends[1])
		s.possible.join(ends[0], ends[1])
	}
	for _, node := range s.Nodes {
		position, isSwitch := s.positions[node.ID]
		if !isSwitch || node.Kind == "ES" {
			continue
		}
		if position == closedPosition {
			s.certain.join(node.ID+":a", node.ID+":b")
		}
		if position != openPosition {
			s.possible.join(node.ID+":a", node.ID+":b")
		}
	}
	for _, id := range s.feeds() {
		s.live[s.certain.find(id)] = true
		s.maybe[s.possible.find(id)] = true
	}
	for _, node := range s.Nodes {
		if s.positions[node.ID] == closedPosition && node.Kind == "ES" {
			s.earthed[s.certain.find(node.ID+":a")] = true
			s.earthed[s.certain.find(node.ID+":b")] = true
		}
	}
	return s
}

func (s *Sections) addWire(id string, ends [2]string) {
	for i := range s.Wires {
		if s.Wires[i].ID == id {
			s.Wires[i].Ends = ends // a repeated id keeps its first place, last value (dict)
			return
		}
	}
	s.Wires = append(s.Wires, Wire{id, ends})
}

func (s *Sections) feeds() []string {
	var ids []string
	for _, node := range s.Nodes {
		if node.Kind == "INFEED" && getOr(node.Data, "infeed_state", "live") != "dead" {
			ids = append(ids, node.ID)
		}
	}
	return ids
}

// State is Sections.state of one electrical point.
func (s *Sections) State(point string) string {
	root := s.certain.find(point)
	live, earthed := s.live[root], s.earthed[root]
	switch {
	case live && earthed:
		return "earth_fault"
	case live:
		return "live"
	case earthed:
		return "earthed"
	case s.maybe[s.possible.find(point)]:
		return "maybe_live"
	}
	return "dead"
}

// Points are the electrical points of a power item: one, or its two sides.
func (s *Sections) Points(id string) []string {
	if joining[s.byID[id].Kind] {
		return []string{id}
	}
	return []string{id + ":a", id + ":b"}
}

// Position is a switch's position (1, 2) or 0 when unknown.
func (s *Sections) Position(id string) int64 { return s.positions[id] }

// Describe returns {"nodes": {id: [states]}, "edges": {id: state}} in Python's order.
func (s *Sections) Describe() (*pyjson.Object, *pyjson.Object) {
	edges := pyjson.NewObject()
	wired := map[string]bool{}
	for _, wire := range s.Wires {
		edges.Set(wire.ID, s.State(wire.Ends[0]))
		wired[wire.Ends[0]], wired[wire.Ends[1]] = true, true
	}
	nodes := pyjson.NewObject()
	for _, node := range s.Nodes {
		if !power[node.Kind] {
			continue
		}
		var points []string
		for _, point := range s.Points(node.ID) {
			if wired[point] {
				points = append(points, point)
			}
		}
		if len(points) == 0 && node.Kind != "INFEED" {
			continue
		}
		if len(points) == 0 {
			points = s.Points(node.ID)
		}
		states := map[string]bool{}
		for _, point := range points {
			states[s.State(point)] = true
		}
		var list []any
		for _, state := range order {
			if states[state] {
				list = append(list, state)
			}
		}
		if list == nil {
			list = []any{}
		}
		nodes.Set(node.ID, list)
	}
	return nodes, edges
}
