package project

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// A value row may read a whole-number status (ST INT…), such as a tap position or a counter
// (change 0173); an ST float or a setting (CF) is still refused in words.
func TestValueRowReadsWholeNumberStatus(t *testing.T) {
	ied := o("id", "tr", "name", "TR1", "controls", []any{}, "signals", []any{
		o("ref", "LD0/ATCC1.TapChg.valWTr.posVal", "kind", "INT8", "fc", "ST", "writable", true),
		o("ref", "LD0/GGIO1.AnIn1.stVal", "kind", "FLOAT32", "fc", "ST", "writable", true),
		o("ref", "LD0/ATCC1.TapChg.ctlModel", "kind", "Enum", "fc", "CF", "writable", true)})
	row := func(ref string) any {
		return o("id", "v", "label", "Tap", "type", "value", "source", o("ied_id", "tr", "ref", ref, "report_ref", nil))
	}
	validate := func(ref string) error {
		_, err := ValidateDiagram(listDiagram(PointPanelKind, panelData("tr", row(ref))), []*pyjson.Object{ied})
		return err
	}
	if err := validate("LD0/ATCC1.TapChg.valWTr.posVal"); err != nil {
		t.Fatalf("a tap position is a value row: %v", err)
	}
	for _, ref := range []string{"LD0/GGIO1.AnIn1.stVal", "LD0/ATCC1.TapChg.ctlModel"} {
		if err := validate(ref); err == nil || !strings.Contains(err.Error(), "whole-number status (ST)") {
			t.Fatalf("%s is refused in words, got %v", ref, err)
		}
	}
}
