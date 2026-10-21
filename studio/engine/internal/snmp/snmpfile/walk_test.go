package snmpfile

import (
	"os"
	"strings"
	"testing"

	"studio-engine/internal/snmp/mib"
	"studio-engine/internal/snmp/model"
)

func standardLib(t testing.TB) *mib.Library {
	t.Helper()
	lib, err := mib.Standard()
	if err != nil {
		t.Fatal(err)
	}
	return lib
}

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func byOID(points []model.Point) map[string]model.Point {
	m := map[string]model.Point{}
	for _, p := range points {
		m[p.OID] = p
	}
	return m
}

func expectPoint(t *testing.T, got map[string]model.Point, oid, typ, value string, hex bool) {
	t.Helper()
	p, ok := got[oid]
	if !ok {
		t.Errorf("%s: missing", oid)
		return
	}
	if p.Type != typ || p.Value != value || p.Hex != hex {
		t.Errorf("%s = %s %q hex=%v; want %s %q hex=%v", oid, p.Type, p.Value, p.Hex, typ, value, hex)
	}
	want := model.BehaviourFixed
	if oid == sysUpTimeOID {
		want = model.BehaviourUptime
	}
	if p.Behaviour != want || p.Writable || p.Rate != 0 {
		t.Errorf("%s behaviour %q writable %v rate %v", oid, p.Behaviour, p.Writable, p.Rate)
	}
}

func TestParseWalkDefaultStyle(t *testing.T) {
	points, problems, err := ParseWalk(readTestdata(t, "walk-default.txt"), standardLib(t))
	if err != nil {
		t.Fatal(err)
	}
	got := byOID(points)
	const ifEntry, ent = "1.3.6.1.2.1.2.2.1.", "1.3.6.1.4.1.99999.1.2."
	expectPoint(t, got, "1.3.6.1.2.1.1.1.0", model.TypeOctets, "Acme relay R-100 firmware 2.1", false)
	expectPoint(t, got, "1.3.6.1.2.1.1.2.0", model.TypeOID, "1.3.6.1.4.1.99999.1.1", false)
	expectPoint(t, got, sysUpTimeOID, model.TypeTimeTicks, "123456", false)
	expectPoint(t, got, "1.3.6.1.2.1.1.4.0", model.TypeOctets, "", false)
	expectPoint(t, got, "1.3.6.1.2.1.1.5.0", model.TypeOctets, "relay-01", false)
	expectPoint(t, got, "1.3.6.1.2.1.1.6.0", model.TypeOctets, "", false)
	expectPoint(t, got, "1.3.6.1.2.1.1.9.1.3.1", model.TypeOctets,
		"The MIB module for SNMPv2 entities,\nspanning a second line with a \"quote\"", false)
	expectPoint(t, got, "1.3.6.1.2.1.1.9.1.3.2", model.TypeOctets,
		"First line of an unquoted text\nsecond line of the same text", false)
	expectPoint(t, got, ifEntry+"3.1", model.TypeInteger, "24", false)
	expectPoint(t, got, ifEntry+"3.2", model.TypeInteger, "6", false)
	expectPoint(t, got, ifEntry+"5.2", model.TypeGauge32, "100000000", false)
	expectPoint(t, got, ifEntry+"6.1", model.TypeOctets, "", true)
	expectPoint(t, got, ifEntry+"6.2", model.TypeOctets, "00 15 5d 0a 0b 0c", true)
	expectPoint(t, got, ifEntry+"7.2", model.TypeInteger, "1", false)
	expectPoint(t, got, ifEntry+"8.2", model.TypeInteger, "7", false)
	expectPoint(t, got, ifEntry+"10.2", model.TypeCounter32, "4294967295", false)
	expectPoint(t, got, ifEntry+"22.2", model.TypeOID, "0.0", false)
	expectPoint(t, got, "1.3.6.1.2.1.31.1.1.1.6.2", model.TypeCounter64, "18446744073709551615", false)
	expectPoint(t, got, "1.3.6.1.2.1.31.1.1.1.18.2", model.TypeOctets,
		"41 63 6d 65 20 62 61 79 20 6f 6e 65 20 66 65 65 64 65 72 20 70 6f 72 74", true)
	expectPoint(t, got, "1.3.6.1.2.1.4.20.1.1.10.0.0.5", model.TypeIPAddress, "10.0.0.5", false)
	expectPoint(t, got, "1.3.6.1.2.1.25.1.2.0", model.TypeOctets, "07 ea 0a 14 0e 15 05 03 2b 04 00", true)
	expectPoint(t, got, "1.3.6.1.2.1.25.5.1.1.2.1", model.TypeInteger, "2048", false)
	expectPoint(t, got, "1.3.6.1.2.1.11.1.0", model.TypeCounter32, "13", false)
	expectPoint(t, got, "1.3.6.1.2.1.11.30.0", model.TypeInteger, "2", false)
	expectPoint(t, got, "1.3.6.1.2.1.33.1.2.1.0", model.TypeInteger, "2", false)
	expectPoint(t, got, "1.3.6.1.2.1.33.1.2.7.0", model.TypeInteger, "25", false)
	expectPoint(t, got, "1.3.6.1.2.1.17.2.15.1.3.1", model.TypeGauge32, "5", false)
	expectPoint(t, got, "1.3.6.1.2.1.197.1.2.1.0", model.TypeInteger, "6", false)
	expectPoint(t, got, "1.3.6.1.2.1.197.1.2.2.0", model.TypeGauge32, "2", false)
	expectPoint(t, got, ent+"1.0", model.TypeOpaque, "9f 78 04 3f c0 00 00", false)
	expectPoint(t, got, ent+"2.0", model.TypeOpaque, "9f 78 04 3f c0 00 00", false)
	expectPoint(t, got, ent+"3.0", model.TypeOctets, "80 00", true)
	expectPoint(t, got, ent+"4.0", model.TypeIPAddress, "10.0.0.1", false)
	expectPoint(t, got, ent+"5.0", model.TypeNull, "", false)
	expectPoint(t, got, ent+"6.0", model.TypeOpaque, "9f 7b 02 01 2c", false)
	expectPoint(t, got, ent+"7.0", model.TypeOpaque, "9f 7a 01 fe", false)
	expectPoint(t, got, ifEntry+"9.2", model.TypeTimeTicks, "0", false)
	if len(points) != 42 {
		t.Errorf("got %d points, want 42", len(points))
	}
	for i := 1; i < len(points); i++ {
		if mib.CompareOID(points[i-1].OID, points[i].OID) >= 0 {
			t.Errorf("not sorted: %s before %s", points[i-1].OID, points[i].OID)
		}
	}
	wantProblems := []struct {
		line int
		text string
	}{
		{32, "appears more than once"},
		{48, "FOO-MIB::fooBar.1: unknown name — import the MIB that defines it, or export the walk with -On"},
		{49, "\"sideways\" is not a number"},
		{50, "is not a number from -2147483648"},
		{51, "\"-1\" is not a whole number from 0 to 4294967295"},
	}
	if len(problems) != len(wantProblems) {
		t.Fatalf("problems = %+v", problems)
	}
	for i, w := range wantProblems {
		if problems[i].Line != w.line || !strings.Contains(problems[i].Text, w.text) {
			t.Errorf("problem %d = %+v; want line %d with %q", i, problems[i], w.line, w.text)
		}
	}
}

func TestParseWalkNumericStyle(t *testing.T) {
	points, problems, err := ParseWalk(readTestdata(t, "walk-numeric.txt"), nil)
	if err != nil || len(problems) != 0 {
		t.Fatalf("err %v problems %+v", err, problems)
	}
	var oids []string
	for _, p := range points {
		oids = append(oids, p.OID)
	}
	want := "1.3.6.1.2.1.1.1.0 1.3.6.1.2.1.1.2.0 1.3.6.1.2.1.1.3.0 1.3.6.1.2.1.2.2.1.6.2 " +
		"1.3.6.1.2.1.2.2.1.8.2 1.3.6.1.2.1.2.2.1.10.2 1.3.6.1.2.1.2.2.1.10.10 " +
		"1.3.6.1.4.1.99999.1.3.1.2.1 1.3.6.1.4.1.99999.1.3.1.3.1 1.3.6.1.4.1.99999.1.3.1.4.1 " +
		"1.3.6.1.4.1.99999.1.9 1.3.6.1.4.1.99999.1.10"
	if strings.Join(oids, " ") != want {
		t.Errorf("order:\n%s\nwant\n%s", strings.Join(oids, " "), want)
	}
	got := byOID(points)
	expectPoint(t, got, "1.3.6.1.2.1.1.1.0", model.TypeOctets, "Acme relay R-100", false)
	expectPoint(t, got, "1.3.6.1.2.1.1.2.0", model.TypeOID, "1.3.6.1.4.1.99999.1.1", false)
	expectPoint(t, got, sysUpTimeOID, model.TypeTimeTicks, "5000", false)
	expectPoint(t, got, "1.3.6.1.2.1.2.2.1.6.2", model.TypeOctets, "00 15 5d 0a 0b 0c", true)
	expectPoint(t, got, "1.3.6.1.4.1.99999.1.3.1.4.1", model.TypeCounter64, "123456789012", false)
	expectPoint(t, got, "1.3.6.1.4.1.99999.1.3.1.2.1", model.TypeOctets, "", false)
}

func TestParseWalkIsoStyle(t *testing.T) {
	text := readTestdata(t, "walk-iso.txt")
	points, problems, err := ParseWalk(text, standardLib(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 9 || len(problems) != 1 || problems[0].Line != 1 {
		t.Fatalf("points %d problems %+v", len(points), problems)
	}
	got := byOID(points)
	expectPoint(t, got, "1.3.6.1.2.1.1.1.0", model.TypeOctets, "Acme relay R-100\nsecond line", false)
	expectPoint(t, got, "1.3.6.1.2.1.1.2.0", model.TypeOID, "1.3.6.1.4.1.99999.1.1", false)
	expectPoint(t, got, sysUpTimeOID, model.TypeTimeTicks, "42", false)
	expectPoint(t, got, "1.3.6.1.2.1.2.2.1.8.3", model.TypeInteger, "1", false)
	expectPoint(t, got, "1.3.6.1.2.1.4.20.1.1.10.0.0.5", model.TypeIPAddress, "10.0.0.5", false)
	_, problems, err = ParseWalk(text, nil)
	if err != nil || len(problems) != 2 || !strings.Contains(problems[1].Text, "\"up\"") {
		t.Errorf("without MIBs: %v %+v", err, problems)
	}
}

func TestParseWalkErrors(t *testing.T) {
	for _, text := range []string{"", "\n\n", "hello world\nthis is not a walk", "foo.1 = INTEGER: 1"} {
		points, _, err := ParseWalk(text, standardLib(t))
		if err == nil || len(points) != 0 || !strings.Contains(err.Error(), "no SNMP values found") {
			t.Errorf("ParseWalk(%q) = %d points, %v", text, len(points), err)
		}
	}
	_, problems, _ := ParseWalk(".1.3.6.1.2.1.1.1.0 = STRING: \"never closed\n", nil)
	if len(problems) != 1 || !strings.Contains(problems[0].Text, "no closing quote") {
		t.Errorf("unclosed quote: %+v", problems)
	}
	points, problems, err := ParseWalk("sysName.0 = relay-02\r\nsysUpTime.0 = 77\r\nifPhysAddress.1 = 0:1:2:3:4:5\r\n",
		standardLib(t))
	if err != nil || len(problems) != 0 || len(points) != 3 {
		t.Fatalf("-OQ style: %v %+v %+v", err, problems, points)
	}
	got := byOID(points)
	expectPoint(t, got, "1.3.6.1.2.1.1.5.0", model.TypeOctets, "relay-02", false)
	expectPoint(t, got, sysUpTimeOID, model.TypeTimeTicks, "77", false)
	expectPoint(t, got, "1.3.6.1.2.1.2.2.1.6.1", model.TypeOctets, "00 01 02 03 04 05", true)
}
