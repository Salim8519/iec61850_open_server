package netcfg

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// Only journaled alias addresses still on a card are offered (ADR-0136); the user's own address
// and an alias already gone are not, and removal needs the exact list confirmed.
func TestLeftoverAddressesAreTheJournaledAliasesStillThere(t *testing.T) {
	directory := t.TempDir()
	ethernet := Adapter{GUID: "{ETH}", Name: "Ethernet", Addresses: []Address{{IP: "192.168.230.146", Prefix: 24},
		{IP: "192.168.230.21", Prefix: 24}}}
	service := &NetworkService{Directory: directory, Journal: NewJournal(directory),
		Source: AdapterFunc(func() ([]Adapter, error) { return []Adapter{ethernet}, nil }), Clock: time.Now}
	for _, ip := range []string{"192.168.230.21", "192.168.230.22"} {
		service.Journal.Append("applied", pyjson.NewObject().Set("kind", "alias").Set("owned", true).
			Set("adapter_guid", "{ETH}").Set("adapter_name", "Ethernet").Set("adapter_index", 14).
			Set("ip", ip).Set("prefix", 24).Set("for", "SNMP · SESW-A").Set("action", "add"), "f1", "")
	}
	listed, err := service.LeftoverAddresses()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := listed.Get("items")
	items := objects(raw)
	if len(items) != 1 || get(items[0], "ip") != "192.168.230.21" || get(items[0], "action") != "remove" ||
		get(listed, "count") != 1 {
		t.Fatalf("leftovers %v", items)
	}
	if _, err := service.RemoveLeftoverAddresses(get(listed, "fingerprint"), false); err == nil {
		t.Fatal("removed without confirmation")
	}
	if _, err := service.RemoveLeftoverAddresses("old", true); err == nil || !strings.Contains(err.Error(), "different now") {
		t.Fatalf("stale list: %v", err)
	}
}
