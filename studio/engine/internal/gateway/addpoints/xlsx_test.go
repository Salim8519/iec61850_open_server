package addpoints

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestTemplateHasDropDowns: the template's Points sheet has the Type, IED, Point and Address
// rules; the hidden Lists sheet holds the types, the IEDs and one named range per IED and
// type with only the fitting references.
func TestTemplateHasDropDowns(t *testing.T) {
	data, err := Template("iec104", NewIndex(project()))
	if err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	if got := strings.Join(book.GetSheetList(), ","); got != "Points,How to fill,Lists" {
		t.Fatalf("sheets %s", got)
	}
	if visible, _ := book.GetSheetVisible("Lists"); visible {
		t.Fatal("Lists sheet is visible")
	}
	header, _ := book.GetRows("Points")
	if strings.Join(header[0], ",") != "Name,Type,IED,Point,Address,Scale" {
		t.Fatalf("header %v", header)
	}
	names := map[string]string{}
	for _, name := range book.GetDefinedName() {
		names[name.Name] = name.RefersTo
	}
	// FEED1 is IED 1, Double point is type 2: column 3 + 1 = D, two positions fit. Nine 104 types
	// (ADR-0107), so IED 2's type 6 is column 3 + 9 + 5 = Q.
	if names["Types"] != "Lists!$A$1:$A$9" || names["IEDs"] != "Lists!$B$1:$B$3" || names["P_1_2"] != "Lists!$D$1:$D$2" ||
		names["P_2_6"] != "Lists!$Q$1:$Q$1" {
		t.Fatalf("names %v", names)
	}
	if cell, _ := book.GetCellValue("Lists", "D2"); cell != "FEED1LD0/XSWI1.Pos.stVal" {
		t.Fatalf("D2 %q", cell)
	}
	if cell, _ := book.GetCellValue("Lists", "B3"); cell != SimulatedChoice {
		t.Fatalf("B3 %q", cell)
	}
	rules, err := book.GetDataValidations("Points")
	if err != nil || len(rules) != 4 {
		t.Fatalf("rules %v %v", rules, err)
	}
	formulas := []string{}
	for _, rule := range rules {
		formulas = append(formulas, rule.Sqref+"="+rule.Formula1)
	}
	want := `B2:B5001=Types|C2:C5001=IEDs|D2:D5001=INDIRECT("P_"&MATCH($C2,IEDs,0)&"_"&MATCH($B2,Types,0))|E2:E5001=1`
	if got := strings.Join(formulas, "|"); got != want {
		t.Fatalf("rules\n got %s\nwant %s", got, want)
	}
}

// TestTemplateRoundTrip: generate, fill (as a person picks from the drop-downs), save, upload:
// every row is read back and checked into points.
func TestTemplateRoundTrip(t *testing.T) {
	ieds := project()
	index := NewIndex(ieds)
	data, err := Template("iec104", index)
	if err != nil {
		t.Fatal(err)
	}
	book, _ := excelize.OpenReader(bytes.NewReader(data))
	double, _ := Entry("iec104", "double")
	command, _ := Entry("iec104", "command_double")
	measured, _ := Entry("iec104", "float")
	filled := [][]any{{"", Label(double), "FEED1", "FEED1LD0/XCBR1.Pos.stVal", 20001, ""},
		{"CB1 open/close", Label(command), "FEED1", "FEED1LD0/XCBR1.Pos", 50001, ""},
		{"Feeder power", Label(measured), "FEED1", "FEED1LD0/MMXU1.TotW.mag.f", "", 0.001},
		{"Tank level", Label(measured), SimulatedChoice, "", "", ""}}
	for at, row := range filled {
		cell, _ := excelize.CoordinatesToCellName(1, at+2)
		if err := book.SetSheetRow("Points", cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	buffer, _ := book.WriteToBuffer()
	book.Close()
	cells, err := ReadWorkbook(buffer.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	rows, numbers, columns := ReadTable(cells, "iec104", index)
	if strings.Join(columns, ",") != "name,type,ied,point,address,scale" || len(rows) != 4 || numbers[0] != 2 {
		t.Fatalf("read %v %v", columns, rows)
	}
	results := CheckRows("iec104", rows, numbers, index, ieds, nil)
	if !OK(results) {
		t.Fatalf("problems %v", problems(results))
	}
	points := Points(results)
	if field(points[0], "description") != "Breaker position" || field(points[1], "address") != int64(50001) ||
		field(points[2], "scale") != 0.001 || field(points[2], "address") != int64(30001) ||
		field(points[3], "source") != "manual" || field(points[3], "address") != int64(30002) {
		t.Fatalf("points %s", pointJSON(t, points))
	}
	if _, err := ReadWorkbook([]byte("type,address\nsingle,1\n")); err != ErrNotWorkbook {
		t.Fatalf("csv as xlsx: %v", err)
	}
}
