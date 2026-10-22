package scl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func typedSample(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, "studio", "tests", "fixtures", "typed_controls", "SampleTyped.scd"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type validSchema struct{}

func (validSchema) Validate([]byte, string) (SchemaVerdict, error) {
	return SchemaVerdict{Status: "valid"}, nil
}

func TestExtendedControlsListEveryTypedClassAndModel(t *testing.T) {
	controls, err := ExtendedControls(typedSample(t), "SampleTyped")
	if err != nil {
		t.Fatal(err)
	}
	if len(controls) != 24 {
		t.Fatalf("want 24 typed controls (6 classes x 4 models), got %d", len(controls))
	}
	byRef := map[string]*pyjson.Object{}
	for _, item := range controls {
		entry := item.(*pyjson.Object)
		byRef[get(entry, "ref").(string)] = entry
	}
	expect := func(ref, key string, want string) {
		t.Helper()
		data, _ := pyjson.Dumps(get(byRef[ref], key))
		if string(data) != want {
			t.Errorf("%s %s = %s, want %s", ref, key, data, want)
		}
	}
	expect("SampleTypedCTRL/GGIO1.ISCSO1", "cdc", `"INC"`)
	expect("SampleTypedCTRL/GGIO1.ISCSO1", "model", `1`)
	expect("SampleTypedCTRL/GGIO1.ISCSO1", "max", `10`)
	expect("SampleTypedCTRL/GGIO1.ISCSO1", "step", `1`)
	expect("SampleTypedCTRL/GGIO2.Mod", "labels", `{"on": 1, "on-blocked": 2, "test": 3, "test/blocked": 4, "off": 5}`)
	expect("SampleTypedCTRL/GGIO2.Mod", "sbo_timeout_ms", `3000`)
	expect("SampleTypedCTRL/GGIO3.AnOut1", "status", `"SampleTypedCTRL/GGIO3.AnOut1.mxVal.f"`)
	expect("SampleTypedCTRL/GGIO3.AnOut1", "ctl_val_ref", `"SampleTypedCTRL/GGIO3.AnOut1.Oper.ctlVal.f"`)
	expect("SampleTypedCTRL/GGIO3.AnOut1", "max", `100.0`)
	expect("SampleTypedCTRL/ATCC4.TapChg", "model", `4`)
	expect("SampleTypedCTRL/ATCC4.TapChg", "value_kind", `"step"`)
	expect("SampleTypedCTRL/ATCC4.TapChg", "min", `-64`)
	expect("SampleTypedCTRL/ATCC4.TapChg", "trans_ind", `"SampleTypedCTRL/ATCC4.TapChg.valWTr.transInd"`)
	expect("SampleTypedCTRL/ATCC1.TapSet", "min", `1`)
	expect("SampleTypedCTRL/ATCC1.TapSet", "max", `17`)
	expect("SampleTypedCTRL/ATCC2.BndSet", "step", `2.5`)
	data, _ := pyjson.Dumps(byRef["SampleTypedCTRL/GGIO1.ISCSO1"])
	want := `{"ref": "SampleTypedCTRL/GGIO1.ISCSO1", "status": "SampleTypedCTRL/GGIO1.ISCSO1.stVal", "cdc": "INC", ` +
		`"ctl_val_kind": "INT32", "ctl_val_ref": "SampleTypedCTRL/GGIO1.ISCSO1.Oper.ctlVal", "value_kind": "int", ` +
		`"model": 1, "sbo_timeout_ms": 3000, "min": 0, "max": 10, "step": 1, "labels": {}, "trans_ind": null}`
	if string(data) != want {
		t.Errorf("shape:\n got %s\nwant %s", data, want)
	}
}

// The IED model (and so its "controls" and warnings) is exactly what it was without typed
// controls: the extended list is never part of it.
func TestImportKeepsControlsAndHasNoExtendedKey(t *testing.T) {
	importer := &Importer{Validator: validSchema{}}
	models, err := importer.Import(typedSample(t), "SampleTyped.scd", SelectAll)
	if err != nil {
		t.Fatal(err)
	}
	model := models[0]
	if _, present := model.Get("extended_controls"); present {
		t.Fatal("extended_controls must not be stored in the IED model")
	}
	if controls := get(model, "controls").([]any); len(controls) != 0 {
		t.Fatalf("no SPC/DPC in the sample, got %v", controls)
	}
	for _, warning := range get(model, "warnings").([]any) {
		if strings.Contains(warning.(string), "is not exposed") {
			t.Fatalf("typed controls must not add warnings: %v", warning)
		}
	}
}

func TestExtendedControlsSkipStatusOnlyAndUnknownIed(t *testing.T) {
	template, err := MakeTemplate("Plain")
	if err != nil {
		t.Fatal(err)
	}
	controls, err := ExtendedControls(template, "Plain")
	if err != nil || len(controls) != 0 {
		t.Fatalf("the SPC/DPC template has no typed controls: %v %v", controls, err)
	}
	if controls, err := ExtendedControls(typedSample(t), "Missing"); err != nil || len(controls) != 0 {
		t.Fatalf("unknown IED: %v %v", controls, err)
	}
	first, _ := CachedExtendedControls("x", "SampleTyped", string(typedSample(t)))
	second, _ := CachedExtendedControls("x", "SampleTyped", string(typedSample(t)))
	if len(first) != 24 || &first[0] != &second[0] {
		t.Fatal("cached list must be reused")
	}
}
