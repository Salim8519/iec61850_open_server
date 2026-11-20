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
	s := p.service(t)
	s.Journal = netcfg.NewJournal(t.TempDir()) // what ScadaBench added is journaled (A6)
	ethernet, requests := map[string]bool{"192.168.40.111": true}, &[]string{}
	inner, call := s.Windows.Adapters, s.Windows.Call
	s.Windows.Adapters = func() ([]netcfg.Adapter, error) {
		adapters, err := inner()
		adapters[1].Addresses = nil
		for address := range ethernet {
			adapters[1].Addresses = append(adapters[1].Addresses, netcfg.Address{IP: address, Prefix: 24,
				SkipAsSource: address != "192.168.40.111"}) // the service adds them as never-source
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
	s.Release() // a's address comes off at its Stop (keepForSession off), b's stays while b runs
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

// Another device already answers at an approved address: Run stops with the service's reason and
// the IED's name, not with the preflight's "not set up" (owner bench, 2026-11-03).
func TestARefusedRealAddressStopsTheRunWithItsReason(t *testing.T) {
	_, s, _ := onEthernet(t)
	call := s.Windows.Call
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		reply, err := call(request)
		if request.Card != "" && request.Op == labnet.OpAdd {
			reply.Failed = map[string]string{"192.168.40.22": "Another device on Ethernet already uses 192.168.40.22; ScadaBench took it off again"}
		}
		return reply, err
	}
	err := s.EnsureForRun(eth, []string{"a", "b"})
	want := "D60 cannot start on 192.168.40.22: Another device on Ethernet already uses 192.168.40.22; " +
		"ScadaBench took it off again. Stop the other device that uses this address, or give D60 another " +
		"address in Network > Set up lab network."
	if err == nil || err.Error() != want {
		t.Fatalf("error %v", err)
	}
	if err := s.EnsureForRun(eth, []string{"a"}); err != nil {
		t.Fatalf("an accepted address refused the Run: %v", err)
	}
}
