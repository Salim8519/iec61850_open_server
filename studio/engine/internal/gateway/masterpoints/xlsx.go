package masterpoints

import (
	"strconv"

	"github.com/xuri/excelize/v2"

	"studio-engine/internal/gateway/addpoints"
)

// TemplateHeader is the first row of the master template: no IED column, the master is
// address-only. ReadTable recognises every title.
var TemplateHeader = []any{"Name", "Address", "Type"}

const (
	helpSheet  = "How to fill"
	listsSheet = "Lists"
)

func helpLines(protocol string) []string {
	word := AddressWord[protocol]
	return []string{
		"How to fill this SCADA master point list (" + ProtocolName(protocol) + ")",
		"1. One row is one point the device should send or accept. Fill the rows on the Points sheet.",
		"2. Name: how your SCADA calls the point, for example \"Feeder 1 breaker\".",
		"3. Address: the " + word + " the device uses for this point (a whole number).",
		"4. Type: pick from the list. Only the types of " + ProtocolName(protocol) + " are listed.",
		"There is no IED column: a SCADA master only knows addresses and types.",
		"Save the file, then in ScadaBench choose My point list > Add points > Paste from Excel > Upload filled template.",
		"ScadaBench checks every row first. If one row is wrong, nothing is added and the row says why.",
	}
}

// Template is the Excel workbook to fill for protocol: a Points sheet with a Type drop-down per
// row and a whole-number rule for Address, a How to fill sheet and the hidden Lists sheet.
func Template(protocol string) ([]byte, error) {
	book := excelize.NewFile()
	defer book.Close()
	steps := []func(*excelize.File, string) error{writeHeader, writeHelp, writeTypes, writeRules}
	if err := book.SetSheetName("Sheet1", addpoints.PointsSheet); err != nil {
		return nil, err
	}
	for _, step := range steps {
		if err := step(book, protocol); err != nil {
			return nil, err
		}
	}
	book.SetActiveSheet(0)
	buffer, err := book.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeHeader(book *excelize.File, _ string) error {
	sheet := addpoints.PointsSheet
	bold, err := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err == nil {
		err = book.SetSheetRow(sheet, "A1", &TemplateHeader)
	}
	if err == nil {
		err = book.SetCellStyle(sheet, "A1", "C1", bold)
	}
	for column, width := range map[string]float64{"A": 36, "B": 12, "C": 44} {
		if err == nil {
			err = book.SetColWidth(sheet, column, column, width)
		}
	}
	if err == nil {
		err = book.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	}
	return err
}

func writeHelp(book *excelize.File, protocol string) error {
	if _, err := book.NewSheet(helpSheet); err != nil {
		return err
	}
	for row, line := range helpLines(protocol) {
		if err := book.SetCellStr(helpSheet, "A"+strconv.Itoa(row+1), line); err != nil {
			return err
		}
	}
	return book.SetColWidth(helpSheet, "A", "A", 110)
}

func writeTypes(book *excelize.File, protocol string) error {
	if _, err := book.NewSheet(listsSheet); err != nil {
		return err
	}
	types := addpoints.MasterTypes(protocol)
	for row, entry := range types {
		if err := book.SetCellStr(listsSheet, "A"+strconv.Itoa(row+1), addpoints.MasterLabel(entry)); err != nil {
			return err
		}
	}
	err := book.SetDefinedName(&excelize.DefinedName{Name: "Types",
		RefersTo: listsSheet + "!$A$1:$A$" + strconv.Itoa(max(1, len(types)))})
	if err != nil {
		return err
	}
	return book.SetSheetVisible(listsSheet, false)
}

func writeRules(book *excelize.File, protocol string) error {
	last := strconv.Itoa(MaxPoints + 1)
	kind := excelize.NewDataValidation(true)
	kind.Sqref = "C2:C" + last
	kind.SetSqrefDropList("Types")
	kind.SetError(excelize.DataValidationErrorStyleStop, "Type", "Pick a type from the list.")
	low, high := lowestAddress[protocol], addressLimit[protocol]
	address := excelize.NewDataValidation(true)
	address.Sqref = "B2:B" + last
	if err := address.SetRange(int(low), int(high), excelize.DataValidationTypeWhole,
		excelize.DataValidationOperatorBetween); err != nil {
		return err
	}
	address.SetError(excelize.DataValidationErrorStyleStop, "Address", "Type a whole number from "+
		strconv.FormatInt(low, 10)+" to "+strconv.FormatInt(high, 10)+".")
	for _, rule := range []*excelize.DataValidation{kind, address} {
		if err := book.AddDataValidation(addpoints.PointsSheet, rule); err != nil {
			return err
		}
	}
	return nil
}

// PlanWorkbook reads a filled template (or any sheet with the same titles) and checks its rows.
func PlanWorkbook(protocol string, data []byte, current []any) ([]Result, []string, error) {
	cells, err := addpoints.ReadWorkbook(data)
	if err != nil {
		return nil, nil, err
	}
	rows, numbers, columns := ReadTable(cells, protocol)
	return CheckRows(protocol, rows, numbers, current), columns, nil
}
