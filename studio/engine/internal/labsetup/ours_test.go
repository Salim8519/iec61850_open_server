package labsetup

import (
	"strings"
	"testing"

	"studio-engine/internal/netcfg"
)

// withEthernetAddress puts D60's address (10.20.0.5) on Ethernet, as an earlier "Put your
// published IEDs on a network" on Ethernet did, and records what FreeOurs removes.
func withEthernetAddress(t *testing.T, p *pc, journaled bool) (*Service, *[]string) {
	s := p.service(t)
	present, freed := true, []string{}
	inner := s.Windows.Adapters
	s.Windows.Adapters = func() ([]netcfg.Adapter, error) {
		adapters, err := inner()
		if present {
			adapters[1].Addresses = append(adapters[1].Addresses, netcfg.Address{IP: "10.20.0.5", Prefix: 24})
		}
		return adapters, err
	}
	s.Windows.Ours = func() map[string][]string {
		if !journaled {
			return map[string][]string{}
		}
		return map[string][]string{"10.20.0.5": {"{WIFI}", "{ETH}"}, "10.1.1.1": {"{ETH}"}}
	}
	s.Windows.FreeOurs = func(addresses map[string]bool) error {
		for address := range addresses {
			freed = append(freed, address)
		}
		present = false
		return nil
	}
	return s, &freed
}

func TestUseTakesScadaBenchsOwnAddressOffTheOtherCard(t *testing.T) {
	p := newPC()
	s, freed := withEthernetAddress(t, p, true)
	status, err := s.Status(nil)
	if err != nil {
		t.Fatal(err)
	}
	row := byID(status.Rows)["b"]
	if row.Status != StatusOurs || row.Blocking() || !strings.Contains(row.Hint, "on “Ethernet” earlier") {
		t.Fatalf("row %+v", row)
	}
	if _, err := s.Use(nil); err != nil {
		t.Fatal(err)
	}
	if len(*freed) != 1 || (*freed)[0] != "10.20.0.5" || p.mode != "lab" || p.ieds[1].LabIP != "10.20.0.5" {
		t.Fatalf("freed %v, project %+v", *freed, p)
	}
}

func TestAnAddressScadaBenchDidNotAddStillBlocks(t *testing.T) {
	p := newPC()
	s, freed := withEthernetAddress(t, p, false)
	if _, err := s.Use(nil); err == nil || !strings.Contains(err.Error(), "is the address of “Ethernet”") {
		t.Fatalf("Use with someone else's address: %v", err)
	}
	if len(*freed) != 0 || p.mode != "loopback" {
		t.Fatalf("something changed: freed %v, mode %s", *freed, p.mode)
	}
}
