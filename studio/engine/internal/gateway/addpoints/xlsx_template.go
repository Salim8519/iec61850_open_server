package addpoints

import (
	"strconv"

	"github.com/xuri/excelize/v2"

	"studio-engine/internal/gateway"
)

// Sheet names of the Excel template.
const (
	PointsSheet = "Points"
	helpSheet   = "How to fill"
	listsSheet  = "Lists"
	// TemplateRows is how many rows of the template have the drop-down lists.
	TemplateRows = gateway.MaxPoints
)

// SimulatedChoice is the IED list entry for a made-up point.
const SimulatedChoice = "Simulated (made up)"

// TemplateHeader is the template's first row; ReadTable recognises every title.
var TemplateHeader = []any{"Name", "Type", "IED", "Point", "Address", "Scale"}

var helpLines = []string{
	"How to fill this point list",
	"1. One row is one SCADA point. Fill the rows on the Points sheet.",
	"2. Type: pick from the list. Only the types of this protocol are listed.",
	"3. IED: pick a project IED, or \"" + SimulatedChoice + "\" for a point whose value you make up.",
	"4. Point: pick from the list. It shows only that IED's signals or commands that fit the type.",
	"   Choose the Type and the IED first, then the Point list fills in.",
	"5. Address: leave it empty and ScadaBench gives the next free address, or type your own.",
	"6. Name: leave it empty and ScadaBench uses the description from the SCL file.",
	"7. Scale: leave it empty for 1. SCADA receives the value multiplied by the scale.",
	"Save the file, then in ScadaBench choose Add points > Paste from Excel > Upload filled template.",
	"ScadaBench checks every row first. If one row is wrong, nothing is added and the row says why.",
}

// Template is the Excel workbook to fill for protocol: a Points sheet with drop-down lists
// (Type, IED, and Point filtered by the chosen IED and type through hidden named ranges),
// a How to fill sheet and the hidden Lists sheet the drop-downs read.
func Template(protocol string, index *Index) ([]byte, error) {
	book := excelize.NewFile()
	defer book.Close()
	if err := book.SetSheetName("Sheet1", PointsSheet); err != nil {
		return nil, err
	}
	steps := []func(*excelize.File, string, *Index) error{writeHeader, writeHelp, writeLists, writeValidations}
	for _, step := range steps {
		if err := step(book, protocol, index); err != nil {
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

func writeHeader(book *excelize.File, _ string, _ *Index) error {
	bold, err := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err == nil {
		err = book.SetSheetRow(PointsSheet, "A1", &TemplateHeader)
	}
	if err == nil {
		err = book.SetCellStyle(PointsSheet, "A1", "F1", bold)
	}
	for column, width := range map[string]float64{"A": 32, "B": 34, "C": 22, "D": 46, "E": 12, "F": 10} {
		if err == nil {
			err = book.SetColWidth(PointsSheet, column, column, width)
		}
	}
	if err == nil {
		err = book.SetPanes(PointsSheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	}
	return err
}

func writeHelp(book *excelize.File, _ string, _ *Index) error {
	if _, err := book.NewSheet(helpSheet); err != nil {
		return err
	}
	for row, line := range helpLines {
		if err := book.SetCellStr(helpSheet, "A"+strconv.Itoa(row+1), line); err != nil {
			return err
		}
	}
	return book.SetColWidth(helpSheet, "A", "A", 110)
}

// writeLists fills the hidden Lists sheet: A the types, B the IEDs, then one column per IED
// and type with the fitting references, each a named range P_<ied>_<type> (1-based).
func writeLists(book *excelize.File, protocol string, index *Index) error {
	if _, err := book.NewSheet(listsSheet); err != nil {
		return err
	}
	types := Types(protocol)
	for row, entry := range types {
		if err := book.SetCellStr(listsSheet, "A"+strconv.Itoa(row+1), Label(entry)); err != nil {
			return err
		}
	}
	names := []string{}
	for _, ied := range index.List {
		names = append(names, ied.Name)
	}
	names = append(names, SimulatedChoice)
	for row, name := range names {
		if err := book.SetCellStr(listsSheet, "B"+strconv.Itoa(row+1), name); err != nil {
			return err
		}
	}
	if err := define(book, "Types", "A", len(types)); err != nil {
		return err
	}
	if err := define(book, "IEDs", "B", len(names)); err != nil {
		return err
	}
	column := 3
	for iedAt, ied := range index.List {
		for typeAt, entry := range types {
			name, _ := excelize.ColumnNumberToName(column)
			refs := FittingRefs(ied, entry.Kind)
			for row, ref := range refs {
				if err := book.SetCellStr(listsSheet, name+strconv.Itoa(row+1), ref); err != nil {
					return err
				}
			}
			if err := define(book, "P_"+strconv.Itoa(iedAt+1)+"_"+strconv.Itoa(typeAt+1), name, len(refs)); err != nil {
				return err
			}
			column++
		}
	}
	return book.SetSheetVisible(listsSheet, false)
}

func define(book *excelize.File, name, column string, rows int) error {
	if rows < 1 {
		rows = 1 // an empty list points at one empty cell: the drop-down is empty
	}
	return book.SetDefinedName(&excelize.DefinedName{Name: name,
		RefersTo: listsSheet + "!$" + column + "$1:$" + column + "$" + strconv.Itoa(rows)})
}

func writeValidations(book *excelize.File, protocol string, _ *Index) error {
	last := strconv.Itoa(TemplateRows + 1)
	kind := excelize.NewDataValidation(true)
	kind.Sqref = "B2:B" + last
	kind.SetSqrefDropList("Types")
	kind.SetError(excelize.DataValidationErrorStyleStop, "Type", "Pick a type from the list.")
	ied := excelize.NewDataValidation(true)
	ied.Sqref = "C2:C" + last
	ied.SetSqrefDropList("IEDs")
	ied.SetError(excelize.DataValidationErrorStyleStop, "IED", "Pick an IED of this project from the list.")
	point := excelize.NewDataValidation(true)
	point.Sqref = "D2:D" + last
	// The formula is written into the sheet XML as it is, so "&" is escaped here.
	point.SetSqrefDropList(`INDIRECT("P_"&amp;MATCH($C2,IEDs,0)&amp;"_"&amp;MATCH($B2,Types,0))`)
	point.SetError(excelize.DataValidationErrorStyleWarning, "Point",
		"This is not one of the listed signals of that IED and type. ScadaBench will check it when you upload.")
	bounds := gateway.AddressRange[protocol]
	address := excelize.NewDataValidation(true)
	address.Sqref = "E2:E" + last
	if err := address.SetRange(int(bounds[0]), int(bounds[1]), excelize.DataValidationTypeWhole,
		excelize.DataValidationOperatorBetween); err != nil {
		return err
	}
	address.SetError(excelize.DataValidationErrorStyleStop, "Address", "Type a whole number from "+
		strconv.FormatInt(bounds[0], 10)+" to "+strconv.FormatInt(bounds[1], 10)+", or leave it empty.")
	for _, rule := range []*excelize.DataValidation{kind, ied, point, address} {
		if err := book.AddDataValidation(PointsSheet, rule); err != nil {
			return err
		}
	}
	return nil
}
