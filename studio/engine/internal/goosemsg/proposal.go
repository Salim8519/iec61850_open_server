package goosemsg

import (
	"math"

	"studio-engine/internal/pyjson"
)

// IEDKind is the SLD component that stands for one whole IED (ADR-0112).
const IEDKind = "IED"

// The IED card of the SLD is drawn 200 px wide (ied-card.css); a new row keeps a 40 px gap
// between cards and 60 px above the drawing's top, on the 20 px grid.
const (
	CardWidth  = 200
	CardHeight = 120
	CardGap    = 40
	RowGap     = 60
	grid       = 20
)

// Placement is one IED component "Add all GOOSE IEDs" proposes.
type Placement struct {
	IEDID, Name string
	X, Y        float64
}

// UsesGoose says whether the stored IED sends or receives GOOSE in its files.
func UsesGoose(ied *pyjson.Object) bool {
	goose := object(field(ied, "goose"))
	return len(list(field(goose, "publishes"))) > 0 || len(list(field(goose, "subscribes"))) > 0
}

func snap(value float64) float64 { return math.Round(value/grid) * grid }

// Propose places one IED component for every project IED that sends or receives GOOSE and has no
// component on the drawing yet, in project order, in one evenly spaced row: above the drawing's top
// (spread over the drawing's width when it is wider than the row), or, once IED components exist,
// continuing their row to the right so adding them one by one never builds a staircase.
func Propose(nodes []any, ieds []*pyjson.Object) []Placement {
	shown := map[string]bool{}
	drawing, row := newExtent(), newExtent()
	for _, item := range nodes {
		node := object(item)
		data := object(field(node, "data"))
		position := object(field(node, "position"))
		x, okX := float(field(position, "x"))
		y, okY := float(field(position, "y"))
		component := text(data, "kind") == IEDKind
		if component && text(data, "ied_id") != "" {
			shown[text(data, "ied_id")] = true
		}
		if okX && okY {
			drawing.add(x, y)
			if component {
				row.add(x, y)
			}
		}
	}
	var chosen []*pyjson.Object
	for _, ied := range ieds {
		if UsesGoose(ied) && !shown[text(ied, "id")] {
			chosen = append(chosen, ied)
		}
	}
	if len(chosen) == 0 {
		return nil
	}
	pitch := float64(CardWidth + CardGap)
	var startX, y float64
	switch {
	case !row.empty(): // continue the row of the components already drawn
		startX, y = row.maxX+pitch, row.minY
	case !drawing.empty():
		startX, y = drawing.minX, drawing.minY-RowGap-CardHeight
		if len(chosen) > 1 {
			pitch = math.Max(pitch, snap((drawing.maxX-drawing.minX)/float64(len(chosen)-1)))
		}
	}
	result := make([]Placement, len(chosen))
	for index, ied := range chosen {
		result[index] = Placement{IEDID: text(ied, "id"), Name: text(ied, "name"),
			X: snap(startX + float64(index)*pitch), Y: snap(y)}
	}
	return result
}

// extent is the top-left corners seen: leftmost, rightmost and topmost.
type extent struct{ minX, maxX, minY float64 }

func newExtent() *extent { return &extent{math.Inf(1), math.Inf(-1), math.Inf(1)} }

func (e *extent) add(x, y float64) {
	e.minX, e.maxX, e.minY = math.Min(e.minX, x), math.Max(e.maxX, x), math.Min(e.minY, y)
}

func (e *extent) empty() bool { return math.IsInf(e.minX, 1) }
