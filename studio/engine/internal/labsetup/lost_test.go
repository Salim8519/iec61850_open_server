package labsetup

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
)

var errLost = errors.New("the lab network service did not answer")

// Change 0221, trial N2: the service puts a real-card address on but its answer is lost. Before,
// nothing was journaled and Stop never took the address off (it removes only what ScadaBench
// added, A6). Now the add is journaled "requested", the card is read, and Stop takes it off.
func TestALostRealAnswerIsSettledAndStopTakesTheAddressOff(t *testing.T) {
	p, s, requests := onEthernet(t)
	call := s.Windows.Call
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		reply, err := call(request)
		if request.Card != "" && request.Op == labnet.OpAdd {
			return labnet.Reply{}, errLost // the service added it; the answer never came
		}
		return reply, err
	}
	if err := s.EnsureForRun(eth, []string{"a"}); err != nil {
		t.Fatalf("the address is on the card, so the Run goes on: %v", err)
	}
	p.running = nil
	if err := s.Release(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*requests, []string{"add 192.168.40.21", "remove 192.168.40.21"}) {
		t.Fatalf("requests %v", *requests)
	}
	events := journalEvents(t, s)
	if !reflect.DeepEqual(events, []string{"requested", "applied", "removed"}) {
		t.Fatalf("journal %v", events)
	}
}

// The answer is lost and the address is not on the card yet (the service was still checking it):
// the Run stops with the reason, and when the address appears later Stop still takes it off.
func TestALostRealAnswerBeforeTheAddressAppears(t *testing.T) {
	p, s, requests := onEthernet(t)
	call, late := s.Windows.Call, false
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		if request.Card != "" && request.Op == labnet.OpAdd {
			*requests = append(*requests, "add (no answer)")
			late = true
			return labnet.Reply{}, errLost
		}
		return call(request)
	}
	err := s.EnsureForRun(eth, []string{"b"})
	if err == nil || !strings.HasPrefix(err.Error(), "D60 cannot start on 192.168.40.22: the lab network service did not answer.") {
		t.Fatalf("error %v", err)
	}
	if late { // the service finishes after the app gave up
		call(labnet.Request{Op: labnet.OpAdd, IPs: []string{"192.168.40.22"}, Card: eth})
	}
	p.running = nil
	s.Release()
	if (*requests)[len(*requests)-1] != "remove 192.168.40.22" {
		t.Fatalf("requests %v", *requests)
	}
}

// A lost lab-card answer is settled from the service's next answer: an address that went on is
// journaled as applied and the Run goes on; one that did not stops the Run with the reason.
func TestALostLabAnswerIsSettled(t *testing.T) {
	p := newPC()
	p.version = "2"
	s := p.service()
	s.Journal = newJournal(t)
	s.Use(nil)
	call := s.Windows.Call
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		reply, err := call(request)
		if request.Op == labnet.OpAdd {
			delete(p.cardIPs, "10.20.0.5") // only 192.168.216.1 really went on
			return labnet.Reply{}, errLost
		}
		return reply, err
	}
	err := s.EnsureForRun(card, []string{"a", "b"})
	if err == nil || strings.Contains(err.Error(), "P545") ||
		!strings.HasPrefix(err.Error(), "D60 cannot start on 10.20.0.5: the lab network service did not answer.") {
		t.Fatalf("error %v", err)
	}
	if events := journalEvents(t, s); !reflect.DeepEqual(events, []string{"applied", "failed"}) {
		t.Fatalf("journal %v", events)
	}
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		reply, err := call(request)
		if request.Op == labnet.OpRemove {
			return labnet.Reply{}, errLost // removed, answer lost
		}
		return reply, err
	}
	if err := s.Release(); err != nil || len(p.cardIPs) != 0 {
		t.Fatalf("stop: %v %v", err, p.cardIPs)
	}
	if events := journalEvents(t, s); events[len(events)-1] != "removed" {
		t.Fatalf("journal %v", events)
	}
}

func newJournal(t *testing.T) *netcfg.Journal { return netcfg.NewJournal(t.TempDir()) }

func journalEvents(t *testing.T, s *Service) []string {
	t.Helper()
	records, err := s.Journal.Records()
	if err != nil {
		t.Fatal(err)
	}
	events := []string{}
	for _, record := range records {
		event, _ := record.Get("event")
		events = append(events, event.(string))
	}
	return events
}
