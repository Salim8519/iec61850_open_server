package health

import (
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// columns is COLUMNS: the export columns in order.
func columns() [9]string {
	return [9]string{"ied", "role", "area", "item", "result", "message", "observed_at", "source", "stale"}
}

// resultWord is WORDS.get(verdict): the word a verdict shows as in exports.
func resultWord(verdict any) (string, bool) {
	switch pyv.Str(verdict) {
	case "ok":
		return "OK", true
	case "warning":
		return "Check", true
	case "problem":
		return "Problem", true
	case "not_checked":
		return "Not checked", true
	}
	return "", false
}

// exportRows is rows(health): one row per check in COLUMNS order (missing keys are nil).
func exportRows(health *pyjson.Object) [][]any {
	var rows [][]any
	for _, entry := range pyv.Objects(pyv.GetList(health, "ieds")) {
		for _, check := range pyv.Objects(pyv.GetList(entry, "checks")) {
			values := []any{pyv.Get(entry, "name"), pyv.Get(entry, "role")}
			names := columns()
			for _, key := range names[2:] {
				values = append(values, pyv.Get(check, key))
			}
			rows = append(rows, values)
		}
	}
	return rows
}

// CSVText is csv_text: csv.DictWriter output with "\n" line ends. None is written as an
// empty field; other values as Python's str().
func CSVText(health *pyjson.Object) string {
	var out strings.Builder
	header := columns()
	out.WriteString(strings.Join(header[:], ",") + "\n")
	for _, values := range exportRows(health) {
		for index, value := range values {
			if index > 0 {
				out.WriteByte(',')
			}
			if value != nil {
				out.WriteString(csvField(pyv.PyStr(value)))
			}
		}
		out.WriteByte('\n')
	}
	return out.String()
}

// CSVField quotes one field like Python's csv module (default dialect, QUOTE_MINIMAL);
// other CSV downloads of the live area use it too.
func CSVField(text string) string { return csvField(text) }

// csvField applies the csv module's QUOTE_MINIMAL rule for the default dialect: quote a
// field that contains the delimiter, the quote character, "\r" or "\n", doubling quotes.
func csvField(text string) string {
	if !strings.ContainsAny(text, ",\"\r\n") {
		return text
	}
	return `"` + strings.ReplaceAll(text, `"`, `""`) + `"`
}

// htmlEscape is html.escape(text, quote=True).
func htmlEscape(text string) string {
	var out strings.Builder
	for _, r := range text {
		switch r {
		case '&':
			out.WriteString("&amp;")
		case '<':
			out.WriteString("&lt;")
		case '>':
			out.WriteString("&gt;")
		case '"':
			out.WriteString("&quot;")
		case '\'':
			out.WriteString("&#x27;")
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// HTMLReport is html_report: a printable, self-contained snapshot; every value is escaped
// and nothing is loaded from the network.
func HTMLReport(health *pyjson.Object, projectName, runID string) string {
	var body strings.Builder
	for _, values := range exportRows(health) {
		body.WriteString("<tr>")
		for index, value := range values {
			if columns()[index] == "result" {
				if word, known := resultWord(value); known {
					value = word
				}
			}
			text := ""
			if value != nil {
				text = pyv.PyStr(value)
			}
			body.WriteString("<td>" + htmlEscape(text) + "</td>")
		}
		body.WriteString("</tr>")
	}
	var head strings.Builder
	for _, name := range columns() {
		head.WriteString("<th>" + capitalize(strings.ReplaceAll(name, "_", " ")) + "</th>")
	}
	verdict, known := resultWord(pyv.Get(health, "verdict"))
	if !known {
		verdict = pyv.PyStr(pyv.Get(health, "verdict"))
	}
	return "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\">" +
		"<title>Station health - " + htmlEscape(projectName) + "</title><style>" +
		"body{font:14px system-ui,sans-serif;margin:24px;color:#0f172a}table{border-collapse:collapse;width:100%}" +
		"th,td{border:1px solid #a9b1bd;padding:6px 8px;text-align:left;vertical-align:top}" +
		"th{background:#eef1f5}</style></head><body>" +
		"<h1>Station health: " + htmlEscape(projectName) + "</h1>" +
		"<p>Overall: <strong>" + verdict + "</strong> · checked " + htmlEscape(pyv.PyStr(pyv.Get(health, "checked_at"))) +
		" · run " + htmlEscape(runID) + "</p>" +
		"<table><thead><tr>" + head.String() + "</tr></thead><tbody>" + body.String() + "</tbody></table></body></html>"
}

// capitalize is str.capitalize for the ASCII column names: first letter upper, rest lower.
func capitalize(text string) string {
	if text == "" {
		return text
	}
	return strings.ToUpper(text[:1]) + strings.ToLower(text[1:])
}
