package scl

import (
	"reflect"
	"strings"
	"testing"

	"studio-engine/internal/scl/xmldoc"
)

// nrLike is an NR/NARI-style IED: S1 (IP, reports), G1 (GOOSE, PIGO) and M1 (SV, PISV). Its MV
// db has no valKind (writable), zeroDb is RO; the second MMXU's DOI marks db RO too.
const nrLike = `<SCL xmlns="http://www.iec.ch/61850/2003/SCL">
<Communication><SubNetwork name="S"><ConnectedAP iedName="R" apName="S1"><Address><P type="IP">10.0.0.5</P></Address></ConnectedAP></SubNetwork></Communication>
<IED name="R">
 <AccessPoint name="S1"><Server><LDevice inst="MEAS"><LN0 lnClass="LLN0" inst="" lnType="L0"><ReportControl name="r"/></LN0>
  <LN lnClass="MMXU" inst="1" lnType="MX"/>
  <LN lnClass="MMXU" inst="2" lnType="MX"><DOI name="TotW"><DAI name="db" valKind="RO"/></DOI></LN></LDevice></Server></AccessPoint>
 <AccessPoint name="G1"><Server><LDevice inst="PIGO"><LN0 lnClass="LLN0" inst="" lnType="L0"><GSEControl name="g"/></LN0></LDevice></Server></AccessPoint>
 <AccessPoint name="M1"><Server><LDevice inst="PISV"><LN0 lnClass="LLN0" inst="" lnType="L0"/></LDevice></Server></AccessPoint>
</IED>
<DataTypeTemplates>
 <LNodeType id="L0" lnClass="LLN0"/>
 <LNodeType id="MX" lnClass="MMXU"><DO name="TotW" type="MV"/></LNodeType>
 <DOType id="MV" cdc="MV"><DA name="mag" bType="Struct" type="AV" fc="MX"/><DA name="db" bType="INT32U" fc="CF"/>
  <DA name="zeroDb" bType="INT32U" fc="CF" valKind="RO"/></DOType>
 <DAType id="AV"><BDA name="f" bType="FLOAT32"/></DAType>
</DataTypeTemplates></SCL>`

func TestStationAccessPointOfNRLayout(t *testing.T) {
	doc, err := xmldoc.Parse([]byte(nrLike), xmldoc.Options{})
	if err != nil {
		t.Fatal(err)
	}
	ied := doc.Root.Find("IED")
	station := stationOf(doc.Root, ied)
	if station.name != "S1" || station.ip != "10.0.0.5" || len(station.others) != 2 {
		t.Fatalf("station %q %q %d", station.name, station.ip, len(station.others))
	}
	note := station.note()
	if !strings.Contains(note, "Simulates S1 (station bus, 10.0.0.5)") || !strings.Contains(note, "G1 (process bus, GOOSE: PIGO)") ||
		!strings.Contains(note, "M1 (process bus, Sampled Values: PISV)") {
		t.Fatal(note)
	}
	if devices := stationDevices(doc.Root, ied); len(devices) != 1 || devices[0].GetOr("inst", "") != "MEAS" {
		t.Fatalf("devices %d", len(devices))
	}
	restore := AsPythonTwin()
	defer restore()
	if twin := stationOf(doc.Root, ied); twin.server != nil || !strings.Contains(twin.refusal, "multiple servers") {
		t.Fatal("the Python twin must keep the old refusal")
	}
}

func TestDeadbandWritesFollowValKind(t *testing.T) {
	results, err := extendedForDocument([]byte(nrLike))
	if err != nil {
		t.Fatal(err)
	}
	if got := results["R"].writable; !reflect.DeepEqual(got, []string{"RMEAS/MMXU1.TotW.db"}) {
		t.Fatalf("writable %v", got)
	}
}
