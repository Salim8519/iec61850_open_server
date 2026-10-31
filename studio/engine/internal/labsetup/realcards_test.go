package labsetup

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
)

const eth = "{ETH}"

// onEthernet is the pc with the project on Ethernet (a real card) and both IEDs' addresses
// approved there; the fake service changes the Ethernet list.
func onEthernet(t *testing.T) (*pc, *Service, *[]string) {
	p := newPC()
	p.mode, p.guid = "lab", eth
	p.ieds[0].LabIP, p.ieds[1].LabIP = "192.168.40.21", "192.168.40.22"
	p.version = "3"
	s := p.service()
	ethernet, requests := map[string]bool{"192.168.40.111": true}, &[]string{}
	inner, call := s.Windows.Adapters, s.Windows.Call
	s.Windows.Adapters = func() ([]netcfg.Adapter, error) {
		adapters, err := inner()
		adapters[1].Addresses = nil
		for address := range ethernet {
			adapters[1].Addresses = append(adapters[1].Addresses, netcfg.Address{IP: address, Prefix: 24})
		}
		return adapters, err
	}
	s.Windows.Approved = func() map[string]map[string]int {
		return map[string]map[string]int{eth: {"192.168.40.21": 24, "192.168.40.22": 24}}
	}
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		if request.Card == "" {
			return call(request)
		}
		*requests = append(*requests, request.Op+" "+strings.Join(request.Wanted(), ","))
		for _, address := range request.Wanted() {
			ethernet[address] = request.Op == labnet.OpAdd
			if request.Op == labnet.OpRemove {
				delete(ethernet, address)
			}
		}
		addresses := []string{}
		for address := range ethernet {
			addresses = append(addresses, address)
		}
		sort.Strings(addresses)
		return labnet.Reply{OK: true, Card: request.Card, Addresses: addresses, Version: "3"}, nil
	}
	return p, s, requests
}

func TestRealCardAddressesFollowStartAndStop(t *testing.T) {
	p, s, requests := onEthernet(t)
	if !s.RealCardsReady(eth) || s.RealCardsReady(card) {
		t.Fatal("real card readiness")
	}
	s.EnsureForRun(eth, []string{"a"})
	p.running = []string{"a"}
	s.EnsureForRun(eth, []string{"b"})
	p.running = []string{"a", "b"}
	if !reflect.DeepEqual(*requests, []string{"add 192.168.40.21", "add 192.168.40.22"}) {
		t.Fatalf("requests %v", *requests)
	}
	p.running = []string{"b"}
	s.Release()
	p.running = nil
	s.Release()
	want := []string{"add 192.168.40.21", "add 192.168.40.22", "remove 192.168.40.21", "remove 192.168.40.22"}
	if !reflect.DeepEqual(*requests, want) {
		t.Fatalf("requests %v", *requests)
	}
	s.Release() // nothing left: no request, and the owner's own address was never touched
	if len(*requests) != 4 {
		t.Fatalf("requests %v", *requests)
	}
}

func TestAnOlderServiceLeavesRealCardsAlone(t *testing.T) {
	p, s, requests := onEthernet(t)
	p.version = "2"
	if s.RealCardsReady(eth) {
		t.Fatal("protocol 2 service treated as able to change real cards")
	}
	s.EnsureForRun(eth, []string{"a", "b"})
	s.Release()
	if len(*requests) != 0 {
		t.Fatalf("requests %v", *requests)
	}
}
