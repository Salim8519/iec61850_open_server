package scl

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// vendorTally counts what one manufacturer's files gave.
type vendorTally struct{ files, ieds, controls, typed int }

// TestEveryVendorControlIsUsable reads the commands of every public fixture and of the local
// vendor corpus (STUDIO_SCL_CORPUS: vendor_cids, test_cids; never committed) and checks each
// one can be sent as the CID describes it (ADR-0138): a control model 1 to 4, a feedback point
// of the same object in the model, and a select timeout that is a whole number of ms.
func TestEveryVendorControlIsUsable(t *testing.T) {
	tally, refused := map[string]*vendorTally{}, 0
	for _, path := range sclFiles(t) {
		source, err := os.ReadFile(path)
		if err != nil || len(source) > 100<<20 {
			continue
		}
		models, err := (&Importer{Validator: acceptAll{}}).Import(source, filepath.Base(path), SelectAll)
		if err != nil {
			refused++ // import diagnostics are covered by the import tests (ADR-0114)
			continue
		}
		makers := manufacturers(source)
		seen := map[string]bool{}
		for _, model := range models {
			name, _ := model.Get("name")
			maker := makers[name.(string)]
			if tally[maker] == nil {
				tally[maker] = &vendorTally{}
			}
			if !seen[maker] {
				seen[maker], tally[maker].files = true, tally[maker].files+1
			}
			tally[maker].ieds++
			signals := map[string]bool{}
			for _, raw := range items(model, "signals") {
				ref, _ := raw.Get("ref")
				signals[ref.(string)] = true
			}
			for _, control := range items(model, "controls") {
				tally[maker].controls++
				checkControl(t, path, control, signals)
			}
			typed, _ := ExtendedControls(source, name.(string))
			for _, raw := range typed {
				if control, ok := raw.(*pyjson.Object); ok {
					tally[maker].typed++
					checkModel(t, path, control)
				}
			}
		}
	}
	makers := make([]string, 0, len(tally))
	for maker := range tally {
		makers = append(makers, maker)
	}
	sort.Strings(makers)
	for _, maker := range makers {
		count := tally[maker]
		t.Logf("%-28q files %3d  IEDs %4d  on/off and open/close commands %5d  other commands %5d",
			maker, count.files, count.ieds, count.controls, count.typed)
	}
	t.Logf("%d files the importer refuses are left to the import tests", refused)
	if len(makers) == 0 {
		t.Fatal("no SCL files found")
	}
}

func checkControl(t *testing.T, path string, control *pyjson.Object, signals map[string]bool) {
	t.Helper()
	checkModel(t, path, control)
	ref, _ := control.Get("ref")
	status, _ := control.Get("status")
	text, _ := status.(string)
	if !signals[text] || !strings.HasPrefix(text, ref.(string)+".") {
		t.Errorf("%s %v: feedback %v is not a point of the same object", filepath.Base(path), ref, status)
	}
	if timeout, _ := control.Get("sbo_timeout_ms"); !wholeMs(timeout) {
		t.Errorf("%s %v: select timeout %v is not a whole number of ms", filepath.Base(path), ref, timeout)
	}
}

func checkModel(t *testing.T, path string, control *pyjson.Object) {
	t.Helper()
	model, _ := control.Get("model")
	if number, ok := model.(int64); !ok || number < 1 || number > 4 {
		ref, _ := control.Get("ref")
		t.Errorf("%s %v: control model %v is not 1 to 4", filepath.Base(path), ref, model)
	}
}

func wholeMs(value any) bool {
	number, ok := value.(int64)
	return ok && number >= 0
}

func items(model *pyjson.Object, key string) []*pyjson.Object {
	list, _ := model.Get(key)
	raw, _ := list.([]any)
	result := []*pyjson.Object{}
	for _, item := range raw {
		if object, ok := item.(*pyjson.Object); ok {
			result = append(result, object)
		}
	}
	return result
}

// manufacturers maps each IED name to its manufacturer attribute ("" when the file has none).
func manufacturers(source []byte) map[string]string {
	result := map[string]string{}
	if doc, err := ParseDocument(source, false); err == nil {
		for _, ied := range doc.Root.FindAll("IED") {
			result[ied.GetOr("name", "")] = ied.GetOr("manufacturer", "")
		}
	}
	return result
}

// Some vendor tools set ctlModel on the instance (DOI/DAI) and some write it as a number; the
// instance wins over the type in both forms, so the command is sent as the CID says.
func TestControlModelOnTheInstanceWins(t *testing.T) {
	for val, want := range map[string]int64{"sbo-with-enhanced-security": 4, "4": 4, "2": 2, "direct-with-enhanced-security": 3} {
		model := mustImport(t, synthetic(map[string]string{
			"{ggio}": `<DOI name="SPCSO1"><DAI name="ctlModel"><Val>` + val + `</Val></DAI></DOI>`}))
		found := false
		for _, control := range items(model, "controls") {
			if ref, _ := control.Get("ref"); ref == "R1LD0/GGIO1.SPCSO1" {
				found = true
				if got, _ := control.Get("model"); got != want {
					t.Errorf("ctlModel %q gives model %v, want %d", val, got, want)
				}
			}
		}
		if !found {
			t.Errorf("ctlModel %q: the command is missing", val)
		}
	}
}
