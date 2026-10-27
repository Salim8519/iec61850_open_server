package service

import (
	"net"
	"strings"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

// The master's "Network" (ADR-0123, Go engine only): which address of this PC the SCADA master
// connects from. "" (All networks, the default) lets Windows choose by its routes, as before;
// otherwise studio_master binds the connection to that address. Saved in the master block as
// "local" beside the other Go-only keys; the frozen Python engine never reads it.

// NetworkChoice is one address of this PC the master can connect from.
type NetworkChoice struct{ Address, Adapter string }

// networkChoices are the usable IPv4 addresses of connected adapters: never a link-local
// (169.254) address, and never an address of a simulated IED: SkipAsSource on real cards, one of
// ieds, or anything on the ScadaBench Lab card, which holds only IED addresses (ADR-0120).
func networkChoices(adapters []netcfg.Adapter, ieds map[string]bool) []NetworkChoice {
	choices := []NetworkChoice{}
	for _, adapter := range adapters {
		if !strings.EqualFold(adapter.Status, "Up") || adapter.Name == labnet.CardName {
			continue
		}
		for _, address := range adapter.Addresses {
			if address.SkipAsSource || ieds[address.IP] || strings.HasPrefix(address.IP, "169.254.") || address.IP == "" {
				continue
			}
			choices = append(choices, NetworkChoice{Address: address.IP, Adapter: adapter.Name})
		}
	}
	return choices
}

func (m *Master) choices() []NetworkChoice {
	if m.Adapters == nil {
		return []NetworkChoice{}
	}
	adapters, err := m.Adapters()
	if err != nil {
		return []NetworkChoice{}
	}
	return networkChoices(adapters, m.iedAddresses())
}

// iedAddresses are the addresses the project's IEDs use: runtime, lab and SCL addresses.
func (m *Master) iedAddresses() map[string]bool {
	used := map[string]bool{}
	for _, ied := range m.projects.IEDs() {
		scl, _ := get(ied, "scl_address").(*pyjson.Object)
		for _, value := range []any{get(ied, "ip"), get(ied, "lab_ip"), get(scl, "ip")} {
			if address, ok := value.(string); ok && address != "" {
				used[address] = true
			}
		}
	}
	return used
}

// windowsSource is the address Windows would send from to reach host (a UDP "connect" only asks
// the routing table; nothing is sent). "" when the host cannot be resolved or reached.
func windowsSource(host string) string {
	if host == "" {
		return ""
	}
	conn, err := net.DialTimeout("udp4", net.JoinHostPort(host, "9"), 2*time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()
	if local, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return local.IP.String()
	}
	return ""
}

func localOf(config *pyjson.Object) string { return text(config, "local") }

// Network is what the "Network" field shows: the saved choice, the addresses of this PC, and
// the address the device will see (Windows' choice for All networks).
func (m *Master) Network() *pyjson.Object {
	config := m.Config()
	local, choices := localOf(config), m.choices()
	list, adapterOf := []any{}, map[string]string{}
	for _, choice := range choices {
		list = append(list, pyjson.NewObject().Set("address", choice.Address).Set("adapter", choice.Adapter))
		adapterOf[choice.Address] = choice.Adapter
	}
	source, note := local, ""
	if local == "" {
		source = windowsSource(text(config, "host"))
	} else if _, found := adapterOf[local]; !found {
		note = "This PC has no address " + local + " now. Choose another network, or All networks."
	} else if auto := windowsSource(text(config, "host")); auto != "" && auto != local {
		note = "Windows reaches this device through " + adapterName(adapterOf, auto) +
			". From " + local + " the connection may fail."
	}
	return pyjson.NewObject().Set("local", local).Set("choices", list).Set("source", source).
		Set("source_adapter", adapterOf[source]).Set("note", note)
}

func adapterName(adapterOf map[string]string, address string) string {
	if name := adapterOf[address]; name != "" {
		return name + " (" + address + ")"
	}
	return address
}

// SaveNetwork stores the chosen address ("" for All networks); only an address of this PC.
func (m *Master) SaveNetwork(local string) error {
	if m.Busy() {
		return apperr.ValueError("Disconnect before changing the network")
	}
	if local != "" {
		known := false
		for _, choice := range m.choices() {
			known = known || choice.Address == local
		}
		if !known {
			return apperr.ValueError("This PC has no address " + local + ". Choose one from the list.")
		}
	}
	return m.Save(with(m.Config(), "local", local))
}

// workerSettings adds the host and, when one is chosen, the address to connect from.
func workerSettings(settings *pyjson.Object, config *pyjson.Object) *pyjson.Object {
	result := with(settings, "host", get(config, "host"))
	if local := localOf(config); local != "" {
		result = with(result, "local", local)
	}
	return result
}

// withNetworkNote adds the Network warning to a failed connect from a chosen address, so a
// connection Windows cannot route says why (the same words for IEC 104, DNP3 and Modbus).
func (m *Master) withNetworkNote(err error) error {
	note := text(m.Network(), "note")
	problem, isApp := err.(*apperr.Error)
	if note == "" || !isApp || strings.Contains(problem.Message, "cannot reach") {
		return err
	}
	copied := *problem
	copied.Message = problem.Message + " " + note
	return &copied
}
