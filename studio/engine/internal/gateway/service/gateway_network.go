package service

import (
	"net"

	"studio-engine/internal/apperr"
	"studio-engine/internal/labnet"
	"studio-engine/internal/pyjson"
)

// The station gateway's Network (ADR-0124, Go engine only). "listen" keeps its two values that the
// Python engine also reads (this_pc, network); "listen_address" narrows network to one address of
// this PC; "allowed_scada" lets only that SCADA address connect (IEC 104 and Modbus TCP: opendnp3
// has no accept filter, and DNP3 already checks the master's link address).

func stringsOf(list []string) []any {
	result := []any{}
	for _, item := range list {
		result = append(result, item)
	}
	return result
}

// Network is what the gateway's Network field shows: the choice ("this_pc", "network" or one
// address), the addresses of this PC by card, the address and port SCADA connects to, and the
// allowed SCADA address.
func (m *Manager) Network() *pyjson.Object {
	config := m.Config()
	listen, address, allowed := text(config, "listen"), text(config, "listen_address"), text(config, "allowed_scada")
	choices := pcChoices(m.Adapters, m.projects.IEDs())
	list, adapterOf := []any{}, map[string]string{}
	for _, choice := range choices {
		list = append(list, pyjson.NewObject().Set("address", choice.Address).Set("adapter", choice.Adapter))
		adapterOf[choice.Address] = choice.Adapter
	}
	choice, connect, others, note := listen, "127.0.0.1", 0, ""
	switch {
	case listen != "network":
		choice = "this_pc"
	case isOwn(config, address):
		choice, connect = address, address
		adapterOf[address] = labnet.CardName
		if m.Lab == nil || !m.Lab.Ready() {
			note = "The ScadaBench Lab card is missing, so " + address + " cannot be used. Create it on the lab network page."
		}
	case address != "":
		choice, connect = address, address
		if _, found := adapterOf[address]; !found {
			note = "This PC has no address " + address + " now. Choose another network."
		}
	default:
		connect = windowsSource("192.0.2.1") // the default-route address (TEST-NET-1, nothing is sent)
		if _, found := adapterOf[connect]; !found && len(choices) > 0 {
			connect = choices[0].Address
		}
		others = len(choices) - 1
	}
	protocol := text(config, "protocol")
	settings, _ := get(config, "settings").(*pyjson.Object)
	port := get(get(settings, protocol).(*pyjson.Object), "port")
	return pyjson.NewObject().Set("choice", choice).Set("choices", list).Set("connect_to", connect).
		Set("connect_adapter", adapterOf[connect]).Set("others", int64(max(others, 0))).Set("port", port).
		Set("allowed", allowed).Set("allowed_works", protocol != "dnp3").Set("note", note).
		Set("own_addresses", stringsOf(ownAddresses(config))).Set("own", isOwn(config, address) && listen == "network").
		Set("lab_ready", m.Lab != nil && m.Lab.Ready())
}

// SaveNetwork stores the choice ("this_pc", "network" or an address of this PC) and the allowed
// SCADA address ("" lets anyone connect).
func (m *Manager) SaveNetwork(choice, allowed string) error {
	if m.State() == "running" {
		return apperr.ValueError("Stop the gateway before changing its network")
	}
	listen, address := choice, ""
	if choice != "this_pc" && choice != "network" {
		known := isOwn(m.Config(), choice)
		for _, item := range pcChoices(m.Adapters, m.projects.IEDs()) {
			known = known || item.Address == choice
		}
		if !known {
			return apperr.ValueError("This PC has no address " + choice + ". Choose one from the list.")
		}
		listen, address = "network", choice
	}
	if allowed != "" && net.ParseIP(allowed).To4() == nil {
		return apperr.ValueError("Type the SCADA address as IPv4, for example 192.168.1.50, or leave it empty")
	}
	previous := text(m.Config(), "listen_address")
	config := with(with(with(m.Config(), "listen", listen), "listen_address", address), "allowed_scada", allowed)
	if err := m.projects.SaveGateway(config); err != nil {
		return err
	}
	m.leaveOwn(previous, address)
	m.changed(ChangeGatewayStatus)
	return nil
}

// networkSettings narrows the bind address to the chosen card and adds the allowed SCADA address.
func (m *Manager) networkSettings(config, settings *pyjson.Object, bind string) (*pyjson.Object, error) {
	if address := text(config, "listen_address"); text(config, "listen") == "network" && isOwn(config, address) {
		if err := m.ensureOwn(address); err != nil {
			return nil, err
		}
		bind = address
	} else if text(config, "listen") == "network" && address != "" {
		known := false
		for _, item := range pcChoices(m.Adapters, m.projects.IEDs()) {
			known = known || item.Address == address
		}
		if !known {
			return nil, apperr.ValueError("This PC has no address " + address + " now. Choose another network in Network.")
		}
		bind = address
	}
	settings = with(settings, "bind", bind)
	if allowed := text(config, "allowed_scada"); allowed != "" && text(config, "protocol") != "dnp3" {
		settings = with(settings, "allowed", allowed)
	}
	return settings, nil
}
