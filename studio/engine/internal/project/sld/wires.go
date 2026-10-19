package sld

import (
	"math"
	"sort"
	"strconv"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// layoutState carries build_diagram's locals into the wiring step.
type layoutState struct {
	sourceID string
	nodes    map[string]*scl.ConnectivityNode
	members  *ordered
	ids      map[string]string // equipment path -> node id
	centres  map[string]point  // node id -> centre
	busY     map[string]float64
	scale    int64
}

// wires draws every connectivity node: a direct wire between two members, otherwise
// junctions (one beside a lone member, a row for a tee or busbar) chained by wires.
func (l *layoutState) wires(built *Built) error {
	var edges []*pyjson.Object
	for _, path := range l.members.keys {
		owners := make([]string, 0, len(l.members.values[path]))
		for _, member := range l.members.values[path] {
			owners = append(owners, l.ids[member])
		}
		node := l.nodes[path]
		label := firstRunes(node.BayName+"/"+node.Name, 100)
		if len(owners) == 2 {
			edges = append(edges, l.wire(owners[0], owners[1]))
			continue
		}
		var junctions map[int64]point
		slots := map[string]int64{}
		if len(owners) == 1 {
			junctions, slots = l.single(path, owners[0])
		} else {
			height, onBus := l.busY[path]
			if onBus {
				height = float64(height * float64(l.scale))
			} else {
				height = l.gapMiddle(owners)
			}
			junctions = map[int64]point{}
			for _, owner := range owners {
				x := rowX(l.centres[owner], height)
				slots[owner] = x
				junctions[x] = point{float64(x), height}
			}
		}
		keys := map[int64]string{}
		xs := make([]int64, 0, len(junctions))
		for x := range junctions {
			xs = append(xs, x)
		}
		sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
		chain := make([]string, 0, len(xs))
		for _, x := range xs {
			centre := junctions[x]
			id := nodeID(l.sourceID, path+"#"+strconv.FormatInt(x, 10))
			keys[x] = id
			chain = append(chain, id)
			l.centres[id] = centre
			built.Nodes = append(built.Nodes, equipmentNode(id, centre[0]-dot/2, centre[1]-dot/2, label, "JUNCTION"))
		}
		for index := 1; index < len(chain); index++ {
			edges = append(edges, l.wire(chain[index-1], chain[index]))
		}
		for _, owner := range owners {
			edges = append(edges, l.wire(owner, keys[slots[owner]]))
		}
	}
	built.Edges = dedupe(edges)
	if len(built.Nodes) > maxNodes || len(built.Edges) > maxWires {
		return limits(len(built.Nodes), len(built.Edges))
	}
	return nil
}

// dedupe is list({edge["id"]: edge for edge in edges}.values()).
func dedupe(edges []*pyjson.Object) []any {
	position := map[string]int{}
	result := []any{}
	for _, edge := range edges {
		id, _ := edge.Get("id")
		if index, seen := position[id.(string)]; seen {
			result[index] = edge
			continue
		}
		position[id.(string)] = len(result)
		result = append(result, edge)
	}
	return result
}

// wire is _wire: vertical wires run south to north, horizontal ones east to west.
func (l *layoutState) wire(a, b string) *pyjson.Object {
	ca, cb := l.centres[a], l.centres[b]
	var ends [4]string
	if math.Abs(cb[1]-ca[1]) >= math.Abs(cb[0]-ca[0]) {
		upper, lower := a, b
		if !(ca[1] < cb[1] || ca[1] == cb[1] && ca[0] <= cb[0]) {
			upper, lower = b, a
		}
		ends = [4]string{upper, "south", lower, "north"}
	} else {
		left, right := a, b
		if !(ca[0] <= cb[0]) {
			left, right = b, a
		}
		ends = [4]string{left, "east", right, "west"}
	}
	first, second := a, b
	if second < first {
		first, second = second, first
	}
	return pyjson.NewObject().Set("id", nodeID(l.sourceID, "wire|"+first+"|"+second)).Set("source", ends[0]).
		Set("sourceHandle", ends[1]).Set("target", ends[2]).Set("targetHandle", ends[3])
}

// single is _single: a lone junction on the side away from the member's other wires.
func (l *layoutState) single(path, owner string) (map[int64]point, map[string]int64) {
	centre := l.centres[owner]
	var others []float64
	for _, node := range l.members.keys {
		paths := l.members.values[node]
		if node == path || !l.owns(paths, owner) {
			continue
		}
		for _, member := range paths {
			if l.ids[member] != owner {
				others = append(others, l.centres[l.ids[member]][1])
			}
		}
	}
	below := len(others) > 0 && pySum(others)/float64(len(others)) > centre[1]
	offset := cardH/2 + 50
	if below || len(others) == 0 {
		offset = -offset
	}
	x := pyRound(centre[0])
	return map[int64]point{x: {float64(x), centre[1] + offset}}, map[string]int64{owner: x}
}

func (l *layoutState) owns(paths []string, owner string) bool {
	for _, member := range paths {
		if l.ids[member] == owner {
			return true
		}
	}
	return false
}

// gapMiddle is _gap_middle: a tee without a drawn bar sits in the widest vertical gap.
func (l *layoutState) gapMiddle(owners []string) float64 {
	heights := make([]float64, len(owners))
	for index, owner := range owners {
		heights[index] = l.centres[owner][1]
	}
	sort.Float64s(heights)
	best, found := point{}, false
	for index := 1; index < len(heights); index++ {
		gap := point{heights[index] - heights[index-1], (heights[index-1] + heights[index]) / 2}
		if !found || gap[0] > best[0] || gap[0] == best[0] && gap[1] > best[1] {
			best, found = gap, true
		}
	}
	if found && best[0] > 0 {
		return best[1]
	}
	return heights[0] - cardH/2 - 40
}

// rowX is _row_x: a card the bar runs through gets its junction beside it.
func rowX(centre point, height float64) int64 {
	if math.Abs(height-centre[1]) < cardH/2+dot/2 {
		return pyRound(centre[0] + cardW/2 + 40)
	}
	return pyRound(centre[0])
}
