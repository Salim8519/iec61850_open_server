package mib

import (
	"os"
	"strings"
	"testing"

	"studio-engine/internal/snmp/model"
)

func loadFile(t *testing.T, lib *Library, name string) []string {
	t.Helper()
	text, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	mods, err := lib.Load(name, text, false)
	if err != nil {
		t.Fatalf("Load %s: %v", name, err)
	}
	return mods
}

func moduleInfo(t *testing.T, lib *Library, name string) ModuleInfo {
	t.Helper()
	for _, m := range lib.Modules() {
		if m.Name == name {
			return m
		}
	}
	t.Fatalf("module %s is not loaded", name)
	return ModuleInfo{}
}

func mustNode(t *testing.T, lib *Library, name string) *Node {
	t.Helper()
	oid, ok := lib.Resolve(name)
	if !ok {
		t.Fatalf("%s does not resolve", name)
	}
	n, inst := lib.Lookup(oid)
	if n == nil || inst != "" {
		t.Fatalf("%s: Lookup(%s) = %v %q", name, oid, n, inst)
	}
	return n
}

func TestVendorMIB(t *testing.T) {
	lib := standard(t)
	if mods := loadFile(t, lib, "ACME-SUBSTATION-MIB.txt"); len(mods) != 1 || mods[0] != "ACME-SUBSTATION-MIB" {
		t.Fatalf("modules = %v", mods)
	}
	info := moduleInfo(t, lib, "ACME-SUBSTATION-MIB")
	if info.Builtin || info.File != "ACME-SUBSTATION-MIB.txt" || len(info.Missing) > 0 || len(info.Problems) > 0 {
		t.Errorf("info = %+v", info)
	}
	root := mustNode(t, lib, "acmeSubstationMIB")
	if root.OID != "1.3.6.1.4.1.99999" || root.Description !=
		"Objects of a fictional protection relay. The description spans several lines." {
		t.Errorf("module identity = %+v", root)
	}
	if d := mustNode(t, lib, "acmeConformance").Description; !strings.Contains(d, "-- not a comment") {
		t.Errorf("string with dashes = %q", d)
	}
	if n := mustNode(t, lib, "acmeRelaySerial"); n.Syntax != model.TypeOctets || n.Hex {
		t.Errorf("serial = %+v", n)
	}
	if n := mustNode(t, lib, "acmeRelayTemperature"); n.Units != "degrees Celsius" || n.Kind != "object" {
		t.Errorf("temperature = %+v", n)
	}
	if n := mustNode(t, lib, "acmeBayEntry"); n.Kind != "row" ||
		strings.Join(n.Indexes, ",") != "acmeBayIndex,acmeBayName" {
		t.Errorf("row = %+v", n)
	}
	brk := mustNode(t, lib, "acmeBayBreaker")
	if brk.Kind != "column" || brk.Access != "read-write" || brk.Enums[2] != "closed" || brk.Enums[0] != "intermediate" {
		t.Errorf("breaker = %+v", brk)
	}
	if n := mustNode(t, lib, "acmeBayMac"); !n.Hex || n.Syntax != model.TypeOctets {
		t.Errorf("mac = %+v", n)
	}
	if n := mustNode(t, lib, "acmeBayEnergy"); n.Syntax != model.TypeCounter64 || n.Units != "Wh" {
		t.Errorf("energy = %+v", n)
	}
	if n := mustNode(t, lib, "acmeBayStatus"); n.Enums[1] != "active" || n.Access != "read-create" {
		t.Errorf("row status = %+v", n)
	}
	trip := mustNode(t, lib, "ACME-SUBSTATION-MIB::acmeBreakerTrip")
	if trip.Kind != "notification" || trip.OID != "1.3.6.1.4.1.99999.0.1" || len(trip.Objects) != 2 {
		t.Errorf("notification = %+v", trip)
	}
	if got := lib.Name("1.3.6.1.4.1.99999.1.3.1.3.5"); got != "ACME-SUBSTATION-MIB::acmeBayBreaker.5" {
		t.Errorf("Name = %q", got)
	}
	if got := lib.EnumText(".1.3.6.1.4.1.99999.1.3.1.3.5", 1); got != "open" {
		t.Errorf("EnumText = %q", got)
	}
	if n := mustNode(t, lib, "acmeCompliance"); n.Kind != "node" || n.OID != "1.3.6.1.4.1.99999.2.3" {
		t.Errorf("compliance = %+v", n)
	}
	kids := lib.Children("1.3.6.1.4.1.99999")
	if len(kids) != 3 || kids[0].Name != "acmeNotifs" || kids[2].Name != "acmeConformance" {
		t.Errorf("children = %v", kids)
	}
}

func TestLoadOrderMissingAndRemove(t *testing.T) {
	lib := standard(t)
	mods := loadFile(t, lib, "ACME-EXTRAS.txt")
	if strings.Join(mods, ",") != "ACME-EXT-MIB,ACME-V1-MIB" {
		t.Fatalf("modules = %v", mods)
	}
	ext := moduleInfo(t, lib, "ACME-EXT-MIB")
	if strings.Join(ext.Missing, ",") != "ACME-SUBSTATION-MIB" {
		t.Errorf("missing = %v", ext.Missing)
	}
	if len(ext.Problems) != 2 || !strings.Contains(ext.Problems[0], "acmeExtBroken") ||
		!strings.Contains(ext.Problems[1], "VENDOR-MACRO") {
		t.Errorf("problems = %q", ext.Problems)
	}
	if _, ok := lib.Resolve("acmeExtVoltage"); ok {
		t.Error("acmeExtVoltage resolved without its parent module")
	}
	loadFile(t, lib, "ACME-SUBSTATION-MIB.txt")
	if oid, ok := lib.Resolve("ACME-EXT-MIB::acmeExtVoltage.0"); !ok || oid != "1.3.6.1.4.1.99999.1.9.1.0" {
		t.Errorf("voltage = %q %v", oid, ok)
	}
	if oid, _ := lib.Resolve("acmeExtCurrent"); oid != "1.3.6.1.4.1.99999.1.9.4" {
		t.Errorf("definition after the broken ones = %q", oid)
	}
	if ext = moduleInfo(t, lib, "ACME-EXT-MIB"); len(ext.Missing) != 0 || ext.Nodes != 3 {
		t.Errorf("after load: %+v", ext)
	}
	if !lib.Remove("ACME-SUBSTATION-MIB") || lib.Remove("ACME-SUBSTATION-MIB") {
		t.Error("Remove should succeed once")
	}
	if ext = moduleInfo(t, lib, "ACME-EXT-MIB"); strings.Join(ext.Missing, ",") != "ACME-SUBSTATION-MIB" {
		t.Errorf("after remove: missing %v", ext.Missing)
	}
	if _, ok := lib.Resolve("acmeExtVoltage"); ok {
		t.Error("acmeExtVoltage still resolves after its parent was removed")
	}
}

func TestSMIv1Traps(t *testing.T) {
	lib := standard(t)
	loadFile(t, lib, "ACME-EXTRAS.txt")
	if p := moduleInfo(t, lib, "ACME-V1-MIB").Problems; len(p) > 0 {
		t.Errorf("v1 problems: %q", p)
	}
	trap := mustNode(t, lib, "acmeV1Alarm")
	if trap.Kind != "notification" || trap.OID != "1.3.6.1.4.1.99999.50.0.7" ||
		strings.Join(trap.Objects, ",") != "acmeV1Alarms" {
		t.Errorf("trap = %+v", trap)
	}
	if n := mustNode(t, lib, "acmeV1Alarms"); n.Syntax != model.TypeCounter32 || n.Access != "read-only" {
		t.Errorf("alarms = %+v", n)
	}
	if n := mustNode(t, lib, "acmeV1Peer"); n.Syntax != model.TypeIPAddress {
		t.Errorf("peer = %+v", n)
	}
}

func TestLoadRejectsFilesWithoutModules(t *testing.T) {
	lib := NewLibrary()
	for _, text := range []string{"", "hello world", "\x00\x01\xff\xfe binary", "FOO DEFINITIONS"} {
		_, err := lib.Load("x.mib", []byte(text), false)
		want := `"x.mib" holds no MIB module (expected "NAME DEFINITIONS ::= BEGIN")`
		if err == nil || err.Error() != want {
			t.Errorf("Load(%q) error = %v", text, err)
		}
	}
	if len(lib.Modules()) != 0 {
		t.Error("a failed load added modules")
	}
}

func TestBrokenInputNeverPanics(t *testing.T) {
	text, err := builtinFS.ReadFile("builtin/IF-MIB.txt")
	if err != nil {
		t.Fatal(err)
	}
	for cut := 0; cut < len(text); cut += 997 {
		lib := NewLibrary()
		_, _ = lib.Load("cut.mib", text[:cut], false)
		_ = lib.Modules()
	}
	lib := NewLibrary()
	mods, err := lib.Load("open.mib", []byte("OPEN-MIB DEFINITIONS ::= BEGIN\n x OBJECT-TYPE \"never closed"), false)
	if err != nil || len(mods) != 1 {
		t.Fatalf("unterminated module: %v %v", mods, err)
	}
	if p := moduleInfo(t, lib, "OPEN-MIB").Problems; len(p) == 0 {
		t.Error("unterminated module should report a problem")
	}
}
