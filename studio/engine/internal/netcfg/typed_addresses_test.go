package netcfg

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// Typed addresses get the planner's checks (Go-only fix; Python marked every valid typed
// address "ok"). Everything runs on fakes: no Windows setting is read or changed.

const labGUID = "{AB12}"

var (
	labCard  = Adapter{GUID: labGUID, Name: "Lab NIC", Status: "Up", Addresses: []Address{{IP: "192.168.10.5", Prefix: 24}}}
	wifiCard = Adapter{GUID: "{CD34}", Name: "Wi-Fi", Status: "Up", Kind: "wifi",
		Addresses: []Address{{IP: "172.16.1.20", Prefix: 24}}}
)

type typedFixture struct {
	service *NetworkService
	probes  [][]string
	listed  int
}

func newTypedFixture(ieds []PlanIED, cards ...Adapter) *typedFixture {
	fixture := &typedFixture{}
	fixture.service = &NetworkService{
		Projects: &fakeProjects{ieds: ieds, mode: "loopback"}, Program: `C:\app\native\studio_ied.exe`,
		Source: AdapterFunc(func() ([]Adapter, error) { fixture.listed++; return cards, nil }),
		Rules:  func() ([]string, error) { return nil, nil }, Clock: time.Now,
		Probe: ProberFunc(func(addresses []string) (map[string]bool, error) {
			fixture.probes = append(fixture.probes, append([]string(nil), addresses...))
			return map[string]bool{}, nil
		})}
	return fixture
}

func scdIEDs(addresses ...string) []PlanIED {
	names := []string{"FEED1", "FEED2", "FEED3"}
	ieds := make([]PlanIED, len(addresses))
	for index, address := range addresses {
		ieds[index] = PlanIED{ID: "ied" + string(rune('1'+index)), Name: names[index], SCLIP: address, Port: 102}
	}
	return ieds
}

func typeInto(manual *pyjson.Object, id string, ip, prefix any) *pyjson.Object {
	return manual.Set(id, pyjson.NewObject().Set("ip", ip).Set("prefix", prefix))
}

func (f *typedFixture) preview(t *testing.T, strategy string, manual any) (map[string]*pyjson.Object, bool) {
	t.Helper()
	result, state, err := f.service.preview(labGUID, strategy, manual)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]*pyjson.Object{}
	for _, row := range RowsObject(state.rows) {
		object := row.(*pyjson.Object)
		rows[pyStr(get(object, "ied_id"))] = object
	}
	return rows, truthy(get(result, "ready"))
}

func wantStatus(t *testing.T, rows map[string]*pyjson.Object, id, status, hint string) {
	t.Helper()
	row := rows[id]
	if get(row, "status") != status || !strings.Contains(pyStr(get(row, "hint")), hint) {
		t.Errorf("%s: status %v hint %q, want %s containing %q", id, get(row, "status"), get(row, "hint"), status, hint)
	}
}

func TestTypedAddressOfAnotherIEDInTheSCDIsADuplicate(t *testing.T) {
	fixture := newTypedFixture(scdIEDs("192.168.10.21", "192.168.10.22"), labCard)
	manual := typeInto(pyjson.NewObject(), "ied2", "192.168.10.21", int64(24))
	rows, ready := fixture.preview(t, StrategySCD, manual)
	wantStatus(t, rows, "ied1", "ok", "")
	wantStatus(t, rows, "ied2", "duplicate", "FEED1 already uses 192.168.10.21. Each IED needs its own address.")
	if ready {
		t.Error("the apply button must stay off")
	}
	if _, err := fixture.service.Apply(fingerprintOf(t, fixture, StrategySCD, manual), true); err == nil ||
		!strings.Contains(err.Error(), "Fix the highlighted address problems") {
		t.Errorf("apply must refuse a duplicate: %v", err)
	}
}

func fingerprintOf(t *testing.T, fixture *typedFixture, strategy string, manual *pyjson.Object) string {
	t.Helper()
	result, _, err := fixture.service.preview(labGUID, strategy, manual)
	if err != nil {
		t.Fatal(err)
	}
	return pyStr(get(result, "fingerprint"))
}

func TestTwoTypedRowsWithTheSameAddressAreADuplicate(t *testing.T) {
	fixture := newTypedFixture(scdIEDs("192.168.10.21", "192.168.10.22", ""), labCard)
	manual := typeInto(typeInto(pyjson.NewObject(), "ied2", "192.168.10.50", int64(24)), "ied3", "192.168.10.50", "24")
	rows, ready := fixture.preview(t, StrategySCD, manual)
	wantStatus(t, rows, "ied2", "ok", "Address typed by you.")
	wantStatus(t, rows, "ied3", "duplicate", "FEED2 already uses 192.168.10.50")
	if ready {
		t.Error("the apply button must stay off")
	}
}

func TestPlannedRowsHoldTheirAddressBeforeTypedRows(t *testing.T) {
	fixture := newTypedFixture(scdIEDs("192.168.10.21", "192.168.10.22"), labCard)
	manual := typeInto(pyjson.NewObject(), "ied1", "192.168.10.22", int64(24)) // FEED2's file address
	rows, ready := fixture.preview(t, StrategySCD, manual)
	wantStatus(t, rows, "ied1", "duplicate", "FEED2 already uses 192.168.10.22")
	wantStatus(t, rows, "ied2", "ok", "")
	if ready {
		t.Error("the apply button must stay off")
	}
}

func TestTypedAddressOfThePCOnAnotherAdapterIsRefused(t *testing.T) {
	fixture := newTypedFixture(scdIEDs("192.168.10.21"), labCard, wifiCard)
	manual := typeInto(pyjson.NewObject(), "ied1", "172.16.1.20", int64(24))
	rows, ready := fixture.preview(t, StrategySCD, manual)
	wantStatus(t, rows, "ied1", "used_by_other_adapter", "172.16.1.20 already belongs to 'Wi-Fi'")
	if ready {
		t.Error("the apply button must stay off")
	}
}

func TestTypedAddressAlreadyOnTheChosenAdapterIsUsableButOnlyOnce(t *testing.T) {
	fixture := newTypedFixture(scdIEDs("192.168.10.21", "192.168.10.22"), labCard)
	own := typeInto(pyjson.NewObject(), "ied1", "192.168.10.5", int64(24))
	rows, ready := fixture.preview(t, StrategySCD, own)
	wantStatus(t, rows, "ied1", "already_on_adapter", "already on the adapter")
	if !ready {
		t.Error("the PC's address on the chosen card stays usable, as for the SCD")
	}
	both := typeInto(own, "ied2", "192.168.10.5", int64(24))
	rows, ready = fixture.preview(t, StrategySCD, both)
	wantStatus(t, rows, "ied2", "duplicate", "FEED1 already uses 192.168.10.5")
	if ready {
		t.Error("two IEDs on the PC's address must not be ready")
	}
}

func TestTypedFreeAddressStaysOkAndAsksNothingExtra(t *testing.T) {
	fixture := newTypedFixture(scdIEDs("192.168.10.21", "192.168.10.22"), labCard, wifiCard)
	manual := typeInto(pyjson.NewObject(), "ied2", "192.168.10.77", int64(24))
	rows, ready := fixture.preview(t, StrategySCD, manual)
	wantStatus(t, rows, "ied2", "ok", "Address typed by you.")
	if get(rows["ied2"], "proposed") != "192.168.10.77" || get(rows["ied2"], "prefix") != 24 || !ready {
		t.Errorf("typed row %v ready %v", rows["ied2"], ready)
	}
	// One adapter listing and one ARP question for the two addresses; the check adds none.
	if fixture.listed != 1 || len(fixture.probes) != 1 || strings.Join(fixture.probes[0], ",") != "192.168.10.21,192.168.10.77" {
		t.Errorf("adapter listings %d, probes %v", fixture.listed, fixture.probes)
	}
}

func TestARPIsNotAskedAboutAddressesAlreadyRefused(t *testing.T) {
	fixture := newTypedFixture(scdIEDs("192.168.10.21", "192.168.10.22", "192.168.10.23"), labCard, wifiCard)
	manual := typeInto(typeInto(pyjson.NewObject(), "ied2", "192.168.10.21", int64(24)), "ied3", "172.16.1.20", int64(24))
	rows, _ := fixture.preview(t, StrategySCD, manual)
	wantStatus(t, rows, "ied2", "duplicate", "FEED1 already uses")
	wantStatus(t, rows, "ied3", "used_by_other_adapter", "'Wi-Fi'")
	if len(fixture.probes) != 1 || strings.Join(fixture.probes[0], ",") != "192.168.10.21" {
		t.Errorf("only FEED1's free address may be probed: %v", fixture.probes)
	}
}

func TestSpacesAroundATypedAddressAreIgnored(t *testing.T) {
	fixture := newTypedFixture(scdIEDs("192.168.10.21", "192.168.10.22"), labCard)
	manual := typeInto(typeInto(pyjson.NewObject(), "ied1", "192.168.10.50 ", int64(24)), "ied2", "\t 192.168.10.51", " 24 ")
	rows, ready := fixture.preview(t, StrategySCD, manual)
	wantStatus(t, rows, "ied1", "ok", "Address typed by you.")
	wantStatus(t, rows, "ied2", "ok", "Address typed by you.")
	if get(rows["ied1"], "proposed") != "192.168.10.50" || get(rows["ied2"], "proposed") != "192.168.10.51" || !ready {
		t.Errorf("rows %v %v ready %v", rows["ied1"], rows["ied2"], ready)
	}
	again := typeInto(pyjson.NewObject(), "ied1", " 192.168.10.22 ", int64(24))
	rows, _ = fixture.preview(t, StrategySCD, again)
	wantStatus(t, rows, "ied1", "duplicate", "FEED2 already uses 192.168.10.22")
	bad := typeInto(pyjson.NewObject(), "ied1", " not an address ", int64(24))
	rows, ready = fixture.preview(t, StrategySCD, bad)
	wantStatus(t, rows, "ied1", "invalid_manual", "Enter an IPv4 address")
	if ready {
		t.Error("an invalid address must stay refused")
	}
}

func TestADuplicateCausedByAReplacedPickIsCleared(t *testing.T) {
	fixture := newTypedFixture(scdIEDs("192.168.10.21", "192.168.10.21"), labCard) // same address twice in the file
	rows, ready := fixture.preview(t, StrategySCD, nil)
	wantStatus(t, rows, "ied2", "duplicate", "FEED1 already uses")
	if ready {
		t.Fatal("two IEDs with one file address are not ready")
	}
	manual := typeInto(pyjson.NewObject(), "ied1", "192.168.10.30", int64(24))
	rows, ready = fixture.preview(t, StrategySCD, manual)
	wantStatus(t, rows, "ied1", "ok", "Address typed by you.")
	wantStatus(t, rows, "ied2", "ok", "")
	if !ready {
		t.Error("typing a new address for the first IED must clear the second one's duplicate")
	}
}

func TestPlannedFreeAddressesStepAsideForATypedOne(t *testing.T) {
	fixture := newTypedFixture(scdIEDs("", ""), labCard) // adapter strategy picks .21 and .22
	rows, _ := fixture.preview(t, StrategyAdapter, nil)
	first, second := pyStr(get(rows["ied1"], "proposed")), pyStr(get(rows["ied2"], "proposed"))
	if first != "192.168.10.21" || second != "192.168.10.22" {
		t.Fatalf("planner picks %s %s", first, second)
	}
	manual := typeInto(pyjson.NewObject(), "ied1", second, int64(24)) // type the address FEED2 would get
	rows, ready := fixture.preview(t, StrategyAdapter, manual)
	wantStatus(t, rows, "ied1", "ok", "Address typed by you.")
	if get(rows["ied2"], "status") != "ok" || get(rows["ied2"], "proposed") == second || !ready {
		t.Errorf("the planned row must move away: %v ready %v", rows["ied2"], ready)
	}
}

func TestTypedSNMPRowsAreCheckedAmongThemselves(t *testing.T) {
	s, _ := snmpService()
	manual := typeInto(typeInto(pyjson.NewObject(), "snmp:snmp1", "192.168.10.70", int64(24)), "snmp:snmp2", "192.168.10.70", int64(24))
	rows, _, ready := previewRows(t, s, manual)
	wantStatus(t, rows, "snmp:snmp1", "ok", "")
	wantStatus(t, rows, "snmp:snmp2", "duplicate", "SNMP · Network switch 1 already uses 192.168.10.70. Each device needs its own address.")
	if ready {
		t.Error("two SNMP devices on one address must not be ready")
	}
}
