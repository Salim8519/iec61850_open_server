package scl

import (
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
)

// ADR-0095: data classes come from the stored SCL's DOTypes, keyed like the model's refs.
func TestDataClassesFollowModelReferences(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "corpus", "template_BayController.scd"))
	if err != nil {
		t.Fatal(err)
	}
	classes, err := DataClasses(source, "BayController")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"BayControllerCTRL/XCBR1.Pos":   "DPC",
		"BayControllerCTRL/XCBR1.Loc":   "SPS",
		"BayControllerCTRL/LLN0.Beh":    "INS",
		"BayControllerCTRL/GGIO1.AnIn1": "MV",
	}
	for ref, class := range want {
		if classes[ref] != class {
			t.Errorf("%s: got %q, want %q", ref, classes[ref], class)
		}
	}
	// Every class key is a data object the importer also built signals under.
	models, err := (&Importer{Validator: loadSeed(t)}).Import(source, "template.scd", SelectAll)
	if err != nil {
		t.Fatal(err)
	}
	refs := map[string]bool{}
	for _, item := range get(models[0], "signals").([]any) {
		refs[get(item.(*pyjson.Object), "ref").(string)] = true
	}
	for _, ref := range []string{"BayControllerCTRL/XCBR1.Pos.stVal", "BayControllerCTRL/GGIO1.AnIn1.mag.f"} {
		if !refs[ref] {
			t.Errorf("model has no signal %s, so class keys would not match", ref)
		}
	}
	other, err := DataClasses(source, "NoSuchIED")
	if err != nil || len(other) != 0 {
		t.Fatalf("unknown IED: %v %v", other, err)
	}
	cached, err := CachedDataClasses("id-1", "BayController", string(source))
	if err != nil || cached["BayControllerCTRL/XCBR1.Pos"] != "DPC" {
		t.Fatalf("cached: %v %v", cached["BayControllerCTRL/XCBR1.Pos"], err)
	}
}
