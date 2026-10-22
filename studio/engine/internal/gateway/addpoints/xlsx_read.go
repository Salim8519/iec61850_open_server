package addpoints

import (
	"bytes"
	"errors"

	"github.com/xuri/excelize/v2"

	"studio-engine/internal/gateway"
)

// ErrNotWorkbook is the plain answer for a file that is not an Excel workbook.
var ErrNotWorkbook = errors.New("This file is not an Excel workbook (.xlsx). Download the template, fill it in " +
	"Excel and upload it again.")

// ReadWorkbook reads the cells of a filled template: its Points sheet, else the first sheet.
// Formulas give their last saved value; rows beyond the point limit are not read.
func ReadWorkbook(data []byte) ([][]string, error) {
	book, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, ErrNotWorkbook
	}
	defer book.Close()
	sheet := PointsSheet
	if index, err := book.GetSheetIndex(PointsSheet); err != nil || index < 0 {
		sheets := book.GetSheetList()
		if len(sheets) == 0 {
			return nil, ErrNotWorkbook
		}
		sheet = sheets[0]
	}
	rows, err := book.GetRows(sheet)
	if err != nil {
		return nil, ErrNotWorkbook
	}
	if len(rows) > gateway.MaxPoints+1 {
		rows = rows[:gateway.MaxPoints+1]
	}
	return rows, nil
}
