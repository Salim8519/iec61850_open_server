package addpoints

import (
	"strings"
	"testing"
)

// TestResolveType reads types as people write them, per protocol.
func TestResolveType(t *testing.T) {
	cases := map[string]string{"double": "double", "Double point": "double", "M_DP_TB_1": "double",
		"m_me_nc_1": "float", "Measured value, scaled (M_ME_NB 11/35)": "integer", "C_SC": "command_single"}
	for text, want := range cases {
		if got, ok := ResolveType("iec104", text); !ok || got != want {
			t.Errorf("%q: %q %v", text, got, ok)
		}
	}
	if got, ok := ResolveType("modbus", "coil"); !ok || got != "command_single" {
		t.Errorf("modbus coil %q", got)
	}
	if _, ok := ResolveType("dnp3", "M_SP"); ok {
		t.Error("104 type id accepted for DNP3")
	}
	for _, protocol := range []string{"iec104", "dnp3", "modbus"} {
		seen := map[string]bool{}
		for _, entry := range Types(protocol) {
			seen[entry.Kind] = true
		}
		// six Python kinds plus the typed ones (ADR-0107); Modbus has no regulating step
		if want := map[string]int{"iec104": 9, "dnp3": 9, "modbus": 8}[protocol]; len(seen) != want || !Types(protocol)[0].MostUsed {
			t.Errorf("%s catalog %v", protocol, Types(protocol))
		}
	}
}

// TestPasteWithHeader: tab-separated rows copied from Excel with a title row; columns in any
// order; names from the SCL when empty; typed and next free addresses; a made-up row.
func TestPasteWithHeader(t *testing.T) {
	ieds := project()
	index := NewIndex(ieds)
	text := "IOA\tType\tIED\tSignal\tDescription\r\n" +
		"20010\tM_DP\tFEED1\tFEED1LD0/XCBR1.Pos.stVal\t\r\n" +
		"\tDouble command\tfeed1\tLD0/XCBR1.Pos\tCB1 command\r\n" +
		"\t\t\t\t\r\n" +
		"\tSingle point\tSimulated\t\tPump running\r\n"
	rows, numbers, columns := ReadTable(SplitText(text), "iec104", index)
	if strings.Join(columns, ",") != "address,type,ied,point,name" || len(rows) != 3 || numbers[2] != 5 {
		t.Fatalf("table %v %v %v", columns, rows, numbers)
	}
	results := CheckRows("iec104", rows, numbers, index, ieds, nil)
	if !OK(results) {
		t.Fatalf("problems %v", problems(results))
	}
	want := `[{"id":"f1:FEED1LD0/XCBR1.Pos.stVal:double","kind":"double","ied":"f1","ref":"FEED1LD0/XCBR1.Pos.stVal",` +
		`"scale":1.0,"description":"Breaker position","ied_name":"FEED1","ied_role":"simulated","monitor_only":false,"address":20010},` +
		`{"id":"f1:FEED1LD0/XCBR1.Pos:command_double","kind":"command_double","ied":"f1","ref":"FEED1LD0/XCBR1.Pos",` +
		`"scale":1.0,"description":"CB1 command","ied_name":"FEED1","ied_role":"simulated","monitor_only":false,"address":50001},` +
		`{"kind":"single","source":"manual","ied":"","ref":"","scale":1.0,"description":"Pump running","ied_name":"Own point",` +
		`"ied_role":"own","monitor_only":false,"value":0,"address":10001,"id":"own:1"}]`
	if got := pointJSON(t, Points(results)); got != want {
		t.Fatalf("points\n got %s\nwant %s", got, want)
	}
}

// TestPasteMarksEveryBadRow: without a title row the columns are guessed; each bad row says
// why and nothing can be added while one is bad.
func TestPasteMarksEveryBadRow(t *testing.T) {
	ieds := project()
	index := NewIndex(ieds)
	text := "10001\tSingle point\tFEED1\tFEED1LD0/PTOC1.Op.general\tTrip\n" +
		"10001\tSingle point\tFEED1\tFEED1LD0/GGIO1.SPCSO1.stVal\tSame address\n" +
		"10003\tAnalog thing\tFEED1\tFEED1LD0/MMXU1.TotW.mag.f\t\n" +
		"10004\tSingle point\tNOPE\tX/Y.stVal\t\n" +
		"30001\tSingle point\tFEED1\tFEED1LD0/MMXU1.TotW.mag.f\t\n" +
		"abc\tDouble point\tFEED1\tFEED1LD0/XSWI1.Pos.stVal\t\n" +
		"10007\tSingle point\tSimulated\tFEED1LD0/GGIO1.SPCSO1.stVal\t\n"
	rows, numbers, columns := ReadTable(SplitText(text), "iec104", index)
	if strings.Join(columns, ",") != "address,type,ied,point,name" || len(rows) != 7 {
		t.Fatalf("guess %v", columns)
	}
	results := CheckRows("iec104", rows, numbers, index, ieds, nil)
	want := []string{"Row 2: address 10001 is already used by another point",
		"Row 3: \"Analog thing\" is not a type here; use one of: Single point, Double point, Measured value, float, " +
			"Measured value, scaled, Single command, Double command, Set point command, Regulating step command, Step position (tap)",
		"Row 4: this project has no IED called \"NOPE\"",
		"Row 5: FEED1LD0/MMXU1.TotW.mag.f is a FLOAT32 MX value and does not fit this type",
		"Row 6: the address must be a whole number",
		"Row 7: a made-up point has no IED path; write the IED name, or clear the Point cell"}
	if OK(results) || strings.Join(problems(results), "\n") != strings.Join(want, "\n") {
		t.Fatalf("problems\n%s", strings.Join(problems(results), "\n"))
	}
	if results[0].Problem != "" || results[0].Cells.Point != "FEED1LD0/PTOC1.Op.general" {
		t.Fatalf("good row %+v", results[0])
	}
}

// TestPasteAgainstTheList: a signal already in the list and an address the list uses are refused.
func TestPasteAgainstTheList(t *testing.T) {
	ieds := project()
	index := NewIndex(ieds)
	current := FromIED("iec104", index.ByID("f1"), "single", []Pick{{Ref: "FEED1LD0/PTOC1.Op.general"}}, false, nil, ieds)
	rows, numbers, _ := ReadTable(SplitText("Address;Type;IED;Point\n;single;FEED1;FEED1LD0/PTOC1.Op.general\n"+
		"10001;single;FEED1;FEED1LD0/GGIO1.SPCSO1.stVal\n"), "iec104", index)
	results := CheckRows("iec104", rows, numbers, index, ieds, Points(current))
	want := "Row 2: this Single point is already in the list|Row 3: address 10001 is already used by another point"
	if strings.Join(problems(results), "|") != want {
		t.Fatalf("problems %q", problems(results))
	}
}
