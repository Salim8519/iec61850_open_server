package scl

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func mustValidation(status string, line int64, message string) *pyjson.Object {
	return pyjson.NewObject().Set("status", status).Set("schema", "SCL2007B4").
		Set("errors", []any{pyjson.NewObject().Set("line", line).Set("message", message)})
}

// stationSCL: PUB publishes GOOSE gcb1; SUB subscribes to it by srcCBName, supervises it with
// LGOS1 whose setSrcRef is the ObjectReference form, and takes two inputs from GW, an IED with
// no section in the file, without serviceType. BAD cannot be imported (Enum without type).
const stationSCL = `<?xml version="1.0" encoding="UTF-8"?>
<SCL xmlns="http://www.iec.ch/61850/2003/SCL" version="2007" revision="B" release="4">
  <Header id="station"/>
  <Communication><SubNetwork name="S1">
    <ConnectedAP iedName="PUB" apName="AP1"><Address><P type="IP">10.0.0.1</P></Address>
      <GSE ldInst="LD0" cbName="gcb1"><Address><P type="MAC-Address">01-0C-CD-01-00-01</P><P type="APPID">0001</P></Address></GSE>
    </ConnectedAP>
    <ConnectedAP iedName="SUB" apName="AP1"><Address><P type="IP">10.0.0.2</P></Address></ConnectedAP>
  </SubNetwork></Communication>
  <IED name="PUB"><AccessPoint name="AP1"><Server><Authentication/><LDevice inst="LD0">
    <LN0 lnClass="LLN0" inst="" lnType="LLN0_T">
      <DataSet name="DS1"><FCDA ldInst="LD0" lnClass="GGIO" lnInst="1" doName="Ind1" daName="stVal" fc="ST"/></DataSet>
      <GSEControl name="gcb1" datSet="DS1" appID="PUB_G1"/>
    </LN0>
    <LN lnClass="GGIO" inst="1" lnType="GGIO_T"/>
  </LDevice></Server></AccessPoint></IED>
  <IED name="SUB"><AccessPoint name="AP1"><Server><Authentication/><LDevice inst="LD0">
    <LN0 lnClass="LLN0" inst="" lnType="LLN0_T"/>
    <LN lnClass="GGIO" inst="1" lnType="GGIO_T"><Inputs>
      <ExtRef iedName="PUB" ldInst="LD0" lnClass="GGIO" lnInst="1" doName="Ind1" daName="stVal" serviceType="GOOSE" srcCBName="gcb1" srcLDInst="LD0" intAddr="in1"/>
      <ExtRef iedName="GW" ldInst="LD0" lnClass="GGIO" lnInst="1" doName="Ind1" daName="stVal" intAddr="in2"/>
      <ExtRef iedName="GW" ldInst="LD0" lnClass="GGIO" lnInst="1" doName="Ind2" daName="stVal" intAddr="in3"/>
    </Inputs></LN>
    <LN lnClass="LGOS" inst="1" lnType="LGOS_T"><DOI name="GoCBRef"><DAI name="setSrcRef"><Val>PUBLD0/LLN0.gcb1</Val></DAI></DOI></LN>
  </LDevice></Server></AccessPoint></IED>
  <IED name="BAD"><AccessPoint name="AP1"><Server><Authentication/><LDevice inst="LD0">
    <LN0 lnClass="LLN0" inst="" lnType="BAD_T"/>
  </LDevice></Server></AccessPoint></IED>
  <DataTypeTemplates>
    <LNodeType id="LLN0_T" lnClass="LLN0"><DO name="Beh" type="ENS_Beh"/></LNodeType>
    <LNodeType id="BAD_T" lnClass="LLN0"><DO name="Beh" type="ENS_NoType"/></LNodeType>
    <LNodeType id="GGIO_T" lnClass="GGIO"><DO name="Ind1" type="SPS_T"/></LNodeType>
    <LNodeType id="LGOS_T" lnClass="LGOS"><DO name="GoCBRef" type="ORG_T"/></LNodeType>
    <DOType id="ENS_Beh" cdc="ENS"><DA name="stVal" bType="Enum" fc="ST" type="BehKind"/></DOType>
    <DOType id="ENS_NoType" cdc="ENS"><DA name="stVal" bType="Enum" fc="ST"/></DOType>
    <DOType id="SPS_T" cdc="SPS"><DA name="stVal" bType="BOOLEAN" fc="ST" dchg="true"/><DA name="q" bType="Quality" fc="ST" qchg="true"/></DOType>
    <DOType id="ORG_T" cdc="ORG"><DA name="setSrcRef" bType="ObjRef" fc="SP" dchg="true"/></DOType>
    <EnumType id="BehKind"><EnumVal ord="1">on</EnumVal><EnumVal ord="5">off</EnumVal></EnumType>
  </DataTypeTemplates>
</SCL>
`

// Item 17: LGOS supervision binds when setSrcRef uses the ObjectReference form.
func TestLGOSAcceptsTheObjectReferenceForm(t *testing.T) {
	models, err := (&Importer{Validator: validSchema{}}).Import([]byte(stationSCL), "s.scd", Selection{Names: []any{"SUB"}})
	if err != nil {
		t.Fatal(err)
	}
	subscribes := get(get(models[0], "goose").(*pyjson.Object), "subscribes").([]any)
	if len(subscribes) != 1 || get(subscribes[0].(*pyjson.Object), "lgos") != "SUBLD0/LGOS1" {
		t.Fatalf("subscribes %v", subscribes)
	}
	if gocbMMSRef("PUBLD0/LLN0.gcb1") != "PUBLD0/LLN0$GO$gcb1" || gocbMMSRef("PUBLD0/LLN0$GO$gcb1") != "" ||
		gocbMMSRef("PUBLD0/GGIO1.Ind1") != "" {
		t.Fatal("gocbMMSRef converts only the GoCB ObjectReference form")
	}
}

// Item 14: inputs without serviceType from an IED that is not in the file are a problem.
func TestFlowFlagsInputsFromIedsOutsideTheFile(t *testing.T) {
	flow, err := ReadFlow(stationSCL)
	if err != nil {
		t.Fatal(err)
	}
	problems := flow.Subscriptions["SUB"].Problems
	if len(problems) != 1 || !strings.HasPrefix(problems[0].(string), "2 input(s) take data from IED GW, which is not in this file") {
		t.Fatalf("problems %v", problems)
	}
}

// Items 6 and 15: preview and the SLD say why an IED can never be loaded.
func TestUnloadableIedsSayWhy(t *testing.T) {
	reasons, err := UnloadableIEDs(stationSCL)
	if err != nil {
		t.Fatal(err)
	}
	if len(reasons) != 1 || !strings.HasPrefix(reasons["BAD"], "importing it fails at BADLD0/LLN0.Beh.stVal: bType is Enum") {
		t.Fatalf("reasons %v", reasons)
	}
	models, err := Preview([]byte(stationSCL))
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range models {
		reason, _ := get(model, "unavailable_reason").(string)
		if (get(model, "name") == "BAD") != strings.HasPrefix(reason, "Cannot be imported at BADLD0/LLN0.Beh.stVal") {
			t.Errorf("%v: %q", get(model, "name"), reason)
		}
	}
}

// Item 16: export refusal words come first, the raw XSD message after "Detail".
func TestExportRefusalIsPlainWords(t *testing.T) {
	err := requireValid(mustValidation("invalid", 12, "Element 'X': not expected."))
	if err == nil || !strings.HasPrefix(err.Error(), "This file cannot be exported: it does not pass the IEC 61850 schema check") ||
		!strings.HasSuffix(err.Error(), "Detail: line 12: Element 'X': not expected.") {
		t.Fatalf("%v", err)
	}
}

// Item 10: the import limit is 100 MB.
func TestImportLimitIs100MB(t *testing.T) {
	padded := strings.Replace(string(synthetic(nil)), "<Header", "<!--"+strings.Repeat(" ", 21<<20)+"--><Header", 1)
	if _, err := ParseDocument([]byte(padded), false); err != nil {
		t.Fatalf("a 21 MB file must parse: %v", err)
	}
	if _, err := ParseDocument(make([]byte, MaxXML+1), false); err == nil || err.Error() != "SCL exceeds the 100 MB import limit" {
		t.Fatalf("over the limit: %v", err)
	}
}
