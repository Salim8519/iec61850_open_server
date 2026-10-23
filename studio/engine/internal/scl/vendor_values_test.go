package scl

import (
	"strings"
	"testing"
)

// cfgAfter is the first cfg line starting with prefix after the line equal to header.
func cfgAfter(t *testing.T, cfg, header, prefix string) string {
	t.Helper()
	lines := strings.Split(cfg, "\n")
	for index, line := range lines {
		if line != header {
			continue
		}
		for _, next := range lines[index+1:] {
			if strings.HasPrefix(next, prefix) {
				return next
			}
		}
	}
	t.Fatalf("no %q after %q", prefix, header)
	return ""
}

// Change 0185 item 1: Unicode255 and Octet values are text (34 vendor IEDs were refused).
func TestUnicodeAndOctetValuesAreText(t *testing.T) {
	model := mustImport(t, synthetic(map[string]string{"{lpld}": `bType="Unicode255" fc="DC"><Val>Feeder 1 protection</Val>`}))
	if got := get(signalOf(model, "R1LD0/LLN0.NamPlt.d"), "value"); got != "Feeder 1 protection" {
		t.Fatalf("Unicode255 value %#v", got)
	}
	if line := cfgAfter(t, get(model, "cfg").(string), "DO(NamPlt 0){", "DA(d "); !strings.HasSuffix(line, `="Feeder 1 protection";`) {
		t.Fatalf("Unicode255 cfg %q", line)
	}
	empty := mustImport(t, synthetic(map[string]string{"{lpld}": `bType="Unicode255" fc="DC">`}))
	if got := get(signalOf(empty, "R1LD0/LLN0.NamPlt.d"), "value"); got != "" {
		t.Fatalf("Unicode255 without Val %#v, want empty text", got)
	}
	if line := cfgAfter(t, get(empty, "cfg").(string), "DO(NamPlt 0){", "DA(d "); !strings.HasSuffix(line, `="";`) {
		t.Fatalf("Unicode255 without Val cfg %q", line)
	}
	octet := mustImport(t, synthetic(map[string]string{"{lpld}": `bType="Octet64" fc="DC"><Val>0A1B</Val>`}))
	if got := get(signalOf(octet, "R1LD0/LLN0.NamPlt.d"), "value"); got != "0A1B" {
		t.Fatalf("Octet64 value %#v", got)
	}
}

// Item 2: Dbpos values written as their standard names.
func TestDbposStandardNames(t *testing.T) {
	for name, want := range map[string]int64{"intermediate-state": 0, "off": 1, "on": 2, "bad-state": 3, "2": 2} {
		model := mustImport(t, synthetic(map[string]string{"{dpc}": "<Val>" + name + "</Val>"}))
		if got := get(signalOf(model, "R1LD0/XCBR1.Pos.stVal"), "value"); got != want {
			t.Errorf("Dbpos %q = %#v, want %d", name, got, want)
		}
	}
	_, err := importOne(t, synthetic(map[string]string{"{dpc}": "<Val>intermediate_state</Val>"}))
	if err == nil || !strings.Contains(err.Error(), "IED R1 at R1LD0/XCBR1.Pos.stVal:") ||
		!strings.Contains(err.Error(), "intermediate-state, off, on, bad-state") {
		t.Fatalf("non-standard Dbpos spelling: %v", err)
	}
}

// Item 3: LDevice@inst and FCDA@ldInst follow the XSD tLDInst pattern (a digit first).
func TestLogicalDeviceInstMayStartWithADigit(t *testing.T) {
	dataset := `<DataSet name="DS1"><FCDA ldInst="1" lnClass="GGIO" lnInst="1" doName="Ind1" daName="stVal" fc="ST"/></DataSet>`
	model := mustImport(t, synthetic(map[string]string{"{ld}": "1", "{ln0}": dataset}))
	if signalOf(model, "R11/GGIO1.Ind1.stVal") == nil || !strings.Contains(get(model, "cfg").(string), "LD(1){") {
		t.Fatal("LDevice inst 1 not imported")
	}
	if _, err := importOne(t, synthetic(map[string]string{"{ld}": "1-x"})); err == nil {
		t.Fatal("an inst outside tLDInst must stay refused")
	}
}

// Item 4: an Enum without Val reads the lowest ord of its EnumType.
func TestEnumWithoutValIsLowestOrd(t *testing.T) {
	model := mustImport(t, synthetic(nil))
	if got := get(signalOf(model, "R1LD0/LLN0.Beh.stVal"), "value"); got != int64(1) {
		t.Fatalf("Beh.stVal %#v, want on(1)", got)
	}
	if line := cfgAfter(t, get(model, "cfg").(string), "DO(Beh 0){", "DA(stVal "); !strings.HasSuffix(line, "=1;") {
		t.Fatalf("Beh cfg %q", line)
	}
	if got := get(signalOf(model, "R1LD0/XCBR1.Pos.ctlModel"), "value"); got != int64(0) {
		t.Fatalf("ctlModel with Val status-only %#v", got)
	}
}

// Item 8: a quote or line break the native model cannot hold is simulated visibly replaced.
func TestQuotedTextIsReplacedWithANote(t *testing.T) {
	model := mustImport(t, synthetic(map[string]string{"{lpld}": `bType="VisString255" fc="DC"><Val>Time to "Alarm"
next</Val>`}))
	if got := get(signalOf(model, "R1LD0/LLN0.NamPlt.d"), "value"); got != "Time to 'Alarm' next" {
		t.Fatalf("simulated text %#v", got)
	}
	if note := warningWith(model, "double quote or a line break"); !strings.Contains(note, "R1LD0/LLN0.NamPlt.d") {
		t.Fatalf("no note naming the attribute: %q", note)
	}
}
