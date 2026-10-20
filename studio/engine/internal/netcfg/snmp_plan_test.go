package netcfg

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// SNMP devices in Network setup (ADR-0092 amendment): same card as the IEDs, free addresses
// checked with ARP, saved addresses kept, their own UDP firewall rule.

type fakeSNMP struct {
	devices []SNMPDevice
	saved   map[string]string
}

func (f *fakeSNMP) SNMPDevices() ([]SNMPDevice, error) { return f.devices, nil }
func (f *fakeSNMP) SetSNMPLabIPs(addresses map[string]string) error {
	f.saved = addresses
	return nil
}

func snmpService(answering ...string) (*NetworkService, *fakeSNMP) {
	index := int64(7)
	adapter := Adapter{GUID: "{AB12}", Name: "Ethernet 2", Status: "Up", Index: &index,
		Addresses: []Address{{IP: "192.168.10.5", Prefix: 24}}}
	busy := map[string]bool{}
	for _, address := range answering {
		busy[address] = true
	}
	snmp := &fakeSNMP{devices: []SNMPDevice{{ID: "snmp1", Name: "Network switch 1", Port: 161},
		{ID: "snmp2", Name: "GPS time server 1", LabIP: "192.168.10.60", Port: 161}}}
	service := &NetworkService{
		Projects: &fakeProjects{ieds: []PlanIED{{ID: "ied1", Name: "Bay1", SCLIP: "192.168.10.21", Port: 102}}, mode: "loopback"},
		Program:  `C:\app\native\studio_ied.exe`, Source: AdapterFunc(func() ([]Adapter, error) { return []Adapter{adapter}, nil }),
		Rules: func() ([]string, error) { return nil, nil }, SNMPRules: func() ([]string, error) { return nil, nil },
		Probe: ProberFunc(func(addresses []string) (map[string]bool, error) {
			found := map[string]bool{}
			for _, address := range addresses {
				found[address] = busy[address]
			}
			return found, nil
		}),
		Clock: time.Now, SNMP: snmp}
	return service, snmp
}

func previewRows(t *testing.T, s *NetworkService, manual any) (map[string]*pyjson.Object, []*pyjson.Object, bool) {
	t.Helper()
	result, state, err := s.preview("{AB12}", StrategySCD, manual)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]*pyjson.Object{}
	for _, row := range RowsObject(state.rows) {
		object := row.(*pyjson.Object)
		rows[pyStr(get(object, "ied_id"))] = object
	}
	return rows, state.items, truthy(get(result, "ready"))
}

func TestSNMPDevicesGetAddressesOnTheIEDCard(t *testing.T) {
	rows, items, ready := previewRows(t, func() *NetworkService { s, _ := snmpService(); return s }(), nil)
	if !ready || get(rows["ied1"], "proposed") != "192.168.10.21" {
		t.Fatalf("IED row changed: %v", rows["ied1"])
	}
	free, saved := pyStr(get(rows["snmp:snmp1"], "proposed")), rows["snmp:snmp2"]
	if !strings.HasPrefix(free, "192.168.10.") || free == "192.168.10.21" || free == "192.168.10.5" || free == "192.168.10.60" {
		t.Fatalf("switch got %q", free)
	}
	if get(saved, "proposed") != "192.168.10.60" || get(saved, "saved") != true || get(saved, "hint") != "Saved address of this device." {
		t.Fatalf("GPS row: %v", saved)
	}
	if get(rows["snmp:snmp1"], "kind") != "snmp" || get(rows["snmp:snmp1"], "name") != "SNMP · Network switch 1" {
		t.Fatalf("switch row: %v", rows["snmp:snmp1"])
	}
	kinds := []string{}
	for _, item := range items {
		kinds = append(kinds, pyStr(get(item, "kind")))
		if get(item, "kind") == "firewall" && len(get(item, "addresses").([]any)) != 1 {
			t.Fatalf("the MMS rule must cover only the IED: %v", item)
		}
		if get(item, "kind") == "snmp_firewall" {
			if len(get(item, "addresses").([]any)) != 2 || !strings.HasSuffix(pyStr(get(item, "program")), `native\studio_snmp.exe`) {
				t.Fatalf("SNMP rule: %v", item)
			}
			command, err := ApplyCommand(item)
			if err != nil || !strings.Contains(command, "-Protocol UDP -LocalPort 161 ") {
				t.Fatalf("SNMP rule command %q %v", command, err)
			}
			if err := validateItem(item, `C:\app\native\studio_ied.exe`); err != nil {
				t.Fatalf("helper refuses the SNMP rule: %v", err)
			}
			if err := validateItem(item, `C:\other\studio_ied.exe`); err == nil {
				t.Fatal("helper accepted an SNMP rule for another program")
			}
		}
	}
	if strings.Join(kinds, ",") != "alias,alias,alias,firewall,snmp_firewall" {
		t.Fatalf("plan kinds %v", kinds)
	}
}

func TestSavedSNMPAddressInUseIsReplacedByAFreeOne(t *testing.T) {
	s, _ := snmpService("192.168.10.60")
	rows, _, ready := previewRows(t, s, nil)
	gps := rows["snmp:snmp2"]
	if !ready || get(gps, "proposed") == "192.168.10.60" || get(gps, "saved") != false || get(gps, "status") != "ok" {
		t.Fatalf("GPS row: %v (ready %v)", gps, ready)
	}
}

func TestTypedSNMPAddressOfAnIEDIsRefusedWithAFreeAlternative(t *testing.T) {
	s, _ := snmpService()
	manual := pyjson.NewObject().Set("snmp:snmp1", pyjson.NewObject().Set("ip", "192.168.10.21").Set("prefix", int64(24)))
	rows, _, ready := previewRows(t, s, manual)
	row := rows["snmp:snmp1"]
	if ready || get(row, "status") != "duplicate" || !strings.Contains(pyStr(get(row, "hint")), "Bay1 already uses 192.168.10.21") {
		t.Fatalf("typed duplicate: %v", row)
	}
	if alternative, _ := get(row, "alternative").(*pyjson.Object); alternative == nil || get(alternative, "ip") == "192.168.10.21" {
		t.Fatalf("no free alternative: %v", row)
	}
}

func TestOnlySNMPDevicesCanGoOnTheNetworkAndIEDOnlyPlansAreUnchanged(t *testing.T) {
	s, _ := snmpService()
	s.Projects = &fakeProjects{mode: "loopback"}
	if _, items, ready := previewRows(t, s, nil); !ready || len(items) != 3 || get(items[2], "kind") != "snmp_firewall" {
		t.Fatalf("SNMP-only plan: %v", items)
	}
	s, _ = snmpService()
	s.SNMP = nil
	if rows, items, _ := previewRows(t, s, nil); len(rows) != 1 || len(items) != 2 {
		t.Fatalf("plan without SNMP devices changed: %v %v", rows, items)
	}
}

func TestApplySplitsIEDAndSNMPAddresses(t *testing.T) {
	ieds, snmp := splitRows([]Row{{IEDID: "ied1", Proposed: "192.168.10.21"}, {IEDID: "snmp:snmp1", Proposed: "192.168.10.61"}})
	if len(ieds) != 1 || ieds[0].IEDID != "ied1" || len(snmp) != 1 || snmp["snmp1"] != "192.168.10.61" {
		t.Fatalf("split %v %v", ieds, snmp)
	}
	text, err := Describe(pyjson.NewObject().Set("kind", "snmp_firewall").Set("action", "add").Set("ports", []any{int64(161)}))
	if err != nil || text != "Allow other devices to reach the simulated SNMP devices on UDP port 161 through Windows Firewall" {
		t.Fatalf("description %q %v", text, err)
	}
}
