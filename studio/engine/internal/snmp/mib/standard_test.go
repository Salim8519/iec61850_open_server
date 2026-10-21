package mib

import (
	"testing"
	"time"

	"studio-engine/internal/snmp/model"
)

func standard(t testing.TB) *Library {
	t.Helper()
	lib, err := Standard()
	if err != nil {
		t.Fatalf("Standard: %v", err)
	}
	return lib
}

func TestStandardLoadsEveryBuiltinCleanly(t *testing.T) {
	start := time.Now()
	lib := standard(t)
	if d := time.Since(start); d > time.Second {
		t.Errorf("Standard took %v, want well under 1 s", d)
	}
	mods := lib.Modules()
	if len(mods) != len(BuiltinNames()) {
		t.Errorf("loaded %d modules, %d built-in files", len(mods), len(BuiltinNames()))
	}
	noOIDs := map[string]bool{"RFC-1212": true, "RFC-1215": true, "SNMPv2-TC": true, "SNMPv2-CONF": true}
	for _, m := range mods {
		if !m.Builtin || (m.Nodes == 0) != noOIDs[m.Name] {
			t.Errorf("%s: builtin=%v nodes=%d", m.Name, m.Builtin, m.Nodes)
		}
		if len(m.Missing) > 0 {
			t.Errorf("%s: missing imports %v", m.Name, m.Missing)
		}
		for _, p := range m.Problems {
			t.Errorf("%s: %s", m.Name, p)
		}
	}
	for _, want := range []string{"SNMPv2-SMI", "IF-MIB", "IP-MIB", "UPS-MIB", "NTPv4-MIB",
		"ENTITY-SENSOR-MIB", "HOST-RESOURCES-MIB", "BRIDGE-MIB", "SNMP-USER-BASED-SM-MIB"} {
		found := false
		for _, m := range mods {
			found = found || m.Name == want
		}
		if !found {
			t.Errorf("built-in module %s is missing", want)
		}
	}
}

func TestResolveAndNameRoundTrip(t *testing.T) {
	lib := standard(t)
	cases := []struct{ in, oid, name string }{
		{"sysDescr.0", "1.3.6.1.2.1.1.1.0", "SNMPv2-MIB::sysDescr.0"},
		{"SNMPv2-MIB::sysUpTime.0", "1.3.6.1.2.1.1.3.0", "SNMPv2-MIB::sysUpTime.0"},
		{"IF-MIB::ifOperStatus.3", "1.3.6.1.2.1.2.2.1.8.3", "IF-MIB::ifOperStatus.3"},
		{"ifHCInOctets.1", "1.3.6.1.2.1.31.1.1.1.6.1", "IF-MIB::ifHCInOctets.1"},
		{".1.3.6.1.2.1.1.5.0", "1.3.6.1.2.1.1.5.0", "SNMPv2-MIB::sysName.0"},
		{"iso.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.1.0", "SNMPv2-MIB::sysDescr.0"},
		{"SNMPv2-SMI::enterprises.9.1.1", "1.3.6.1.4.1.9.1.1", "SNMPv2-SMI::enterprises.9.1.1"},
		{"mib-2.33.1.2.1.0", "1.3.6.1.2.1.33.1.2.1.0", "UPS-MIB::upsBatteryStatus.0"},
		{"RFC1213-MIB::ifDescr.2", "1.3.6.1.2.1.2.2.1.2.2", "IF-MIB::ifDescr.2"},
		{".iso.org.dod.internet.mgmt.mib-2.system.sysName.0", "1.3.6.1.2.1.1.5.0", "SNMPv2-MIB::sysName.0"},
		{"ntpEntStatusCurrentMode.0", "1.3.6.1.2.1.197.1.2.1.0", "NTPv4-MIB::ntpEntStatusCurrentMode.0"},
		{"entPhySensorValue.7", "1.3.6.1.2.1.99.1.1.1.4.7", "ENTITY-SENSOR-MIB::entPhySensorValue.7"},
		{`snmpTargetAddrTDomain."trap1"`, "1.3.6.1.6.3.12.1.2.1.2.5.116.114.97.112.49",
			"SNMP-TARGET-MIB::snmpTargetAddrTDomain.5.116.114.97.112.49"},
	}
	for _, c := range cases {
		oid, ok := lib.Resolve(c.in)
		if !ok || oid != c.oid {
			t.Errorf("Resolve(%q) = %q, %v; want %q", c.in, oid, ok, c.oid)
			continue
		}
		if got := lib.Name(oid); got != c.name {
			t.Errorf("Name(%q) = %q; want %q", oid, got, c.name)
		}
		if back, ok := lib.Resolve(c.name); !ok || back != c.oid {
			t.Errorf("Resolve(Name) = %q, %v; want %q", back, ok, c.oid)
		}
	}
	for _, bad := range []string{"", "noSuchThing.0", "IF-MIB::sysDescr.0", "ifDescr.x", "IF-MIB::"} {
		if oid, ok := lib.Resolve(bad); ok {
			t.Errorf("Resolve(%q) = %q, want failure", bad, oid)
		}
	}
	if got := lib.Name("5.1.2"); got != "5.1.2" {
		t.Errorf("Name of unknown root = %q", got)
	}
}

func TestNodeDetails(t *testing.T) {
	lib := standard(t)
	node := func(name string) *Node {
		oid, ok := lib.Resolve(name)
		n, inst := lib.Lookup(oid)
		if !ok || n == nil || inst != "" {
			t.Fatalf("%s: not found (%q %v)", name, oid, ok)
		}
		return n
	}
	checks := []struct {
		name, kind, syntax string
		hex                bool
	}{
		{"sysDescr", "object", model.TypeOctets, false},
		{"sysUpTime", "object", model.TypeTimeTicks, false},
		{"sysObjectID", "object", model.TypeOID, false},
		{"ifHCInOctets", "column", model.TypeCounter64, false},
		{"ifPhysAddress", "column", model.TypeOctets, true},
		{"ifInOctets", "column", model.TypeCounter32, false},
		{"ifSpeed", "column", model.TypeGauge32, false},
		{"ipAdEntAddr", "column", model.TypeIPAddress, false},
		{"ifTable", "table", "", false},
		{"ifEntry", "row", "", false},
		{"ifXEntry", "row", "", false},
		{"linkDown", "notification", "", false},
		{"upsBatteryStatus", "object", model.TypeInteger, false},
		{"entPhysicalSerialNum", "column", model.TypeOctets, false},
		{"usmUserEngineID", "column", model.TypeOctets, true},
		{"ifMIB", "node", "", false},
	}
	for _, c := range checks {
		n := node(c.name)
		if n.Kind != c.kind || n.Syntax != c.syntax || n.Hex != c.hex {
			t.Errorf("%s: kind %q syntax %q hex %v; want %q %q %v", c.name, n.Kind, n.Syntax, n.Hex,
				c.kind, c.syntax, c.hex)
		}
	}
	oper := node("ifOperStatus")
	if oper.Enums[1] != "up" || oper.Enums[2] != "down" || oper.Access != "read-only" || oper.Module != "IF-MIB" {
		t.Errorf("ifOperStatus = %+v", oper)
	}
	if ups := node("upsBatteryStatus"); ups.Enums[2] != "batteryNormal" || ups.Enums[4] != "batteryDepleted" {
		t.Errorf("upsBatteryStatus enums = %v", ups.Enums)
	}
	if tv := node("ifPromiscuousMode"); tv.Enums[1] != "true" || tv.Syntax != model.TypeInteger {
		t.Errorf("TruthValue enums = %v", tv.Enums)
	}
	if x := node("ifXEntry"); len(x.Indexes) != 1 || x.Indexes[0] != "ifIndex" {
		t.Errorf("ifXEntry AUGMENTS indexes = %v", x.Indexes)
	}
	if ld := node("linkDown"); len(ld.Objects) != 3 || ld.OID != "1.3.6.1.6.3.1.1.5.3" {
		t.Errorf("linkDown = %+v", ld)
	}
	if d := node("sysDescr").Description; d == "" || len([]rune(d)) > 300 {
		t.Errorf("sysDescr description %q", d)
	}
	if lib.EnumText("1.3.6.1.2.1.2.2.1.8.4", 2) != "down" || lib.EnumText("1.3.6.1.2.1.1.1.0", 1) != "" {
		t.Error("EnumText")
	}
	if units := node("upsBatteryTemperature").Units; units != "degrees Centigrade" {
		t.Errorf("upsBatteryTemperature units %q", units)
	}
}

func TestChildrenAndLookup(t *testing.T) {
	lib := standard(t)
	kids := lib.Children("1.3.6.1.2.1.1")
	if len(kids) < 9 || kids[0].Name != "sysDescr" || kids[1].Name != "sysObjectID" {
		t.Fatalf("system children = %d, first %v", len(kids), kids)
	}
	for i := 1; i < len(kids); i++ {
		if CompareOID(kids[i-1].OID, kids[i].OID) >= 0 {
			t.Errorf("children not sorted: %s, %s", kids[i-1].OID, kids[i].OID)
		}
	}
	if n, inst := lib.Lookup("1.3.6.1.4.1.99999.7"); n == nil || n.Name != "enterprises" || inst != "99999.7" {
		t.Errorf("Lookup enterprise = %v %q", n, inst)
	}
	if n, _ := lib.Lookup("9.9"); n != nil {
		t.Errorf("Lookup(9.9) = %v, want nil", n)
	}
	if len(lib.Roots()) != 3 {
		t.Errorf("roots = %v", lib.Roots())
	}
}

func TestEmptyLibraryKnowsRoots(t *testing.T) {
	lib := NewLibrary()
	if oid, ok := lib.Resolve("enterprises.99999.1"); !ok || oid != "1.3.6.1.4.1.99999.1" {
		t.Errorf("Resolve enterprises = %q %v", oid, ok)
	}
	if got := lib.Name("1.3.6.1.2.1.1.1.0"); got != "mib-2.1.1.0" {
		t.Errorf("Name = %q", got)
	}
	if oid, ok := lib.Resolve("zeroDotZero"); !ok || oid != "0.0" {
		t.Errorf("zeroDotZero = %q", oid)
	}
}

func BenchmarkParseBuiltins(b *testing.B) {
	for i := 0; i < b.N; i++ {
		mods := map[string]*module{}
		for _, name := range BuiltinNames() {
			text, _ := builtinFS.ReadFile("builtin/" + name + ".txt")
			for _, m := range parseModules(name, string(text), true) {
				mods[m.name] = m
			}
		}
		build(mods)
	}
}
