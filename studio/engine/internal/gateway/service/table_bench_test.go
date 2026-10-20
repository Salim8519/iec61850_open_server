package service

import (
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// BenchmarkTable5000 is the body of GET /tools/gateway/points for 5000 points (Table plus
// the Flask encoding): what one push re-read of gateway.points costs before its diff.
func BenchmarkTable5000(b *testing.B) {
	hub := newHub()
	var points []any
	for number := 0; number < 5000; number++ {
		ref, kind, value := benchRef(number)
		hub.define("bay", ref, Signal{Value: value})
		points = append(points, pyjson.NewObject().Set("id", ref).Set("kind", kind).Set("ied", "bay").Set("ref", ref).
			Set("address", int64(number+1)).Set("scale", 1.0).Set("description", ""))
	}
	ied := pyjson.NewObject().Set("id", "bay").Set("name", "Bay").Set("role", "simulated").Set("state", "running")
	section := gateway.EmptyGateway()
	get(section, "points").(*pyjson.Object).Set("iec104", points)
	manager := NewManager(&fakeProjects{ieds: []*pyjson.Object{ied}, section: section}, hub, nil, "", nil)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rows, err := manager.Table("iec104")
		if err != nil {
			b.Fatal(err)
		}
		if _, err := pyjson.Flask(pyjson.NewObject().Set("points", rows)); err != nil {
			b.Fatal(err)
		}
	}
}
