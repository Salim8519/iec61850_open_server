package labsetup

import (
	"reflect"
	"strings"
	"testing"

	"studio-engine/internal/labnet"
)

// stuckService makes the fake service stop answering from its first add on (the answer is lost
// and so is every answer after it) until finish is called; finish then does the adds it had
// taken, late, as the real service does when it was busy (change 0227).
func stuckService(p *pc, s *Service) (finish func()) {
	call, stuck, late := s.Windows.Call, false, []string{}
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		if request.Op == labnet.OpAdd && !stuck {
			stuck, late = true, append(late, request.Wanted()...)
		}
		if stuck {
			p.calls = append(p.calls, request.Op+" (no answer)")
			return labnet.Reply{}, errLost
		}
		return call(request)
	}
	return func() {
		stuck = false
		s.Windows.Call = call
		for _, address := range late {
			call(labnet.Request{Op: labnet.OpAdd, IP: address})
		}
	}
}

// Change 0227: an IED's add whose answer was lost while the service was busy, so neither the
// service nor Windows listed the address yet; the service put it on later. Before, the add was
// journaled "failed" and, since this copy removes only what its journal says it added (ADR-0141),
// the address stayed on the card until Windows restarted. Now the add stays "requested" and the
// next Release settles it from the card's list: on the card, so taken off.
func TestALateLabAddIsTakenOffAtTheNextRelease(t *testing.T) {
	p := newPC()
	s := p.service(t)
	s.Use(nil)
	finish := stuckService(p, s)
	err := s.EnsureForRun(card, []string{"a"})
	if err == nil || !strings.HasPrefix(err.Error(), "P545 cannot start on 192.168.216.1: the lab network service did not answer.") {
		t.Fatalf("error %v", err)
	}
	finish() // the service finishes the add after the app gave up
	if !p.cardIPs["192.168.216.1"] {
		t.Fatal("the late add did not happen")
	}
	if err := s.Release(); err != nil { // the Stop after the refused Run
		t.Fatal(err)
	}
	if p.cardIPs["192.168.216.1"] {
		t.Fatalf("the late address stayed on the card: journal %v", journalEvents(t, s))
	}
	if events := journalEvents(t, s); !reflect.DeepEqual(events, []string{"requested", "removed"}) {
		t.Fatalf("journal %v", events)
	}
}

// The gateway's own address: Start's add loses its answer, the failed start takes the address off
// at once while the service is still busy (that removal's answer is lost too), and the service
// puts the address on afterwards. The removal is not counted as done (only the service's list
// proves an address is gone), so the next Release takes the address off.
func TestALateOwnAddressIsTakenOffAfterTheFailedStart(t *testing.T) {
	p := newPC()
	s := p.service(t)
	finish := stuckService(p, s)
	if err := s.EnsureOwn("10.10.0.100", gateway); err == nil {
		t.Fatal("a lost add was reported as on the card")
	}
	s.RemoveOwn("10.10.0.100", gateway) // leaveCard after the failed start
	finish()
	if err := s.Release(); err != nil {
		t.Fatal(err)
	}
	if p.cardIPs["10.10.0.100"] {
		t.Fatalf("the late own address stayed on the card: journal %v calls %v", journalEvents(t, s), p.calls)
	}
}

// An add the service answers no more, that never reaches the card: the next Release, with the
// service answering again, settles it as not there, and the same address put on later by another
// copy is never taken for ours.
func TestALostAddThatNeverLandsIsSettledOnce(t *testing.T) {
	p := newPC()
	s := p.service(t)
	call := s.Windows.Call
	stuckService(p, s)                 // the add is lost and never done
	s.EnsureOwn("10.10.0.100", gateway) // its failed start
	s.Windows.Call = call
	s.Release()
	if events := journalEvents(t, s); !reflect.DeepEqual(events, []string{"requested", "skipped"}) {
		t.Fatalf("journal %v", events)
	}
	p.cardIPs["10.10.0.100"] = true // another copy's now
	s.Release()
	if !p.cardIPs["10.10.0.100"] {
		t.Fatal("another copy's address was taken off")
	}
}
