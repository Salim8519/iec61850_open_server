package snmpfile

import (
	"reflect"
	"strings"
	"testing"

	"studio-engine/internal/snmp/model"
)

func TestCSVRoundTrip(t *testing.T) {
	lib := standardLib(t)
	points, _, err := ParseWalk(readTestdata(t, "walk-default.txt"), lib)
	if err != nil {
		t.Fatal(err)
	}
	points = append(points,
		model.Point{OID: "1.3.6.1.4.1.99999.9.1", Type: model.TypeCounter64, Value: "5",
			Behaviour: model.BehaviourCounter, Rate: 0.125, Writable: true, Label: "Feeder 1, packets"},
		model.Point{OID: "1.3.6.1.4.1.99999.9.2", Type: model.TypeOctets, Value: "  padded, \"text\"  ",
			Behaviour: model.BehaviourFixed, Writable: true})
	text := WriteCSV(points, lib)
	if !strings.HasPrefix(text, "oid,name,type,value,hex,behaviour,rate,writable,label\n") || !strings.HasSuffix(text, "\n") {
		t.Fatalf("CSV text:\n%s", text)
	}
	if !strings.Contains(text, "\n1.3.6.1.2.1.1.1.0,SNMPv2-MIB::sysDescr.0,octets,Acme relay R-100 firmware 2.1,false,fixed,,false,\n") {
		t.Errorf("sysDescr row missing:\n%s", text)
	}
	if !strings.Contains(text, "\n1.3.6.1.4.1.99999.9.1,SNMPv2-SMI::enterprises.99999.9.1,counter64,5,false,counter,0.125,true,\"Feeder 1, packets\"\n") {
		t.Errorf("counter row missing:\n%s", text)
	}
	back, problems, err := ParseCSV(text, lib)
	if err != nil || len(problems) != 0 {
		t.Fatalf("ParseCSV: %v %+v", err, problems)
	}
	if !reflect.DeepEqual(back, points) {
		for i := range points {
			if i < len(back) && !reflect.DeepEqual(back[i], points[i]) {
				t.Errorf("row %d: got %+v want %+v", i, back[i], points[i])
			}
		}
		t.Fatalf("round trip differs: %d vs %d points", len(back), len(points))
	}
	if again := WriteCSV(back, lib); again != text {
		t.Error("second WriteCSV differs from the first")
	}
	if back, _, err := ParseCSV(WriteCSV(points, nil), nil); err != nil || !reflect.DeepEqual(back, points) {
		t.Errorf("round trip without MIBs: %v", err)
	}
}

func TestParseCSVRowsAndProblems(t *testing.T) {
	points, problems, err := ParseCSV(readTestdata(t, "points.csv"), standardLib(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []model.Point{
		{OID: "1.3.6.1.2.1.1.5.0", Type: "octets", Value: "relay-01", Behaviour: "fixed", Writable: true},
		{OID: "1.3.6.1.2.1.1.3.0", Type: "timeticks", Value: "100", Behaviour: "uptime"},
		{OID: "1.3.6.1.2.1.2.2.1.8.2", Type: "integer", Value: "1", Behaviour: "fixed"},
		{OID: "1.3.6.1.2.1.2.2.1.6.2", Type: "octets", Value: "00 15 5d 0a 0b 0c", Hex: true, Behaviour: "fixed"},
		{OID: "1.3.6.1.2.1.1.1.0", Type: "octets", Value: "two\nlines, with \"quotes\"", Behaviour: "fixed"},
		{OID: "1.3.6.1.2.1.2.2.1.10.2", Type: "counter32", Value: "7", Behaviour: "counter", Rate: 2.5, Writable: true},
		{OID: "1.3.6.1.4.1.99999.1.1", Type: "gauge32", Value: "43", Behaviour: "fixed"},
		{OID: "1.3.6.1.4.1.99999.1.13", Type: "oid", Value: "1.3.6.1.2.1.2.2.1.1", Behaviour: "fixed"},
		{OID: "1.3.6.1.4.1.99999.1.14", Type: "opaque", Value: "9f 78 04 3f c0 00 00", Behaviour: "fixed"},
		{OID: "1.3.6.1.4.1.99999.1.15", Type: "null", Behaviour: "fixed"},
	}
	if !reflect.DeepEqual(points, want) {
		for i := range points {
			if i < len(want) && !reflect.DeepEqual(points[i], want[i]) {
				t.Errorf("point %d = %+v\nwant %+v", i, points[i], want[i])
			}
		}
		t.Fatalf("got %d points, want %d", len(points), len(want))
	}
	wantProblems := []struct {
		line int
		text string
	}{
		{10, "2147483648"}, {11, "4294967296"}, {12, "18446744073709551616"}, {13, "dotted quad"},
		{14, "1.3.x.6"}, {15, "hex value"}, {16, "maybe"}, {17, "sometimes"}, {18, "needs type timeticks"},
		{19, "NO-SUCH-MIB::thing.0: unknown name"}, {20, "null point has no value"}, {21, "rate \"fast\""},
		{22, "type \"float\" is unknown"}, {23, "neither an oid nor a name"}, {24, "appears more than once"},
	}
	if len(problems) != len(wantProblems) {
		t.Fatalf("problems = %+v", problems)
	}
	for i, w := range wantProblems {
		if problems[i].Line != w.line || !strings.Contains(problems[i].Text, w.text) {
			t.Errorf("problem %d = %+v; want line %d with %q", i, problems[i], w.line, w.text)
		}
	}
}

func TestParseCSVErrors(t *testing.T) {
	lib := standardLib(t)
	for _, text := range []string{"", "1.3.6.1.2.1.1.5.0,,octets,x\n", "oid,colour\n1.3.6.1,red\n",
		"name,type\nsysName.0,octets\n"} {
		if _, _, err := ParseCSV(text, lib); err == nil || !strings.Contains(err.Error(), "header") {
			t.Errorf("ParseCSV(%q) error = %v", text, err)
		}
	}
	points, problems, err := ParseCSV("oid,name,type,value,hex,behaviour,rate,writable\n", lib)
	if err != nil || len(points) != 0 || len(problems) != 0 {
		t.Errorf("header only: %v %v %v", points, problems, err)
	}
	_, problems, err = ParseCSV("oid,type,value\n1.3.6.1.2.1.1.5.0,integer,abc\n,octets,x\n", lib)
	if err == nil || len(problems) != 2 || !strings.Contains(err.Error(), "none of the 2 CSV rows") {
		t.Errorf("all rows bad: %v %+v", err, problems)
	}
	points, problems, err = ParseCSV("\ufeffOID , Value\n1.3.6.1.2.1.1.5.0,x\n2.5,\"bad\"quote\n1.3.6.1.2.1.1.6.0,y\n", lib)
	if err != nil || len(points) != 2 || len(problems) != 1 || problems[0].Line != 3 {
		t.Errorf("BOM, spaced header and a malformed row: %v %+v %v", points, problems, err)
	}
}
