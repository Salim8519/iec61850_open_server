package sld

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

const (
	cardW, cardH, dot = 108.0, 130.0, 40.0
	gapX, gapY        = 40.0, 80.0
	grid              = 20.0
	minScale          = 90
	maxScale          = 300
	maxNodes          = 2000
	maxWires          = 5000
	notDrawn          = "ScadaBench has no symbol for this equipment type yet"
)

// KindOf is kind_of: the diagram kind of an equipment, "" when not drawn.
func KindOf(item *scl.Equipment) string {
	kind := map[string]string{"CBR": "CB", "DIS": "DS", "CTR": "CT", "VTR": "VT"}[item.Type]
	if kind == "DS" {
		for _, path := range item.Terminals {
			if path == scl.Grounded {
				return "ES"
			}
		}
	}
	return kind
}

func nodeID(sourceID, text string) string {
	digest := sha256.Sum256([]byte(sourceID + "/" + text))
	return "scl-" + hex.EncodeToString(digest[:])[:16]
}

func snap(value float64) int64 { return pyRound(value/grid) * grid }

// scaleFor is _scale: the smallest scale at which no two cards overlap, 0 for None.
func scaleFor(points []point) int64 {
	needed := float64(minScale)
	items := append([]point(nil), points...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i][0] < items[j][0] || items[i][0] == items[j][0] && items[i][1] < items[j][1]
	})
	for index, a := range items {
		for _, b := range items[index+1:] {
			dx, dy := math.Abs(a[0]-b[0]), math.Abs(a[1]-b[1])
			fitX, fitY := math.Inf(1), math.Inf(1)
			if dx != 0 {
				fitX = (cardW + gapX) / dx
			}
			if dy != 0 {
				fitY = (cardH + gapY) / dy
			}
			if fit := pyMin(fitX, fitY); fit > needed { // max(needed, fit)
				needed = fit
			}
		}
	}
	if needed > maxScale {
		return 0
	}
	return int64(-floorDiv(-needed, 10) * 10)
}

// Built is build_diagram's result.
type Built struct {
	Nodes, Edges []any
	Skipped      []any
	Equipment    map[string]*scl.Equipment // node id -> equipment
	EquipmentIDs []string                  // dict order of Equipment
	Diagnostics  []string
	Layout       string
}

// BuildDiagram is build_diagram(topology, source_id); bindings are left empty. The error
// is the ValueError for a station beyond the diagram limits.
func BuildDiagram(topology *scl.Topology, sourceID string) (*Built, error) {
	built := &Built{Equipment: map[string]*scl.Equipment{}, Diagnostics: append([]string(nil), topology.Diagnostics...)}
	var drawn []*scl.Equipment
	built.Skipped = []any{}
	for _, item := range topology.Equipment {
		if KindOf(item) != "" {
			drawn = append(drawn, item)
		} else {
			built.Skipped = append(built.Skipped, skipped(item.Path, item.Type))
		}
	}
	for _, item := range topology.Other {
		built.Skipped = append(built.Skipped, skipped(item.Path, item.Element))
	}
	members := newOrdered()
	for _, item := range drawn {
		for _, path := range item.Terminals {
			if path != "" && path != scl.Grounded {
				members.add(path, item.Path)
			}
		}
	}
	layout := &layoutState{sourceID: sourceID, nodes: topology.Nodes, members: members, ids: map[string]string{},
		centres: map[string]point{}}
	var points map[string]point
	built.Layout, points, layout.busY, layout.scale = placement(drawn, members, topology.Nodes, &built.Diagnostics)
	for _, item := range drawn {
		layout.ids[item.Path] = nodeID(sourceID, item.Path)
	}
	for _, item := range drawn {
		x, y := snap(float64(points[item.Path][0]*float64(layout.scale))), snap(float64(points[item.Path][1]*float64(layout.scale)))
		id := layout.ids[item.Path]
		layout.centres[id] = point{float64(x) + cardW/2, float64(y) + cardH/2}
		label := item.Name
		if label == "" {
			label = KindOf(item)
		}
		built.Nodes = append(built.Nodes, equipmentNode(id, x, y, label, KindOf(item)))
	}
	if err := layout.wires(built); err != nil {
		return nil, err
	}
	for _, item := range drawn {
		id := layout.ids[item.Path]
		if _, seen := built.Equipment[id]; !seen {
			built.EquipmentIDs = append(built.EquipmentIDs, id)
		}
		built.Equipment[id] = item
	}
	return built, nil
}

func skipped(path, kind string) *pyjson.Object {
	return pyjson.NewObject().Set("path", path).Set("type", kind).Set("reason", notDrawn)
}

func equipmentNode(id string, x, y any, label, kind string) *pyjson.Object {
	return pyjson.NewObject().Set("id", id).Set("type", "equipment").
		Set("position", pyjson.NewObject().Set("x", x).Set("y", y)).
		Set("data", pyjson.NewObject().Set("label", firstRunes(label, 100)).Set("kind", kind).Set("rotation", int64(0)))
}

// firstRunes is text[:n] on a Python str.
func firstRunes(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n])
}

// placement is _placement: esld coordinates when every drawn item has them and they fit,
// otherwise the automatic grid.
func placement(drawn []*scl.Equipment, members *ordered, nodes map[string]*scl.ConnectivityNode,
	diagnostics *[]string) (string, map[string]point, map[string]float64, int64) {
	esld := map[string]point{} // a dict: a repeated path keeps its last coordinates
	for _, item := range drawn {
		if item.ESLD != nil {
			esld[item.Path] = point(*item.ESLD)
		}
	}
	order := make([]point, 0, len(esld)) // sorted by scaleFor; equal tuples are identical
	for _, value := range esld {
		order = append(order, value)
	}
	if len(drawn) > 0 && len(esld) == len(drawn) {
		if scale := scaleFor(order); scale != 0 {
			busY := map[string]float64{}
			for _, path := range members.keys {
				if value := nodes[path].BusY; value != nil {
					busY[path] = *value
				}
			}
			return "esld", esld, busY, scale
		}
		*diagnostics = append(*diagnostics, "Drawing coordinates in the SCD overlap; the automatic layout was used")
	} else if len(esld) > 0 {
		*diagnostics = append(*diagnostics, "Only some equipment has drawing coordinates; the automatic layout was used")
	}
	grid, busY := autoPositions(drawn, members, nodes)
	return "automatic", grid, busY, minScale
}

// Limits is the ValueError text of a station beyond the diagram limits.
func limits(nodes, wires int) error {
	return fmt.Errorf("This substation needs %d symbols and %d wires; the diagram limit is %d symbols and %d wires",
		nodes, wires, maxNodes, maxWires)
}
