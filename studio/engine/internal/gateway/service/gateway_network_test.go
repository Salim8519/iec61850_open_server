package service

import (
	"reflect"
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

type fakeLab struct {
	ready bool
	card  map[string]bool
	err   error
	// held reads the owner's InUse report while EnsureOwn adds (trial N3: held before added)
	held   func() map[string]string
	heldAt []bool
}

func (l *fakeLab) Ready() bool { return l.ready }
func (l *fakeLab) CheckOwn(address, _ string) (string, error) {
	if l.err != nil {
		return "", l.err
	}
	return address, nil
}
func (l *fakeLab) EnsureOwn(address, _ string) error {
	if l.held != nil {
		_, found := l.held()[address]
		l.heldAt = append(l.heldAt, found)
	}
	l.card[address] = true
	return nil
}
func (l *fakeLab) RemoveOwn(address, _ string) { delete(l.card, address) }

// ADR-0125, amended by ADR-0141 (trial N9): a typed address is checked and stays in the list
// without touching the lab card; it goes on the card when the gateway starts (held before it is
// added, so no Release takes it off meanwhile, trial N3) and comes off when the gateway stops.
func TestGatewayOwnAddressOnTheLabCard(t *testing.T) {
	m := networkGateway("iec104")
	if err := m.AddOwnAddress("10.10.0.100"); err == nil {
		t.Fatal("no lab card: refused")
	}
	lab := &fakeLab{ready: true, card: map[string]bool{}}
	m.Lab = lab
	if err := m.AddOwnAddress("10.10.0.100"); err != nil || lab.card["10.10.0.100"] {
		t.Fatalf("a stopped gateway put its address on the card: %v %v", err, lab.card)
	}
	got := m.Network()
	if get(got, "choice") != "10.10.0.100" || get(got, "connect_to") != "10.10.0.100" || get(got, "own") != true ||
		len(get(got, "own_addresses").([]any)) != 1 {
		t.Fatalf("network %v", got)
	}
	if m.OwnAddresses()["10.10.0.100"] != OwnName {
		t.Fatal("the lab page does not see the gateway's address")
	}
	if len(m.CardAddresses()) != 0 {
		t.Fatal("a stopped gateway holds an address on the card")
	}
	lab.held = m.CardAddresses
	settings, err := m.networkSettings(m.Config(), pyjson.NewObject(), "0.0.0.0") // Start
	if err != nil || get(settings, "bind") != "10.10.0.100" || !lab.card["10.10.0.100"] {
		t.Fatalf("start: %v %v", settings, err)
	}
	if !reflect.DeepEqual(lab.heldAt, []bool{true}) || m.CardAddresses()["10.10.0.100"] != OwnName {
		t.Fatalf("not held while added: %v", lab.heldAt)
	}
	m.leaveCard() // Stop
	if lab.card["10.10.0.100"] || len(m.CardAddresses()) != 0 {
		t.Fatalf("still on the card after Stop: %v", lab.card)
	}
	if err := m.SaveNetwork("network", ""); err != nil {
		t.Fatal(err)
	}
	if err := m.SaveNetwork("10.10.0.100", ""); err != nil {
		t.Fatalf("a listed own address can be chosen again: %v", err)
	}
}
