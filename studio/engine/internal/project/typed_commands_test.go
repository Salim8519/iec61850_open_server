package project

import (
	"strings"
	"testing"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Typed command rows and remembered command options (ADR-0108).

// typedIEDs gives ied0 a typed control list as ADR-0104's SCL reader gives it: an APC, an INC
// and a BAC (an unknown class LTRK keeps the class check covered).
func typedIEDs(t *testing.T) []*pyjson.Object {
	t.Helper()
	original := extendedControlsOf
	extendedControlsOf = func(ied *pyjson.Object) any {
		if text(ied, "id") != "ied0" {
			return nil
		}
		return []any{o("ref", "LD0/APCSO1", "cdc", "APC", "model", int64(4)),
			o("ref", "LD0/ISCSO1", "cdc", "INC", "model", int64(1)),
			o("ref", "LD0/BACSO1", "cdc", "BAC", "model", int64(1)),
			o("ref", "LD0/LTRK1", "cdc", "LTRK", "model", int64(1))}
	}
	t.Cleanup(func() { extendedControlsOf = original })
	return testIEDs()
}

func typedRow(id, ref string, extra ...any) *pyjson.Object {
	entry := o("id", id, "label", "Command "+id, "type", "command", "control", o("ied_id", "ied0", "ref", ref))
	for index := 0; index < len(extra); index += 2 {
		entry.Set(extra[index].(string), extra[index+1])
	}
	return entry
}

func TestPointPanelTypedCommands(t *testing.T) {
	ieds := typedIEDs(t)
	check := func(rows ...any) error {
		_, err := ValidateDiagram(listDiagram(PointPanelKind, panelData("ied0", rows...)), ieds)
		return err
	}
	if err := check(typedRow("a", "LD0/APCSO1", "unit", "kV"), typedRow("i", "LD0/ISCSO1"), typedRow("b", "LD0/BACSO1"),
		command("c", "ied0", 1)); err != nil {
		t.Fatalf("APC with a unit, INC, BAC and SPC are accepted: %v", err)
	}
	cases := map[string]*pyjson.Object{
		"is a INC command, so it cannot have a unit": typedRow("i", "LD0/ISCSO1", "unit", "taps"),
		"but LD0/LTRK1 is LTRK":                      typedRow("l", "LD0/LTRK1"),
		"uses a control this IED does not have":      typedRow("x", "LD0/NOPE1"),
		"cannot have a unit, scale":                  typedRow("s", "LD0/APCSO1", "scale", 2.0),
	}
	for want, row := range cases {
		if err := check(row); err == nil || !strings.Contains(studioMessage(err), want) {
			t.Errorf("want %q, got %v", want, err)
		}
	}
	if err := check(command("c", "ied0", 1, "unit", "A")); err == nil || !strings.Contains(studioMessage(err), "SPC command") {
		t.Errorf("an SPC command row has no unit, got %v", err)
	}
}

func TestCommandOptionsOnSwitchesOnly(t *testing.T) {
	options := o("test", true, "orcat", int64(3), "orident", "bench")
	result, err := ValidateDiagram(listDiagram("CB", o("command_options", options)), testIEDs())
	if err != nil {
		t.Fatal(err)
	}
	stored := object(pyval.Lookup(firstData(t, result), "command_options", nil))
	if got := dumps(t, stored); got != `{"test": true, "interlock_check": false, "synchro_check": false, "orcat": 3, "orident": "bench"}` {
		t.Fatalf("stored options %s", got)
	}
	plain, err := ValidateDiagram(listDiagram("CB", o()), testIEDs())
	if err != nil {
		t.Fatal(err)
	}
	if _, present := firstData(t, plain).Get("command_options"); present {
		t.Fatal("a breaker that never sent options keeps its saved bytes")
	}
	if _, err := ValidateDiagram(listDiagram("RELAY", o("command_options", o())), testIEDs()); err == nil ||
		!strings.Contains(studioMessage(err), "Only a breaker, disconnector or earth switch") {
		t.Fatalf("other kinds refuse options, got %v", err)
	}
	for _, bad := range []*pyjson.Object{o("orcat", int64(9)), o("orident", strings.Repeat("x", 65)), o("extra", true)} {
		if _, err := ValidateDiagram(listDiagram("DS", o("command_options", bad)), testIEDs()); err == nil {
			t.Errorf("options %v must be refused", bad)
		}
	}
}
