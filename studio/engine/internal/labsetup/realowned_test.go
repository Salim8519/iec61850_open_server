package labsetup

import (
	"strings"
	"testing"

	"studio-engine/internal/netcfg"
)

// An approved address that is now the PC's own (set by the user, never journaled as added by
// ScadaBench) stays at Stop, launch and exit, and Run does not claim it (fix plan A6).
func TestAnApprovedAddressScadaBenchDidNotAddStays(t *testing.T) {
	_, s, requests := onEthernet(t)
	inner := s.Windows.Adapters
	ownSet := false
	s.Windows.Adapters = func() ([]netcfg.Adapter, error) {
		adapters, err := inner()
		if ownSet {
			adapters[1].Addresses = append(adapters[1].Addresses, netcfg.Address{IP: "192.168.40.21", Prefix: 24, SkipAsSource: true})
		}
		return adapters, err
	}
	ownSet = true // the user typed 192.168.40.21 as the card's own address in Windows
	s.Release()   // launch
	if err := s.EnsureForRun(eth, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	s.Release() // Stop, then exit
	s.Release()
	if len(*requests) != 0 {
		t.Fatalf("the PC's own address was touched: %v", *requests)
	}
}

// Windows forgets the service's addresses when it restarts. An address still journaled as added
// but gone from the card is settled, so the user may later give it to the card (A6).
func TestAnAddressGoneAfterARestartIsSettled(t *testing.T) {
	p, s, requests := onEthernet(t)
	s.EnsureForRun(eth, []string{"a"})
	p.running = []string{"a"}
	inner := s.Windows.Adapters
	ownSet := false
	s.Windows.Adapters = func() ([]netcfg.Adapter, error) {
		adapters, err := inner()
		adapters[1].Addresses = []netcfg.Address{{IP: "192.168.40.111", Prefix: 24}} // 40.21 is gone
		if ownSet {
			adapters[1].Addresses = append(adapters[1].Addresses, netcfg.Address{IP: "192.168.40.21", Prefix: 24, SkipAsSource: true})
		}
		return adapters, err
	}
	p.running = nil // the PC restarted: Windows forgot the address
	s.Release()     // launch after the restart: settled, nothing removed
	ownSet = true
	s.Release() // the user's own address now: still nothing removed
	if strings.Join(*requests, "|") != "add 192.168.40.21" {
		t.Fatalf("requests %v", *requests)
	}
	records, _ := s.Journal.Records()
	last := records[len(records)-1]
	if event, _ := last.Get("event"); event != "skipped" {
		t.Fatalf("last journal record %v", last)
	}
}

// A journal written before this fix may record the user's own address as added (Run sent an add
// for an address already there). Windows lists the card's own address as a source address, the
// service's never: such an address stays (A6).
func TestAnOlderJournalNeverRemovesTheCardsOwnAddress(t *testing.T) {
	_, s, requests := onEthernet(t)
	s.journal("applied", "real_address", "192.168.40.111", "D60", eth, "")
	s.Windows.Approved = func() map[string]map[string]int {
		return map[string]map[string]int{eth: {"192.168.40.111": 24}}
	}
	s.Release()
	if len(*requests) != 0 {
		t.Fatalf("the card's own address was removed: %v", *requests)
	}
}
