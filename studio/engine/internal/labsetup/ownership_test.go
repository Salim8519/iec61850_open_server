package labsetup

import (
	"reflect"
	"strings"
	"testing"
)

// ADR-0141 (2026-11-20 trials finding): Release takes off only the Lab-card addresses this copy's
// journal says it added. Before, it removed every address no IED of this copy used, also the ones
// another ScadaBench copy (installed and development, or a trial engine) had just added, and that
// copy's Run failed with "MMS bind failed" (7 times in one night).
func TestReleaseLeavesAnotherCopysAddresses(t *testing.T) {
	p := newPC()
	s := p.service(t)
	s.Use(nil)
	p.ours("10.9.9.1")               // this copy's leftover
	p.cardIPs["198.18.24.10"] = true // another copy's running IED
	if err := s.Release(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.cardIPs, map[string]bool{"198.18.24.10": true}) {
		t.Fatalf("card %v", p.cardIPs)
	}
	if err := s.EndSession(); err != nil || !p.cardIPs["198.18.24.10"] {
		t.Fatalf("exit removed another copy's address: %v %v", err, p.cardIPs)
	}
}

// A journaled address that is gone (Windows restarted, another copy took it off) is settled once,
// so the same address added later by another copy is never taken for ours.
func TestAGoneAddressIsSettledAndNeverTakenForOurs(t *testing.T) {
	p := newPC()
	s := p.service(t)
	p.ours("10.9.9.1")
	delete(p.cardIPs, "10.9.9.1") // gone without our journal knowing
	s.Release()
	if events := journalEvents(t, s); !reflect.DeepEqual(events, []string{"applied", "skipped"}) {
		t.Fatalf("journal %v", events)
	}
	p.cardIPs["10.9.9.1"] = true // another copy uses it now
	s.Release()
	if !p.cardIPs["10.9.9.1"] {
		t.Fatal("another copy's address was taken off")
	}
}

// Trial N10: Stop keeps the IED addresses for the session, the next Run adds nothing, and another
// project (or This PC only) takes the old project's addresses off.
func TestAddressesStayForTheSessionAndLeaveWithTheProject(t *testing.T) {
	if !keepForSession {
		t.Skip("addresses are not kept for the session while keepForSession is off (owner, 2026-11-20)")
	}
	p := newPC()
	s := p.service(t)
	s.Use(nil)
	s.Runtime.Start([]string{"a", "b"})
	s.Runtime.Stop(nil)
	p.calls = nil
	s.Runtime.Start([]string{"a", "b"})
	s.Runtime.Stop(nil)
	for _, call := range p.calls {
		if strings.HasPrefix(call, "add") || strings.HasPrefix(call, "remove") {
			t.Fatalf("a second Run/Stop changed the card: %v", p.calls)
		}
	}
	p.ieds = []IED{{ID: "c", Name: "REF615", LabIP: "10.30.0.1"}} // another project opened
	if err := s.Release(); err != nil || len(p.cardIPs) != 0 {
		t.Fatalf("the old project's addresses stayed: %v %v", err, p.cardIPs)
	}
}

// Trial N10: after a kill, the next launch (a new session over the same journal) takes the
// addresses the killed session kept off.
func TestALaunchAfterAKillTakesTheSessionsAddressesOff(t *testing.T) {
	if !keepForSession {
		t.Skip("addresses are not kept for the session while keepForSession is off (owner, 2026-11-20)")
	}
	p := newPC()
	s := p.service(t)
	s.Use(nil)
	s.Runtime.Start([]string{"a"})
	s.Runtime.Stop(nil)
	if !p.cardIPs["192.168.216.1"] {
		t.Fatal("not kept for the session")
	}
	relaunched := p.service(t) // killed: no exit ran; same data folder
	if err := relaunched.Release(); err != nil || len(p.cardIPs) != 0 {
		t.Fatalf("left after kill and relaunch: %v %v", err, p.cardIPs)
	}
}

// Trial N9/N3: an owner's address stays while the owner holds it (InUse), even though no IED uses
// it; RemoveOwn takes off only an address this copy added.
func TestOwnAddressesFollowTheirOwner(t *testing.T) {
	p := newPC()
	s := p.service(t)
	p.inUse = map[string]string{"10.10.0.100": gateway}
	if err := s.EnsureOwn("10.10.0.100", gateway); err != nil {
		t.Fatal(err)
	}
	s.Release()
	if !p.cardIPs["10.10.0.100"] {
		t.Fatal("Release took a held own address off")
	}
	p.inUse = map[string]string{}
	s.RemoveOwn("10.10.0.100", gateway)
	if p.cardIPs["10.10.0.100"] {
		t.Fatal("not removed at the owner's stop")
	}
	p.cardIPs["10.10.0.101"] = true // another copy's gateway
	s.RemoveOwn("10.10.0.101", gateway)
	if !p.cardIPs["10.10.0.101"] {
		t.Fatal("another copy's address was removed")
	}
}
