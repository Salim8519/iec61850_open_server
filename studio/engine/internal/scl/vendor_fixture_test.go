package scl

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// vendorSCL is a minimal synthetic SCL reproducing one vendor-file shape at a time
// (change 0185). Placeholders: {ld} LDevice inst, {ln0} LLN0 extra content, {gse} extra
// GSE in the ConnectedAP, {lpld} type and Val of LPHD.PhyNam-like DA "d", {sps} extra DAs of
// the SPS type, {dpc} DPC stVal content, {spc} the SPCSO1 type, {ggio} GGIO1 extra content.
const vendorSCL = `<?xml version="1.0" encoding="UTF-8"?>
<SCL xmlns="http://www.iec.ch/61850/2003/SCL" version="2007" revision="B" release="4">
  <Header id="synthetic"/>
  <Communication><SubNetwork name="S1"><ConnectedAP iedName="R1" apName="AP1"><Address>
    <P type="IP">10.0.0.5</P><P type="IP-SUBNET">255.255.255.0</P></Address>{gse}</ConnectedAP></SubNetwork></Communication>
  <IED name="R1" manufacturer="Synthetic" type="T"><AccessPoint name="AP1"><Server><Authentication/>
    <LDevice inst="{ld}">
      <LN0 lnClass="LLN0" inst="" lnType="LLN0_T">{ln0}</LN0>
      <LN lnClass="GGIO" inst="1" lnType="GGIO_T">{ggio}</LN>
      <LN lnClass="XCBR" inst="1" lnType="XCBR_T"/>
    </LDevice></Server></AccessPoint></IED>
  <DataTypeTemplates>
    <LNodeType id="LLN0_T" lnClass="LLN0"><DO name="Beh" type="ENS_Beh"/><DO name="NamPlt" type="LPL_T"/></LNodeType>
    <LNodeType id="GGIO_T" lnClass="GGIO"><DO name="Beh" type="ENS_Beh"/><DO name="Ind1" type="SPS_T"/><DO name="AnIn1" type="MV_T"/><DO name="SPCSO1" type="{spc}"/></LNodeType>
    <LNodeType id="XCBR_T" lnClass="XCBR"><DO name="Beh" type="ENS_Beh"/><DO name="Pos" type="DPC_T"/></LNodeType>
    <DOType id="ENS_Beh" cdc="ENS"><DA name="stVal" bType="Enum" fc="ST" type="BehKind"/><DA name="q" bType="Quality" fc="ST"/><DA name="t" bType="Timestamp" fc="ST"/></DOType>
    <DOType id="LPL_T" cdc="LPL"><DA name="vendor" bType="VisString255" fc="DC"/><DA name="d" {lpld}</DA></DOType>
    <DOType id="SPS_T" cdc="SPS"><DA name="stVal" bType="BOOLEAN" fc="ST"/><DA name="q" bType="Quality" fc="ST"/><DA name="t" bType="Timestamp" fc="ST"/>{sps}</DOType>
    <DOType id="MV_T" cdc="MV"><DA name="mag" bType="Struct" fc="MX" type="AV"/><DA name="q" bType="Quality" fc="MX"/><DA name="t" bType="Timestamp" fc="MX"/></DOType>
    <DOType id="SPC_T" cdc="SPC"><DA name="stVal" bType="BOOLEAN" fc="ST" dchg="true"/><DA name="q" bType="Quality" fc="ST" qchg="true"/><DA name="Oper" bType="Struct" fc="CO" type="SPCOper"/><DA name="ctlModel" bType="Enum" fc="CF" type="CtlModelKind"><Val>direct-with-normal-security</Val></DA></DOType>
    <DOType id="SPC_NOSTVAL" cdc="SPC"><DA name="Oper" bType="Struct" fc="CO" type="SPCOper"/><DA name="ctlModel" bType="Enum" fc="CF" type="CtlModelKind"><Val>direct-with-normal-security</Val></DA></DOType>
    <DOType id="BSC_BOOL" cdc="BSC"><DA name="valWTr" bType="Struct" fc="ST" type="VWT"/><DA name="Oper" bType="Struct" fc="CO" type="SPCOper"/><DA name="ctlModel" bType="Enum" fc="CF" type="CtlModelKind"><Val>direct-with-normal-security</Val></DA></DOType>
    <DOType id="DPC_T" cdc="DPC"><DA name="stVal" bType="Dbpos" fc="ST">{dpc}</DA><DA name="q" bType="Quality" fc="ST"/><DA name="ctlModel" bType="Enum" fc="CF" type="CtlModelKind"><Val>status-only</Val></DA></DOType>
    <DAType id="AV"><BDA name="f" bType="FLOAT32"/></DAType>
    <DAType id="VWT"><BDA name="posVal" bType="INT8"/><BDA name="transInd" bType="BOOLEAN"/></DAType>
    <DAType id="SPCOper"><BDA name="ctlVal" bType="BOOLEAN"/><BDA name="ctlNum" bType="INT8U"/><BDA name="T" bType="Timestamp"/><BDA name="Test" bType="BOOLEAN"/><BDA name="Check" bType="Check"/></DAType>
    <EnumType id="BehKind"><EnumVal ord="1">on</EnumVal><EnumVal ord="2">blocked</EnumVal><EnumVal ord="3">test</EnumVal><EnumVal ord="4">test/blocked</EnumVal><EnumVal ord="5">off</EnumVal></EnumType>
    <EnumType id="CtlModelKind"><EnumVal ord="0">status-only</EnumVal><EnumVal ord="1">direct-with-normal-security</EnumVal><EnumVal ord="2">sbo-with-normal-security</EnumVal><EnumVal ord="3">direct-with-enhanced-security</EnumVal><EnumVal ord="4">sbo-with-enhanced-security</EnumVal></EnumType>
  </DataTypeTemplates>
</SCL>
`

// synthetic fills vendorSCL; unset placeholders take the plain, importable defaults.
func synthetic(overrides map[string]string) []byte {
	values := map[string]string{"{ld}": "LD0", "{ln0}": "", "{gse}": "", "{lpld}": `bType="VisString255" fc="DC">`,
		"{sps}": "", "{dpc}": "", "{spc}": "SPC_T", "{ggio}": ""}
	for key, value := range overrides {
		values[key] = value
	}
	text := vendorSCL
	for key, value := range values {
		text = strings.ReplaceAll(text, key, value)
	}
	return []byte(text)
}

// importOne imports the synthetic IED with an always-valid schema check.
func importOne(t *testing.T, source []byte) (*pyjson.Object, error) {
	t.Helper()
	models, err := (&Importer{Validator: validSchema{}, NewID: func() string { return "id" }}).Import(source, "r.cid", SelectAll)
	if err != nil {
		return nil, err
	}
	return models[0], nil
}

func mustImport(t *testing.T, source []byte) *pyjson.Object {
	t.Helper()
	model, err := importOne(t, source)
	if err != nil {
		t.Fatalf("import refused: %v", err)
	}
	return model
}

// signalOf is the signal with this reference, or nil.
func signalOf(model *pyjson.Object, ref string) *pyjson.Object {
	for _, item := range get(model, "signals").([]any) {
		if signal := item.(*pyjson.Object); get(signal, "ref") == ref {
			return signal
		}
	}
	return nil
}

// warningWith is the first warning containing text, or "".
func warningWith(model *pyjson.Object, text string) string {
	for _, item := range get(model, "warnings").([]any) {
		if strings.Contains(item.(string), text) {
			return item.(string)
		}
	}
	return ""
}

// cfgLine is the first cfg line starting with prefix inside the LN, or "".
func cfgLine(model *pyjson.Object, prefix string) string {
	for _, line := range strings.Split(get(model, "cfg").(string), "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

func TestSyntheticFixtureImports(t *testing.T) {
	model := mustImport(t, synthetic(nil))
	if signalOf(model, "R1LD0/GGIO1.Ind1.stVal") == nil {
		t.Fatal("baseline fixture lost its signals")
	}
}
