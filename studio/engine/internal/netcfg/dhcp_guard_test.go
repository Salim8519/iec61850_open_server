package netcfg

import (
	"strings"
	"testing"
)

func guardRows() []Row {
	return []Row{{IEDID: "a", Proposed: "192.168.0.21", Status: "ok"},
		{IEDID: "b", Proposed: "192.168.8.100", Status: "already_on_adapter"},
		{IEDID: "c", Proposed: "192.168.0.22", Status: "in_use_on_network", Hint: "taken"}}
}

func TestDHCPCardGetsNoFixedAddress(t *testing.T) {
	asked := ""
	service := &NetworkService{DHCP: func(guid string) (bool, error) { asked = guid; return true, nil }}
	rows := guardRows()
	if err := service.guardDHCP(rows, Adapter{GUID: "{W}", Name: "Wi-Fi"}); err != nil {
		t.Fatal(err)
	}
	if asked != "{W}" || rows[0].Status != StatusAdapterDHCP || rows[0].usable() {
		t.Fatalf("the new address stays usable on a DHCP card: %+v", rows[0])
	}
	if !strings.Contains(rows[0].Hint, "\"Wi-Fi\" gets its address automatically (DHCP)") {
		t.Fatalf("hint does not name the card: %q", rows[0].Hint)
	}
	if rows[1].Status != "already_on_adapter" || rows[2].Hint != "taken" {
		t.Fatalf("other rows changed: %+v", rows[1:])
	}
}

func TestFixedCardKeepsItsRows(t *testing.T) {
	for _, service := range []*NetworkService{{}, {DHCP: func(string) (bool, error) { return false, nil }}} {
		rows := guardRows()
		if err := service.guardDHCP(rows, Adapter{GUID: "{E}", Name: "Ethernet"}); err != nil {
			t.Fatal(err)
		}
		if rows[0].Status != "ok" {
			t.Fatalf("a fixed card lost its row: %+v", rows[0])
		}
	}
}
