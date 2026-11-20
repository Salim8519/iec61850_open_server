package labnet

import (
	"reflect"
	"strings"
	"testing"
)

const ethernet = "{7B29C0B1-72E8-4C7D-9FAA-6A695893211D}"
const wifi = "{2E7EB2E1-9714-45FC-966C-FB239B857844}"

// realFake is fakeOps plus real cards: Ethernet (fixed) and Wi-Fi (DHCP).
type realFake struct {
	fakeOps
	approved  map[string]map[string]int
	addedReal []string
	duplicate string
	answering string          // a device answers ARP at this address
	unasked   map[string]bool // addresses Ask cannot ask (no own address on that network)
	settled   []string
	down      bool // Ethernet has no link (cable out)
}

func newRealFake() *realFake {
	f := &realFake{fakeOps: fakeOps{all: map[string][]string{card: {}, ethernet: {"192.168.40.111"},
		wifi: {"192.168.8.100"}}}}
	f.approved = map[string]map[string]int{ethernet: {"192.168.230.21": 24, "192.168.230.24": 24},
		wifi: {"192.168.8.21": 24}}
	return f
}

func (f *realFake) Cards() (map[string]RealCard, map[string][]string, error) {
	return map[string]RealCard{ethernet: {GUID: ethernet, Name: "Ethernet", Up: !f.down},
		wifi: {GUID: wifi, Name: "Wi-Fi", Up: true, DHCP: true}}, f.all, nil
}
func (f *realFake) Approved() map[string]map[string]int { return f.approved }
func (f *realFake) Ask(card RealCard, prefixes map[string]int) map[string]bool {
	answers := map[string]bool{}
	for ip := range prefixes {
		if !f.unasked[ip] && card.Up { // as Windows: no ARP question without a link
			answers[ip] = ip == f.answering
		}
	}
	return answers
}
func (f *realFake) AddReal(card RealCard, ip string, prefix int, asked bool) error {
	ready := map[bool]string{true: " ready", false: " checked by Windows"}[asked]
	f.addedReal = append(f.addedReal, card.Name+" "+ip+"/"+itoa(prefix)+ready)
	f.all[card.GUID] = append(f.all[card.GUID], ip)
	return nil
}
func (f *realFake) RemoveReal(card RealCard, ip string) error {
	f.all[card.GUID] = without(f.all[card.GUID], ip)
	return nil
}
func (f *realFake) Settle(card RealCard, ips []string) map[string]string {
	f.settled = append(f.settled, ips...)
	found := map[string]string{}
	for _, ip := range ips {
		if ip == f.duplicate {
			found[ip] = "Another device uses " + ip
			f.all[card.GUID] = without(f.all[card.GUID], ip)
		}
	}
	return found
}

func itoa(value int) string {
	digits := ""
	for ; value > 0; value /= 10 {
		digits = string(rune('0'+value%10)) + digits
	}
	return digits
}

func TestOnlyApprovedAddressesGoOnARealCard(t *testing.T) {
	f := newRealFake()
	reply := Handle(f, Request{Op: OpAdd, IPs: []string{"192.168.230.21", "192.168.230.99"}, Card: ethernet}, "3")
	if reply.OK || !strings.Contains(reply.Failed["192.168.230.99"], "not approved") {
		t.Fatalf("unapproved address accepted: %+v", reply)
	}
	if !reflect.DeepEqual(f.addedReal, []string{"Ethernet 192.168.230.21/24 ready"}) || len(f.added) != 0 {
		t.Fatalf("added %v (lab card %v)", f.addedReal, f.added)
	}
	reply = Handle(f, Request{Op: OpRemove, IPs: []string{"192.168.230.21", "192.168.40.111"}, Card: ethernet}, "3")
	if !reflect.DeepEqual(reply.Addresses, []string{"192.168.40.111"}) || reply.Failed["192.168.40.111"] == "" {
		t.Fatalf("the owner's own address was removed or allowed: %+v", reply)
	}
}

func TestNoFixedAddressOnADHCPCardOrOnAnotherCardsAddress(t *testing.T) {
	f := newRealFake()
	reply := Handle(f, Request{Op: OpAdd, IP: "192.168.8.21", Card: wifi}, "3")
	if reply.OK || !strings.Contains(reply.Message, "DHCP") || len(f.addedReal) != 0 {
		t.Fatalf("DHCP card changed: %+v", reply)
	}
	f.all[card] = []string{"192.168.230.24"}
	reply = Handle(f, Request{Op: OpAdd, IP: "192.168.230.24", Card: ethernet}, "3")
	if reply.OK || !strings.Contains(reply.Message, "another card") {
		t.Fatalf("an address of another card was added: %+v", reply)
	}
}

func TestADeviceThatAnswersKeepsItsAddress(t *testing.T) {
	f := newRealFake()
	f.answering = "192.168.230.24"
	reply := Handle(f, Request{Op: OpAdd, IPs: []string{"192.168.230.21", "192.168.230.24"}, Card: ethernet}, "3")
	if reply.OK || !strings.Contains(reply.Failed["192.168.230.24"], "already answers") ||
		!reflect.DeepEqual(f.addedReal, []string{"Ethernet 192.168.230.21/24 ready"}) || len(f.settled) != 0 {
		t.Fatalf("reply %+v added %v settled %v", reply, f.addedReal, f.settled)
	}
}

func TestADuplicateOnTheNetworkIsReportedAndTakenOff(t *testing.T) {
	f := newRealFake()
	f.duplicate = "192.168.230.24"
	f.unasked = map[string]bool{"192.168.230.24": true} // Windows checks this one
	reply := Handle(f, Request{Op: OpAdd, IPs: []string{"192.168.230.21", "192.168.230.24"}, Card: ethernet}, "3")
	if reply.OK || reply.Failed["192.168.230.24"] == "" || contains(reply.Addresses, "192.168.230.24") ||
		!contains(reply.Addresses, "192.168.230.21") {
		t.Fatalf("reply %+v", reply)
	}
}

func TestARealCardRequestNeedsAGUIDAndAServiceThatKnowsRealCards(t *testing.T) {
	if _, err := DecodeRequest([]byte(`{"op":"add","ip":"10.0.0.5","card":"Ethernet"}`)); err == nil {
		t.Fatal("a card name was accepted")
	}
	if _, err := DecodeRequest([]byte(`{"op":"status","card":"` + ethernet + `"}`)); err != nil {
		t.Fatal(err)
	}
	plain := &fakeOps{all: map[string][]string{card: {}}}
	if reply := Handle(plain, Request{Op: OpAdd, IP: "10.0.0.5", Card: ethernet}, "3"); reply.OK {
		t.Fatal("ops without real cards changed one")
	}
	if !RealCards("3") || RealCards("2") || RealCards("") {
		t.Fatal("RealCards version rule")
	}
}

// A DHCP card's address is the PC's own: an approved address there is never taken off (A6).
func TestNothingIsRemovedFromADHCPCard(t *testing.T) {
	f := newRealFake()
	f.all[wifi] = []string{"192.168.8.21"} // the router gave the card the approved address
	reply := Handle(f, Request{Op: OpRemove, IP: "192.168.8.21", Card: wifi}, "3")
	if reply.OK || !strings.Contains(reply.Message, "DHCP") || !contains(f.all[wifi], "192.168.8.21") {
		t.Fatalf("the DHCP card's own address was removed: %+v (card %v)", reply, f.all[wifi])
	}
}
