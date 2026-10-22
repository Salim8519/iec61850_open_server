package addpoints

import (
	"strings"
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// TestCandidatesFitTheType: only ST/MX values whose IEC 61850 type the kind carries, the
// suggested (Propose) ones first; commands by data class; points in the list are marked.
func TestCandidatesFitTheType(t *testing.T) {
	ied := NewIndex(project()).ByID("f1")
	items, total := Candidates(ied, "single", "", map[string]bool{"f1:FEED1LD0/PTOC1.Op.general:single": true})
	if total != 2 || items[0].Ref != "FEED1LD0/PTOC1.Op.general" || !items[0].Taken || !items[0].Suggested ||
		items[1].Ref != "FEED1LD0/GGIO1.SPCSO1.stVal" || items[1].Name != "Spare output" {
		t.Fatalf("single %+v", items)
	}
	if items, _ := Candidates(ied, "double", "xswi", nil); len(items) != 1 || items[0].Name != "XSWI1 Pos" {
		t.Fatalf("search %+v", items)
	}
	if items, _ := Candidates(ied, "float", "", nil); len(items) != 1 || items[0].FC != "MX" { // CF Enum left out
		t.Fatalf("float %+v", items)
	}
	double, _ := Candidates(ied, "command_double", "", nil)
	single, _ := Candidates(ied, "command_single", "", nil)
	if len(double) != 1 || double[0].StatusRef != "FEED1LD0/XCBR1.Pos.stVal" || double[0].Name != "Breaker position" ||
		len(single) != 1 || single[0].Ref != "FEED1LD0/GGIO1.SPCSO1" {
		t.Fatalf("commands %+v %+v", double, single)
	}
}

// TestPlanPairsCommandWithStatus: a ticked command gets its status point, both at the same
// distance from their blocks (IEC 104 20001 and 50001), names from the SCL; the points are
// IED points (linked to the IED command) and pass the saved-list rules.
func TestPlanPairsCommandWithStatus(t *testing.T) {
	ieds := project()
	ied := NewIndex(ieds).ByID("f1")
	results := FromIED("iec104", ied, "command_double", []Pick{{Ref: "FEED1LD0/XCBR1.Pos"}}, true, nil, ieds)
	if len(problems(results)) != 0 || len(results) != 2 {
		t.Fatalf("results %v", problems(results))
	}
	command, status := results[0].Point, results[1].Point
	if field(command, "address") != int64(50001) || field(status, "address") != int64(20001) ||
		field(command, "id") != "f1:FEED1LD0/XCBR1.Pos:command_double" || field(status, "kind") != "double" ||
		field(command, "description") != "Breaker position" || results[1].StatusFor != "f1:FEED1LD0/XCBR1.Pos:command_double" ||
		field(command, "monitor_only") != false {
		t.Fatalf("pair %s", pointJSON(t, Points(results)))
	}
	clean, invalid, err := gateway.ValidatePoints("iec104", Points(results), ieds)
	if err != nil || len(invalid) != 0 || len(clean) != 2 {
		t.Fatalf("validate %v %v", invalid, err)
	}
	// The status is already in the list at 20005: the command follows it to 50005, nothing else is added.
	current := []any{pyjson.NewObject().Set("id", "f1:FEED1LD0/XCBR1.Pos.stVal:double").Set("kind", "double").
		Set("ied", "f1").Set("ref", "FEED1LD0/XCBR1.Pos.stVal").Set("address", int64(20005))}
	results = FromIED("iec104", ied, "command_double", []Pick{{Ref: "FEED1LD0/XCBR1.Pos"}}, true, current, ieds)
	if len(results) != 1 || field(results[0].Point, "address") != int64(50005) {
		t.Fatalf("follow %s", pointJSON(t, Points(results)))
	}
	// Without the status tick only the command is planned, at the first free command address.
	results = FromIED("iec104", ied, "command_double", []Pick{{Ref: "FEED1LD0/XCBR1.Pos"}}, false, nil, ieds)
	if len(results) != 1 || field(results[0].Point, "address") != int64(50001) {
		t.Fatalf("alone %s", pointJSON(t, Points(results)))
	}
}

// TestPlanRefusesWhatDoesNotFit: wrong type, unknown reference, a point already listed and
// a typed address that is used are refused with words; free addresses skip used ones.
func TestPlanRefusesWhatDoesNotFit(t *testing.T) {
	ieds := project()
	ied := NewIndex(ieds).ByID("f1")
	current := []any{pyjson.NewObject().Set("id", "f1:FEED1LD0/PTOC1.Op.general:single").Set("kind", "single").
		Set("ied", "f1").Set("ref", "FEED1LD0/PTOC1.Op.general").Set("address", int64(10001))}
	results := FromIED("iec104", ied, "single", []Pick{{Ref: "FEED1LD0/XCBR1.Pos.stVal"}, {Ref: "FEED1LD0/NOPE.stVal"},
		{Ref: "FEED1LD0/PTOC1.Op.general"}, {Ref: "FEED1LD0/GGIO1.SPCSO1.stVal", Address: 10001, HasAddress: true}},
		false, current, ieds)
	want := []string{"FEED1LD0/XCBR1.Pos.stVal is a Dbpos ST value and does not fit this type",
		"FEED1 has no signal FEED1LD0/NOPE.stVal", "this Single point is already in the point list",
		"address 10001 is already used by another point"}
	if strings.Join(problems(results), "|") != strings.Join(want, "|") {
		t.Fatalf("problems %q", problems(results))
	}
	results = FromIED("iec104", ied, "single", []Pick{{Ref: "FEED1LD0/GGIO1.SPCSO1.stVal", Name: "Spare"}}, false, current, ieds)
	if field(results[0].Point, "address") != int64(10002) || field(results[0].Point, "description") != "Spare" {
		t.Fatalf("next free %s", pointJSON(t, Points(results)))
	}
	if results := FromIED("iec104", ied, "command_single", []Pick{{Ref: "FEED1LD0/XCBR1.Pos"}}, true, nil, ieds); len(results) != 1 ||
		results[0].Problem != "FEED1LD0/XCBR1.Pos is a DPC object; choose a double command" {
		t.Fatalf("class %v", problems(results))
	}
}

// TestPlanRealIEDAndOtherProtocols: a real IED's points say monitor_only (the page warns);
// Modbus and DNP3 pair coil/CROB with the status index.
func TestPlanRealIEDAndOtherProtocols(t *testing.T) {
	ieds := project()
	index := NewIndex(ieds)
	results := FromIED("dnp3", index.ByID("r1"), "command_double", []Pick{{Ref: "REAL1LD0/XCBR1.Pos"}}, true, nil, ieds)
	if len(problems(results)) != 0 || field(results[0].Point, "monitor_only") != true ||
		field(results[0].Point, "ied_role") != "real" || field(results[0].Point, "address") != int64(0) ||
		field(results[1].Point, "address") != int64(0) {
		t.Fatalf("real %s %v", pointJSON(t, Points(results)), problems(results))
	}
	current := []any{pyjson.NewObject().Set("kind", "single").Set("address", int64(0)).Set("ied", "x").Set("ref", "y")}
	results = FromIED("modbus", index.ByID("f1"), "command_single", []Pick{{Ref: "FEED1LD0/GGIO1.SPCSO1"}}, true, current, ieds)
	if len(problems(results)) != 0 || field(results[0].Point, "address") != int64(1) || field(results[1].Point, "address") != int64(1) {
		t.Fatalf("modbus %s %v", pointJSON(t, Points(results)), problems(results))
	}
}

// TestOwnRange: N made-up points from address X, numbered names; a used address stops it.
func TestOwnRange(t *testing.T) {
	results := OwnRange("modbus", "float", 3, 1000, "Tank", []any{pyjson.NewObject().Set("id", "own:4").
		Set("source", "manual").Set("kind", "single").Set("address", int64(0))})
	if !OK(results) || len(results) != 3 || field(results[1].Point, "address") != int64(1002) ||
		field(results[2].Point, "id") != "own:7" || field(results[0].Point, "description") != "Tank 1" {
		t.Fatalf("range %s", pointJSON(t, Points(results)))
	}
	used := []any{pyjson.NewObject().Set("kind", "float").Set("address", int64(1004)).Set("ied", "x").Set("ref", "y")}
	results = OwnRange("modbus", "float", 3, 1000, "", used)
	if OK(results) || len(results) != 3 || results[2].Problem != "Point 3: address 1004 is already used by another point" {
		t.Fatalf("clash %v", problems(results))
	}
	if results := OwnRange("iec104", "single", 0, 1, "", nil); OK(results) {
		t.Fatal("count 0 accepted")
	}
}
