package scl

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// Item 5: a DA whose template states no trigger option gets the IEC 61850-7-3 ones.
func TestStandardTriggerOptionsWhenTheTemplateStatesNone(t *testing.T) {
	model := mustImport(t, synthetic(nil))
	cfg := get(model, "cfg").(string)
	for _, check := range []struct{ header, prefix, want string }{
		{"DO(Ind1 0){", "DA(stVal ", "DA(stVal 0 0 0 1 0)"},          // ST stVal: dchg
		{"DO(Ind1 0){", "DA(q ", "DA(q 0 23 0 2 0)"},                 // ST q: qchg
		{"DO(Ind1 0){", "DA(t ", "DA(t 0 22 0 0 0)"},                 // t: none
		{"DO(AnIn1 0){", "DA(mag ", "DA(mag 0 27 1 5 0){"},           // MX mag: dchg + dupd
		{"DO(AnIn1 0){", "DA(f ", "DA(f 0 10 1 5 0)"},                // BDA inherits the DA's
		{"DO(SPCSO1 0){", "DA(stVal ", "DA(stVal 0 0 0 1 0)"},        // stated in the file: kept
		{"DO(SPCSO1 0){", "DA(ctlModel ", "DA(ctlModel 0 12 4 0 0)"}, // CF: no default
	} {
		if line := cfgAfter(t, cfg, check.header, check.prefix); !strings.HasPrefix(line, check.want) {
			t.Errorf("%s %s: %q, want %q", check.header, check.prefix, line, check.want)
		}
	}
	explicit := mustImport(t, synthetic(map[string]string{"{sps}": `<DA name="cnt" bType="INT32" fc="ST" dchg="false"/>`}))
	if line := cfgAfter(t, get(explicit, "cfg").(string), "DO(Ind1 0){", "DA(cnt "); !strings.HasPrefix(line, "DA(cnt 0 3 0 0 0)") {
		t.Errorf("an explicit dchg=false must be kept: %q", line)
	}
	if note := warningWith(model, "IEC 61850-7-3 trigger options"); !strings.HasPrefix(note, "12 attribute(s)") {
		t.Errorf("note %q", note)
	}
}

// Item 6: an import error names the IED and the SCL path; preview flags the IED beforehand.
func TestImportErrorsNameTheIedAndPath(t *testing.T) {
	source := synthetic(map[string]string{"{sps}": `<DA name="dirX" bType="Enum" fc="ST"/>`})
	_, err := importOne(t, source)
	want := "IED R1 at R1LD0/GGIO1.Ind1.dirX: bType is Enum but the attribute names no EnumType"
	if err == nil || !strings.HasPrefix(err.Error(), want) || strings.Contains(err.Error(), "None") {
		t.Fatalf("error %v, want %q", err, want)
	}
	models, err := Preview(source)
	if err != nil {
		t.Fatal(err)
	}
	reason := get(models[0], "unavailable_reason").(string)
	if !strings.HasPrefix(reason, "Cannot be imported at R1LD0/GGIO1.Ind1.dirX: bType is Enum") {
		t.Fatalf("preview reason %q", reason)
	}
	if models, _ := Preview(synthetic(nil)); get(models[0], "unavailable_reason") != "" {
		t.Fatal("an importable IED must stay selectable")
	}
}

// Item 7: a dataset member without doName leaves only that dataset (and its blocks) out.
func TestFCDAWithoutDoNameLeavesOnlyItsDatasetOut(t *testing.T) {
	ln0 := `<DataSet name="Whole"><FCDA ldInst="LD0" lnClass="GGIO" lnInst="1" fc="ST"/></DataSet>
	  <DataSet name="Good"><FCDA ldInst="LD0" lnClass="GGIO" lnInst="1" doName="Ind1" daName="stVal" fc="ST"/></DataSet>
	  <ReportControl name="rWhole" datSet="Whole" confRev="1"><RptEnabled max="1"/></ReportControl>
	  <ReportControl name="rGood" datSet="Good" confRev="1"><RptEnabled max="1"/></ReportControl>
	  <GSEControl name="gWhole" datSet="Whole" appID="W"/>`
	model := mustImport(t, synthetic(map[string]string{"{ln0}": ln0}))
	datasets, _ := pyjson.Dumps(get(model, "datasets"))
	reports, _ := pyjson.Dumps(get(model, "reports"))
	if strings.Contains(string(datasets), "Whole") || !strings.Contains(string(datasets), `"Good"`) ||
		strings.Contains(string(reports), "rWhole") || !strings.Contains(string(reports), "rGood01") {
		t.Fatalf("datasets %s reports %s", datasets, reports)
	}
	if strings.Contains(get(model, "cfg").(string), "DS(Whole)") {
		t.Fatal("the unusable dataset reached the native model")
	}
	for _, text := range []string{"Dataset LD0/LLN0.Whole has a member without doName (LD0/GGIO1 [ST]",
		"Report block LD0/LLN0.rWhole uses dataset Whole", "GOOSE block R1/LD0/gWhole uses dataset Whole"} {
		if warningWith(model, text) == "" {
			t.Errorf("missing warning %q", text)
		}
	}
}

// Item 9: commands ScadaBench cannot operate are listed in a note, never dropped silently.
func TestCommandsThatCannotBeOperatedAreNoted(t *testing.T) {
	model := mustImport(t, synthetic(map[string]string{"{spc}": "SPC_NOSTVAL"}))
	if len(get(model, "controls").([]any)) != 0 {
		t.Fatal("a command without status is not operable")
	}
	if note := warningWith(model, "have no status attribute (stVal)"); !strings.Contains(note, "R1LD0/GGIO1.SPCSO1") {
		t.Fatalf("command-only note %q", note)
	}
	odd := mustImport(t, synthetic(map[string]string{"{spc}": "BSC_BOOL"}))
	if note := warningWith(odd, "use a value or status type"); !strings.Contains(note, "R1LD0/GGIO1.SPCSO1 (BSC: valWTr.posVal") &&
		!strings.Contains(note, "R1LD0/GGIO1.SPCSO1 (BSC: Oper.ctlVal is BOOLEAN)") {
		t.Fatalf("odd command note %q", note)
	}
}

// Item 13: a GSE without APPID gets one "missing APPID" warning, not two contradictory ones.
func TestGSEWithoutAPPIDIsOneWarning(t *testing.T) {
	gse := `<GSE ldInst="LD0" cbName="g1"><Address><P type="MAC-Address">01-0C-CD-01-00-01</P></Address></GSE>`
	ln0 := `<DataSet name="Good"><FCDA ldInst="LD0" lnClass="GGIO" lnInst="1" doName="Ind1" daName="stVal" fc="ST"/></DataSet>
	  <GSEControl name="g1" datSet="Good" appID="G1"/>`
	model := mustImport(t, synthetic(map[string]string{"{gse}": gse, "{ln0}": ln0}))
	if warningWith(model, "GOOSE address R1/LD0/g1 has no APPID") == "" || warningWith(model, "unreadable APPID") != "" ||
		warningWith(model, "has no network address") != "" {
		t.Fatalf("warnings %v", get(model, "warnings"))
	}
}
