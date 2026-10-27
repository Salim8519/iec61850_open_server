package service

import (
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

func networkGateway(protocol string) *Manager {
	section := gateway.EmptyGateway().Set("protocol", protocol)
	m := NewManager(&fakeProjects{section: section}, newHub(), nil, "", nil)
	m.Adapters = func() ([]netcfg.Adapter, error) {
		return []netcfg.Adapter{{Name: "Ethernet", Status: "Up", Addresses: []netcfg.Address{{IP: "192.168.1.10"}}}}, nil
	}
	return m
}

// ADR-0124: This PC only by default; one card of this PC binds there; the allowed SCADA address
// goes to IEC 104 and Modbus, never to DNP3 (opendnp3 cannot refuse a caller by address).
func TestGatewayNetworkBindsTheChosenCardAndAllowedScada(t *testing.T) {
	m := networkGateway("iec104")
	if got := m.Network(); get(got, "choice") != "this_pc" || get(got, "connect_to") != "127.0.0.1" {
		t.Fatalf("default: %v", got)
	}
	if err := m.SaveNetwork("10.9.9.9", ""); err == nil {
		t.Fatal("an address that is not on this PC is refused")
	}
	if err := m.SaveNetwork("192.168.1.10", "192.168.1.50x"); err == nil {
		t.Fatal("a SCADA address that is not IPv4 is refused")
	}
	if err := m.SaveNetwork("192.168.1.10", "192.168.1.50"); err != nil {
		t.Fatal(err)
	}
	config := m.Config()
	if get(config, "listen") != "network" || get(config, "listen_address") != "192.168.1.10" {
		t.Fatalf("stored: %v", config)
	}
	settings, err := m.networkSettings(config, pyjson.NewObject(), "0.0.0.0")
	if err != nil || get(settings, "bind") != "192.168.1.10" || get(settings, "allowed") != "192.168.1.50" {
		t.Fatalf("worker settings %v %v", settings, err)
	}
	dnp3, _ := networkGateway("dnp3").networkSettings(with(config, "protocol", "dnp3"), pyjson.NewObject(), "0.0.0.0")
	if get(dnp3, "allowed") != nil {
		t.Fatal("DNP3 gets no allowed address")
	}
	if err := m.SaveNetwork("network", ""); err != nil || get(m.Config(), "listen_address") != "" {
		t.Fatal("All networks clears the card")
	}
}
