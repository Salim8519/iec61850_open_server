package service

import (
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

func networkMaster() *Master {
	section := gateway.EmptyGateway().Set("role", "master").Set("master", gateway.EmptyMaster().Set("host", "127.0.0.1"))
	ied := pyjson.NewObject().Set("id", "f1").Set("ip", "127.50.0.2").Set("lab_ip", "10.10.10.21")
	m := NewMaster(&fakeProjects{ieds: []*pyjson.Object{ied}, section: section}, nil, "", nil)
	m.Adapters = func() ([]netcfg.Adapter, error) {
		return []netcfg.Adapter{
			{Name: "Ethernet", Status: "Up", Addresses: []netcfg.Address{{IP: "192.168.1.10"},
				{IP: "192.168.1.51", SkipAsSource: true}, {IP: "169.254.3.4"}}},
			{Name: "VMware Network Adapter VMnet2", Status: "Up", Addresses: []netcfg.Address{{IP: "10.10.10.1"}, {IP: "10.10.10.21"}}},
			{Name: "ScadaBench Lab", Status: "Up", Addresses: []netcfg.Address{{IP: "192.168.216.30"}}},
			{Name: "Wi-Fi", Status: "Disconnected", Addresses: []netcfg.Address{{IP: "10.0.0.5"}}},
		}, nil
	}
	return m
}

// ADR-0123: the list holds the PC's own usable addresses only: no IED address (SkipAsSource, an IED's
// address, anything on the ScadaBench Lab card), no link-local address, nothing on a disconnected card.
func TestMasterNetworkListsThePCsOwnAddresses(t *testing.T) {
	network := networkMaster().Network()
	text, _ := pyjson.Compact(get(network, "choices"))
	want := `[{"address":"192.168.1.10","adapter":"Ethernet"},{"address":"10.10.10.1","adapter":"VMware Network Adapter VMnet2"}]`
	if string(text) != want {
		t.Fatalf("got  %s\nwant %s", text, want)
	}
	if get(network, "local") != "" || get(network, "source") != "127.0.0.1" {
		t.Fatalf("All networks by default, Windows' address shown: %v", network)
	}
}

// Only an address of this PC is saved, and the worker gets it with the device address.
func TestMasterNetworkSavesOnlyAnAddressOfThisPC(t *testing.T) {
	m := networkMaster()
	if err := m.SaveNetwork("10.0.0.5"); err == nil {
		t.Fatal("an address of a card that is not connected is refused")
	}
	if err := m.SaveNetwork("10.10.10.1"); err != nil {
		t.Fatal(err)
	}
	settings := workerSettings(pyjson.NewObject().Set("port", int64(2404)), m.Config())
	if get(settings, "local") != "10.10.10.1" || get(settings, "host") != "127.0.0.1" {
		t.Fatalf("worker settings: %v", settings)
	}
	if note := text(m.Network(), "note"); note == "" {
		t.Fatal("a network that does not reach the device says so")
	}
	if err := m.SaveNetwork(""); err != nil || get(workerSettings(pyjson.NewObject(), m.Config()), "local") != nil {
		t.Fatal("All networks sends no local address")
	}
}
