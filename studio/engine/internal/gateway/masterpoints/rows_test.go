package masterpoints

import (
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
)

func TestPastedRowsAreCheckedBeforeAnythingIsAdded(t *testing.T) {
	good, columns := PlanText("iec104", "Name\tIOA\tType\nBreaker Q0\t20001\tM_DP_NA_1\nVoltage\t30001\tMeasured value, float\n", nil)
	if !OK(good) || strings.Join(columns, ",") != "name,address,type" || text(good[0].Point, "type") != "double" ||
		text(good[1].Point, "type") != "float" || good[1].Number != 3 {
		t.Fatalf("pasted with a title row: %s %v", jsonText(t, PlanJSON(good)), columns)
	}
	guessed, _ := PlanText("iec104", "Breaker Q1;20002;Double point", nil)
	if !OK(guessed) || Address(guessed[0].Point) != 20002 {
		t.Fatalf("without a title row the columns are guessed: %s", jsonText(t, PlanJSON(guessed)))
	}
	current, _ := Validate("iec104", []any{obj("type", "single", "address", int64(10001), "name", "Alarm")})
	bad, _ := PlanText("iec104", "Name,Address,Type\nA,10001,single\nB,,double\nC,5,counter of sheep\nD,6,double\n", current)
	problems := []string{}
	for _, row := range bad {
		problems = append(problems, row.Problem)
	}
	want := []string{`Row 2: IOA 10001 is already used by "Alarm"`, "Row 3: write the IOA the device uses (a whole number)",
		`Row 4: "counter of sheep" is not a IEC 104 type`, ""}
	for at, prefix := range want {
		if !strings.HasPrefix(problems[at], prefix) || (prefix == "" && problems[at] != "") {
			t.Errorf("row %d: got %q want %q", at+2, problems[at], prefix)
		}
	}
	if OK(bad) || len(field(PlanJSON(bad), "points").([]any)) != 0 {
		t.Fatal("one bad row adds nothing")
	}
	if plan := PlanJSON(bad); text(field(plan, "rows").([]any)[0].(*pyjson.Object), "problem") == "" {
		t.Fatal("bad rows carry their reason")
	}
}

// The CSV export reads back to the same list, and the template filled in Excel reads back too.
func TestCSVAndExcelRoundTrip(t *testing.T) {
	list, err := Validate("modbus", []any{
		obj("type", "holding_float32", "address", int64(100), "name", "P, total", "scale", 0.5, "unit", "kW", "word_order", "little"),
		obj("type", "coil", "address", int64(3), "name", "Breaker", "value", int64(0)),
		obj("type", "discrete_input", "address", int64(12), "name", "Alarm"),
	})
	if err != nil {
		t.Fatal(err)
	}
	csv := string(ExportCSV("modbus", list))
	if !strings.HasPrefix(csv, "\xef\xbb\xbfName,Address,Type,") || !strings.Contains(csv, "Holding register, float32 (FC3, 2 registers)") {
		t.Fatalf("csv: %q", csv)
	}
	back, _ := PlanText("modbus", csv, nil)
	if !OK(back) {
		t.Fatalf("csv import: %s", jsonText(t, PlanJSON(back)))
	}
	for at := range list {
		want, got := jsonText(t, list[at]), jsonText(t, back[at].Point)
		if want != got {
			t.Errorf("round trip %d\n got %s\nwant %s", at, got, want)
		}
	}
	book, err := Template("iec104")
	if err != nil {
		t.Fatal(err)
	}
	file, err := excelize.OpenReader(strings.NewReader(string(book)))
	if err != nil {
		t.Fatal(err)
	}
	if rules, _ := file.GetDataValidations(addpoints.PointsSheet); len(rules) != 2 {
		t.Fatalf("the template has the Type list and the address rule, got %d rules", len(rules))
	}
	if header, _ := file.GetRows(addpoints.PointsSheet); len(header) == 0 || strings.Join(header[0], ",") != "Name,Address,Type" {
		t.Fatalf("no IED column: %v", header)
	}
	file.SetSheetRow(addpoints.PointsSheet, "A2", &[]any{"Feeder 1 breaker", 20001, "Double point (M_DP 3/31)"})
	file.SetSheetRow(addpoints.PointsSheet, "A3", &[]any{"Feeder 1 close", 50001, "Double command (C_DC 46/59)"})
	filled, _ := file.WriteToBuffer()
	results, _, err := PlanWorkbook("iec104", filled.Bytes(), nil)
	if err != nil || !OK(results) || text(results[1].Point, "command") != "double" || Address(results[0].Point) != 20001 {
		t.Fatalf("filled template: %v %s", err, jsonText(t, PlanJSON(results)))
	}
	if _, _, err := PlanWorkbook("iec104", []byte("not a workbook"), nil); err == nil {
		t.Fatal("a file that is not a workbook is refused")
	}
}
