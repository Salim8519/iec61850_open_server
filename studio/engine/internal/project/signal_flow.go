package project

import (
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/flow"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// projects/signal_flow.py: the declared signal-flow graph of a stored SCL document.

// FlowSources is flow_sources(service): [{id, filename, ieds}].
func FlowSources(project *pyjson.Object) []any {
	result := []any{}
	for _, entry := range StoredSources(project) {
		result = append(result, pyjson.NewObject().Set("id", entry.ID).Set("filename", entry.Filename).
			Set("ieds", entry.IEDNames()))
	}
	return result
}

// FlowGraph is graph(service, source, ied): ied "" is None.
func FlowGraph(project *pyjson.Object, source, ied string) (*pyjson.Object, error) {
	entry, err := RequireSource(project, source)
	if err != nil {
		return nil, err
	}
	loaded := map[string]bool{}
	for _, item := range items(valueOf(project, "ieds")) {
		loaded[text(object(item), "name")] = true
	}
	declared, err := scl.ReadFlow(entry.Text)
	if err != nil {
		return nil, err
	}
	result := flow.BuildGraph(declared, loaded, ied)
	if ied != "" {
		found := false
		for _, item := range items(valueOf(result, "ieds")) {
			if text(object(item), "name") == ied {
				found = true
			}
		}
		if !found {
			result.Set("ieds", []any{pyjson.NewObject().Set("name", ied).Set("desc", "").Set("manufacturer", "").
				Set("loaded", loaded[ied])})
		}
	}
	answer := pyjson.NewObject().Set("source", entry.ID).Set("filename", entry.Filename)
	for _, key := range result.Keys() {
		value, _ := result.Get(key)
		answer.Set(key, value)
	}
	return answer, nil
}

// EdgeCSV is edge_csv(service, source, edge_id): the signal rows of one edge and the
// download name "<publisher>-to-<subscriber>-<type>.csv".
func EdgeCSV(project *pyjson.Object, source, edgeID string) (string, string, error) {
	result, err := FlowGraph(project, source, "")
	if err != nil {
		return "", "", err
	}
	var edge *pyjson.Object
	for _, item := range items(valueOf(result, "edges")) {
		if text(object(item), "id") == edgeID {
			edge = object(item)
			break
		}
	}
	if edge == nil {
		return "", "", apperr.Coded("Unknown signal-flow link", "not_found")
	}
	ref := ""
	for _, item := range items(valueOf(result, "blocks")) {
		if text(object(item), "id") == text(edge, "block") {
			ref = text(object(item), "ref")
			break
		}
	}
	var out strings.Builder
	csvRow(&out, "type", "publisher", "block", "subscriber", "source", "input", "description")
	for _, item := range items(valueOf(edge, "signals")) {
		signal := object(item)
		csvRow(&out, text(edge, "type"), text(edge, "publisher"), ref, text(edge, "subscriber"),
			text(signal, "source"), text(signal, "int_addr"), text(signal, "desc"))
	}
	name := text(edge, "publisher") + "-to-" + text(edge, "subscriber") + "-" + text(edge, "type") + ".csv"
	return out.String(), name, nil
}

// csvRow is csv.writer(lineterminator="\n").writerow for text fields (QUOTE_MINIMAL).
func csvRow(out *strings.Builder, fields ...string) {
	for index, field := range fields {
		if index > 0 {
			out.WriteByte(',')
		}
		if strings.ContainsAny(field, ",\"\r\n") {
			out.WriteString(`"` + strings.ReplaceAll(field, `"`, `""`) + `"`)
		} else {
			out.WriteString(field)
		}
	}
	out.WriteByte('\n')
}

func valueOf(object *pyjson.Object, key string) any {
	if object == nil {
		return nil
	}
	value, _ := object.Get(key)
	return value
}
