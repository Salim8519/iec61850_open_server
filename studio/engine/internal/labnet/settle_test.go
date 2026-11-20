package labnet

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// Change 0221, trial N6: with the cable out the service used to add the address (Windows could
// not check it), report it added and leave it on the card while no IED could listen on it. Now
// nothing is added and the reason names the card and the address.
func TestADisconnectedCardGetsNoAddress(t *testing.T) {
	f := newRealFake()
	f.down = true
	reply := Handle(f, Request{Op: OpAdd, IPs: []string{"192.168.230.21"}, Card: ethernet}, "3")
	reason := reply.Failed["192.168.230.21"]
	if reply.OK || !strings.Contains(reason, "“Ethernet” is disconnected") || !strings.Contains(reason, "192.168.230.21") {
		t.Fatalf("reply %+v", reply)
	}
	if len(f.addedReal) != 0 || contains(f.all[ethernet], "192.168.230.21") || len(f.settled) != 0 {
		t.Fatalf("added %v card %v settled %v", f.addedReal, f.all[ethernet], f.settled)
	}
	f.down = false // the cable is back: the next Start works
	if reply := Handle(f, Request{Op: OpAdd, IPs: []string{"192.168.230.21"}, Card: ethernet}, "3"); !reply.OK {
		t.Fatalf("after the cable is back: %+v", reply)
	}
	f.down = true // taking an address off works without a link
	if reply := Handle(f, Request{Op: OpRemove, IPs: []string{"192.168.230.21"}, Card: ethernet}, "3"); !reply.OK {
		t.Fatalf("remove without a link: %+v", reply)
	}
}

// probe is a card where some addresses stay tentative, one is a duplicate, the rest are usable.
type probe struct {
	tentative, duplicate map[string]bool
	removed              []string
	now                  time.Time
	sleeps               int
}

func (p *probe) Duplicate(_ RealCard, ip string) bool { return p.duplicate[ip] }
func (p *probe) Usable(ip string) bool                { return !p.tentative[ip] && !p.duplicate[ip] }
func (p *probe) RemoveReal(_ RealCard, ip string) error {
	p.removed = append(p.removed, ip)
	return nil
}
func (p *probe) clock() time.Time { return p.now }
func (p *probe) sleep(d time.Duration) {
	p.sleeps++
	p.now = p.now.Add(d)
}

// Before change 0221 an address Windows was still checking after 6 s was reported OK and stayed.
func TestStillCheckedAddressesAreTakenOffAndReported(t *testing.T) {
	p := &probe{tentative: map[string]bool{"10.1.1.2": true}, duplicate: map[string]bool{"10.1.1.3": true}}
	card := RealCard{Name: "Ethernet", Up: true}
	failed := settle(p, card, []string{"10.1.1.1", "10.1.1.2", "10.1.1.3"}, 6*time.Second, p.clock, p.sleep)
	if len(failed) != 2 || !strings.Contains(failed["10.1.1.3"], "already uses 10.1.1.3") {
		t.Fatalf("failed %v", failed)
	}
	want := "Windows was still checking 10.1.1.2 on “Ethernet” after 6 seconds, so ScadaBench took it off again. " +
		"Check the cable and the network switch, then press Start again"
	if failed["10.1.1.2"] != want {
		t.Fatalf("reason %q", failed["10.1.1.2"])
	}
	sort.Strings(p.removed)
	if !reflect.DeepEqual(p.removed, []string{"10.1.1.2", "10.1.1.3"}) {
		t.Fatalf("removed %v", p.removed)
	}
	if p.sleeps < 55 || p.sleeps > 61 {
		t.Fatalf("waited %d steps, want about 6 s", p.sleeps)
	}
}

// The happy path does not wait: every address usable at once means no sleep at all.
func TestSettleDoesNotWaitWhenEveryAddressIsUsable(t *testing.T) {
	p := &probe{}
	failed := settle(p, RealCard{Name: "Ethernet", Up: true}, []string{"10.1.1.1", "10.1.1.4"}, 6*time.Second, p.clock, p.sleep)
	if len(failed) != 0 || p.sleeps != 0 || len(p.removed) != 0 {
		t.Fatalf("failed %v sleeps %d removed %v", failed, p.sleeps, p.removed)
	}
	if failed := settle(p, RealCard{}, nil, 6*time.Second, p.clock, p.sleep); len(failed) != 0 || p.sleeps != 0 {
		t.Fatal("nothing to settle must not wait")
	}
}

// An address that becomes usable during the wait is fine and stays.
func TestAnAddressCheckedInTimeStays(t *testing.T) {
	p := &probe{tentative: map[string]bool{"10.1.1.2": true}}
	sleep := func(d time.Duration) {
		p.sleep(d)
		if p.sleeps == 30 { // Windows finished its check after 3 s
			delete(p.tentative, "10.1.1.2")
		}
	}
	failed := settle(p, RealCard{Name: "Ethernet", Up: true}, []string{"10.1.1.2"}, 6*time.Second, p.clock, sleep)
	if len(failed) != 0 || len(p.removed) != 0 || p.sleeps != 30 {
		t.Fatalf("failed %v removed %v sleeps %d", failed, p.removed, p.sleeps)
	}
}
