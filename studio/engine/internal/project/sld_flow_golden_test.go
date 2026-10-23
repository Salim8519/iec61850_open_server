package project

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/flow"
	"studio-engine/internal/project/sld"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// goldenError is golden_common.error_dict for the errors these functions raise.
func goldenError(err error) string {
	result := pyjson.NewObject()
	var app *apperr.Error
	if errors.As(err, &app) && app.Kind == apperr.Studio {
		details := app.Details
		if details == nil {
			details = []any{}
		}
		result.Set("type", "StudioError").Set("message", app.Message).Set("code", app.Code).Set("details", details)
	} else {
		result.Set("type", "ValueError").Set("message", err.Error()).Set("code", nil).Set("details", nil)
	}
	data, _ := pyjson.Dumps(pyjson.NewObject().Set("error", result))
	return string(data)
}

func goldenDumps(value any) string {
	data, _ := pyjson.Dumps(value)
	return string(data)
}

// TestSLDAndFlowMatchPython replays golden/sld_flow.json (scl/testdata/sld_flow_gen.py):
// read_substation, build_diagram, bind, read_flow + build_graph and edge_csv.
func TestSLDAndFlowMatchPython(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "scl", "testdata", "golden", "sld_flow.json"))
	if err != nil {
		t.Skip("sld_flow golden missing")
	}
	decoded, _ := pyjson.Decode(data)
	compared, failures := 0, 0
	check := func(name, got string, want any) {
		compared++
		if wanted := goldenDumps(want); got != wanted {
			failures++
			if failures <= 8 {
				t.Errorf("%s:\n got  %.700s\n want %.700s", name, got, wanted)
			}
		}
	}
	for _, raw := range decoded.([]any) {
		entry := raw.(*pyjson.Object)
		name := text(entry, "name")
		source := text(entry, "text")
		if path := text(entry, "case"); path != "" {
			payload, err := os.ReadFile(filepath.Join("..", "..", "..", "..", filepath.FromSlash(path)))
			if err != nil {
				t.Fatal(err)
			}
			source = strings.TrimPrefix(string(payload), "\xef\xbb\xbf")
		}
		topology, err := scl.ReadSubstation(source)
		if err != nil {
			check(name+" substation", goldenError(err), get(entry, "substation"))
		} else {
			check(name+" substation", goldenDumps(topology.JSON()), get(entry, "substation"))
			built, err := sld.BuildDiagram(topology, "src")
			if err != nil {
				check(name+" diagram", goldenError(err), get(entry, "diagram"))
			} else {
				check(name+" diagram", goldenDumps(built.JSON()), get(entry, "diagram"))
				ieds := []*pyjson.Object{}
				for _, ied := range items(get(entry, "ieds")) {
					ieds = append(ieds, object(ied))
				}
				bindings := []any{}
				for _, node := range built.Nodes {
					if item := built.Equipment[text(object(node), "id")]; item != nil {
						kind := text(object(get(object(node), "data")), "kind")
						state, control, reasons := sld.Bind(kind, item.LNodes, ieds, topology.Domains, nil)
						listed := []any{}
						for _, reason := range reasons {
							listed = append(listed, reason)
						}
						bindings = append(bindings, []any{orNone(state), orNone(control), listed})
					}
				}
				check(name+" bindings", goldenDumps(bindings), get(entry, "bindings"))
			}
		}
		checkFlow(t, name, source, entry, check)
	}
	t.Logf("%d Python answers compared, %d differ", compared, failures)
}

func checkFlow(t *testing.T, name, source string, entry *pyjson.Object, check func(string, string, any)) {
	for _, raw := range items(get(entry, "graphs")) {
		item := object(raw)
		ied, _ := get(item, "ied").(string)
		declared, err := scl.ReadFlow(source)
		var loaded map[string]bool
		if err == nil {
			loaded = map[string]bool{}
			if len(declared.IEDs) > 0 {
				loaded[text(declared.IEDs[0], "name")] = true
			}
		}
		want, ok := item.Get("result")
		if !ok {
			want = pyjson.NewObject().Set("error", get(item, "error"))
		}
		if err != nil {
			check(name+" graph "+ied, goldenError(err), want)
		} else {
			check(name+" graph "+ied, goldenDumps(flow.BuildGraph(declared, loaded, ied)), want)
		}
	}
	for _, raw := range items(get(entry, "csv")) {
		item := object(raw)
		ieds := []any{}
		for _, loaded := range items(get(item, "loaded")) {
			ieds = append(ieds, pyjson.NewObject().Set("name", loaded).Set("filename", "f.scd").Set("source", source))
		}
		project := pyjson.NewObject().Set("ieds", ieds)
		csvText, filename, err := EdgeCSV(project, SourceID(source), stringOf(get(item, "edge")))
		if want, ok := item.Get("result"); ok {
			check(name+" csv", goldenDumps([]any{csvText, filename}), want)
		} else {
			check(name+" csv", goldenError(err), pyjson.NewObject().Set("error", get(item, "error")))
		}
	}
}

func orNone(value *pyjson.Object) any {
	if value == nil {
		return nil
	}
	return value
}

func stringOf(value any) string {
	text, _ := value.(string)
	return text
}
