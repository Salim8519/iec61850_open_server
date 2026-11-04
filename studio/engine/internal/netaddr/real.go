package netaddr

import (
	"net/netip"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

// real reviews the address plan of the project's real card with every address kept as it is and
// this one changed, then applies it: Windows approves the new address once and the network
// service adds it when the IED or device starts (ADR-0132, ADR-0136).
func (s *Service) real(guid, key, ip, name, when string) (*pyjson.Object, error) {
	adapters, err := s.Adapters()
	if err != nil {
		return nil, err
	}
	var card netcfg.Adapter
	for _, adapter := range adapters {
		if strings.EqualFold(adapter.GUID, guid) {
			card = adapter
		}
	}
	if card.GUID == "" {
		return nil, apperr.New("The project's network card is not on this PC now. Open Settings › Network " +
			"and choose a card.")
	}
	manual := pyjson.NewObject()
	ieds, err := s.IEDs()
	if err != nil {
		return nil, err
	}
	for _, ied := range ieds {
		if ied.LabIP != "" {
			manual.Set(ied.ID, typed(card, ied.LabIP))
		}
	}
	for _, device := range s.SNMP() {
		if device.LabIP != "" {
			manual.Set(netcfg.SNMPRowPrefix+device.ID, typed(card, device.LabIP))
		}
	}
	manual.Set(key, typed(card, ip))
	preview, err := s.Preview(guid, netcfg.StrategySCD, manual)
	if err != nil {
		return nil, err
	}
	if ready, _ := preview.Get("ready"); ready != true {
		return nil, apperr.New(problem(preview, key))
	}
	fingerprint, _ := preview.Get("fingerprint")
	applied, err := s.Apply(fingerprint, true)
	if err != nil {
		return nil, err
	}
	if ok, _ := applied.Get("applied"); ok != true {
		message, _ := applied.Get("message")
		text, _ := message.(string)
		return nil, apperr.New(text)
	}
	return done(name + " will answer at " + ip + " on " + card.Name + " " + when + "."), nil
}

// typed is one manual address with the prefix of the card's own network that holds it (24 when
// the card has no address on that network).
func typed(card netcfg.Adapter, ip string) *pyjson.Object {
	prefix := 24
	if address, err := netip.ParseAddr(ip); err == nil {
		for _, own := range card.Addresses {
			if network, err := netip.ParseAddr(own.IP); err == nil {
				if within, err := network.Prefix(own.Prefix); err == nil && within.Contains(address) {
					prefix = own.Prefix
					break
				}
			}
		}
	}
	return pyjson.NewObject().Set("ip", ip).Set("prefix", prefix)
}

// problem is the plain reason the plan is not ready: the changed row's first, else the first
// other row that blocks it.
func problem(preview *pyjson.Object, key string) string {
	raw, _ := preview.Get("rows")
	rows, _ := raw.([]any)
	var first string
	for _, item := range rows {
		row, ok := item.(*pyjson.Object)
		if !ok {
			continue
		}
		status, _ := row.Get("status")
		if status == "ok" || status == "already_on_adapter" {
			continue
		}
		name, _ := row.Get("name")
		hint, _ := row.Get("hint")
		text := pyStr(name) + ": " + pyStr(hint)
		if id, _ := row.Get("ied_id"); id == key {
			return text
		}
		if first == "" {
			first = text
		}
	}
	if first == "" {
		return "The address plan is not ready. Open Settings › Network to see why."
	}
	return first
}

func pyStr(value any) string {
	text, _ := value.(string)
	return text
}
