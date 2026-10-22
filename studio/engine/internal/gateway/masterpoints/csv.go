package masterpoints

import (
	"bytes"
	"encoding/csv"
	"strconv"

	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// CSVHeader is the export's title row; PlanText reads every title back (a round trip).
var CSVHeader = []string{"Name", "Address", "Type", "Scale", "Offset", "Unit", "SBO", "Value", "Word order"}

// ExportCSV is the master point list as CSV (UTF-8 with a byte order mark, so Excel opens it).
func ExportCSV(protocol string, list []any) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("\xef\xbb\xbf")
	writer := csv.NewWriter(&buffer)
	writer.Write(CSVHeader)
	for _, raw := range list {
		point, ok := raw.(*pyjson.Object)
		if !ok {
			continue
		}
		label := text(point, "type")
		if entry, known := addpoints.MasterEntry(protocol, label); known {
			label = addpoints.MasterLabel(entry)
		}
		selected, value := "", ""
		if text(point, "command") != "" {
			selected = "no"
			if field(point, "select") == true {
				selected = "yes"
			}
			value = pytext.Str(field(point, "value"))
		}
		order := ""
		if field(point, "registers") == int64(2) {
			order = text(point, "word_order")
		}
		writer.Write([]string{text(point, "name"), strconv.FormatInt(Address(point), 10), label,
			numberText(field(point, "scale")), numberText(field(point, "offset")), text(point, "unit"), selected, value, order})
	}
	writer.Flush()
	return buffer.Bytes()
}

func numberText(raw any) string {
	value, _ := pytext.Float(raw)
	return strconv.FormatFloat(value, 'g', -1, 64)
}
