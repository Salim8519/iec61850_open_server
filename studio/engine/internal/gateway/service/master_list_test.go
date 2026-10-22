package service

import (
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/gateway/masterpoints"
	"studio-engine/internal/pyjson"
)

// The Modbus poll reads the user's ranges first, then the reads the master point list needs that the
// user's ranges do not cover already (coil 0 is in the user's coils 0-7).
func TestModbusPollAddsTheListRanges(t *testing.T) {
	manual := pyjson.NewObject().Set("table", "coils").Set("start", int64(0)).Set("count", int64(8)).
		Set("data_type", "bool").Set("word_order", "big")
	list, err := masterpoints.Validate("modbus", []any{
		pyjson.NewObject().Set("type", "holding_float32").Set("address", int64(100)),
		pyjson.NewObject().Set("type", "holding_float32").Set("address", int64(102)),
		pyjson.NewObject().Set("type", "coil").Set("address", int64(0)),
	})
	if err != nil {
		t.Fatal(err)
	}
	master := masterpoints.WithList(gateway.EmptyMaster().Set("modbus_ranges", []any{manual}), "modbus", list)
	ranges := (&Master{}).pollRanges(master)
	text, _ := pyjson.Compact(ranges)
	want := `[{"table":"coils","start":0,"count":8,"data_type":"bool","word_order":"big"},` +
		`{"table":"holding_registers","start":100,"count":2,"data_type":"float32","word_order":"big"}]`
	if string(text) != want {
		t.Fatalf("got  %s\nwant %s", text, want)
	}
	if again := (&Master{}).pollRanges(masterpoints.WithList(master, "modbus", []any{})); len(again) != 1 {
		t.Fatal("without list points the poll reads only the user's ranges")
	}
}

// Mapped is the gateway's view of one of its points: the IED value behind it, for the master's
// optional Check against IED. It needs the simulation, not the gateway worker.
func TestGatewayMappedReadsTheIEDBehindAGatewayPoint(t *testing.T) {
	hub := newHub()
	ied := pyjson.NewObject().Set("id", "f1").Set("name", "FEED1").Set("role", "simulated").Set("state", "running")
	section := gateway.EmptyGateway()
	points, _ := section.Get("points")
	points.(*pyjson.Object).Set("iec104", []any{pyjson.NewObject().Set("id", "f1:Q0:double").Set("kind", "double").
		Set("ied", "f1").Set("ref", "LD0/XCBR1.Pos.stVal").Set("address", int64(20001)).Set("scale", 1.0)})
	manager := NewManager(&fakeProjects{ieds: []*pyjson.Object{ied}, section: section}, hub, nil, "", nil)
	hub.set("f1", "LD0/XCBR1.Pos.stVal", int64(2))
	mapped := manager.Mapped("iec104", 20001, "double")
	if !mapped.Found || !mapped.Online || mapped.Value != int64(2) || mapped.Source != "FEED1 LD0/XCBR1.Pos.stVal" {
		t.Fatalf("mapped: %+v", mapped)
	}
	if other := manager.Mapped("iec104", 20001, "single"); other.Found {
		t.Fatal("the kind must match too")
	}
	hub.state = "stopped"
	if stopped := manager.Mapped("iec104", 20001, "double"); stopped.Online {
		t.Fatal("a stopped simulation has no current value")
	}
}
