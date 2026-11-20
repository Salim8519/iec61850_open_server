//go:build trial

package labsetup

// Trial harness for N1 and N2 (speed and network trials, 2026-11-20; change 0221). It drives the
// installed ScadaBench lab network service and reads Windows' own address list; only the project
// and the coordinator are stand-ins. Run it with studio/tools/trials/n1_n2_lab_failures.ps1.
// Addresses: 198.18.21.x (the service refuses loopback 127.x, so 127.67.21.x is the refused one).

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
)

type trialProject struct {
	ieds    []IED
	card    string
	running []string
}

func (p *trialProject) SimulatedIEDs() ([]IED, error)                   { return p.ieds, nil }
func (p *trialProject) Network() (string, string)                       { return "lab", p.card }
func (p *trialProject) UseCard(map[string]string, string, string) error { return nil }
func (p *trialProject) SetLabIP(string, string) error                   { return nil }

// Reserved keeps every address on the card that is not a trial address: other agents' Runs share
// the card, and Release takes off everything nobody keeps (the first run of this harness took
// 10.67.0.10-29 off and put them straight back).
func (p *trialProject) Reserved() map[string]string {
	kept := map[string]string{}
	found, _ := netcfg.AddressesOn(p.card)
	for address := range found {
		if !strings.HasPrefix(address, "198.18.21.") && !strings.HasPrefix(address, "127.67.21.") {
			kept[address] = "someone else"
		}
	}
	return kept
}
func (p *trialProject) InUse() map[string]string      { return p.Reserved() }
func (p *trialProject) Running() []string             { return p.running }
func (p *trialProject) Start([]string) error          { return nil }
func (p *trialProject) Stop([]string) error           { return nil }
func (p *trialProject) Stopped(fn func() error) error { return fn() }

func trialService(t *testing.T, addresses ...string) (*Service, *trialProject) {
	card := labnet.InstalledCard()
	if _, running := labnet.ServiceState(); card == "" || !running {
		t.Skip("the ScadaBench lab network service is not installed and running")
	}
	project := &trialProject{card: card}
	for index, address := range addresses {
		project.ieds = append(project.ieds, IED{ID: fmt.Sprintf("ied%02d", index), Name: fmt.Sprintf("TRIAL-%02d", index), LabIP: address})
	}
	journal := netcfg.NewJournal(t.TempDir())
	journal.Path = strings.TrimSuffix(journal.Path, "journal.jsonl") + "lab_journal.jsonl"
	shell := netcfg.SystemAdapters{Shell: netcfg.SystemExecutor{}}
	s := &Service{Projects: project, Runtime: project, Journal: journal, Asker: NewAsker(labnet.AskOn),
		Windows: Windows{Adapters: shell.Adapters, State: labnet.ServiceState, Card: labnet.InstalledCard,
			Call: labnet.Call, Ping: labnet.Ping, Approved: labnet.ReadApproved}}
	return s, project
}

func ids(project *trialProject) []string {
	list := []string{}
	for _, ied := range project.ieds {
		list = append(list, ied.ID)
	}
	return list
}

// onWindows is the trial addresses Windows lists on the lab card (the witness).
func onWindows(card string) []string {
	found, _ := netcfg.AddressesOn(card)
	list := []string{}
	for address := range found {
		if strings.HasPrefix(address, "198.18.21.") || strings.HasPrefix(address, "127.67.21.") {
			list = append(list, address)
		}
	}
	sort.Strings(list)
	return list
}

func journalLines(s *Service) []string {
	records, _ := s.Journal.Records()
	lines := []string{}
	for _, record := range records {
		raw, _ := record.Get("item")
		item := raw.(interface{ Get(string) (any, bool) })
		ip, _ := item.Get("ip")
		event, _ := record.Get("event")
		result, _ := record.Get("result")
		lines = append(lines, fmt.Sprintf("%v %v %v", event, ip, result))
	}
	return lines
}

func report(t *testing.T, name string, s *Service, project *trialProject, runErr, stopErr any) {
	t.Logf("[%s] run error: %v", name, runErr)
	t.Logf("[%s] stop error: %v", name, stopErr)
	t.Logf("[%s] on the card after Stop (Windows): %v", name, onWindows(project.card))
	for _, line := range journalLines(s) {
		t.Logf("[%s] journal: %s", name, line)
	}
}

// release calls Release whether it returns nothing (before change 0221) or an error (after).
func release(s *Service) any {
	var result any = nil
	func() { result = callRelease(s) }()
	return result
}

// N1: one address the service accepts and two it refuses (loopback): today Run says nothing.
func TestTrialN1RefusedAddresses(t *testing.T) {
	s, project := trialService(t, "198.18.21.11", "127.67.21.12", "127.67.21.13")
	runErr := s.EnsureForRun(project.card, ids(project))
	t.Logf("[N1 refused] on the card after Run (Windows): %v", onWindows(project.card))
	report(t, "N1 refused", s, project, runErr, release(s))
}

// N2: the service adds the address but its answer is lost (a client that gave up early).
func TestTrialN2LostAnswer(t *testing.T) {
	s, project := trialService(t, "198.18.21.21", "198.18.21.22")
	call := s.Windows.Call
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		reply, err := call(request)
		if request.Op != labnet.OpStatus && err == nil {
			return labnet.Reply{}, errors.New("the lab network service did not answer") // answer lost
		}
		return reply, err
	}
	runErr := s.EnsureForRun(project.card, ids(project))
	t.Logf("[N2 lost answer] on the card after Run (Windows): %v", onWindows(project.card))
	stopErr := release(s)
	s.Windows.Call = call
	report(t, "N2 lost answer", s, project, runErr, stopErr)
	t.Logf("[N2 lost answer] clean-up: %v; on the card: %v", release(s), onWindows(project.card))
}

// Happy path: Run's address step and Stop's release for 7 and 20 IEDs, timed.
func TestTrialHappyPathTiming(t *testing.T) {
	runs := 10
	if text := os.Getenv("TRIAL_RUNS"); text != "" {
		fmt.Sscan(text, &runs)
	}
	for _, size := range []int{7, 20} {
		addresses := []string{}
		for index := 0; index < size; index++ {
			addresses = append(addresses, fmt.Sprintf("198.18.21.%d", 100+index))
		}
		s, project := trialService(t, addresses...)
		var runTimes, stopTimes []time.Duration
		for run := 0; run < runs; run++ {
			start := time.Now()
			if err := s.EnsureForRun(project.card, ids(project)); err != nil {
				t.Fatalf("run %d: %v", run, err)
			}
			runTimes = append(runTimes, time.Since(start))
			if got := len(onWindows(project.card)); got != size {
				t.Fatalf("run %d: %d of %d addresses on the card", run, got, size)
			}
			start = time.Now()
			if err := release(s); err != nil {
				t.Fatalf("stop %d: %v", run, err)
			}
			stopTimes = append(stopTimes, time.Since(start))
			if left := onWindows(project.card); len(left) != 0 {
				t.Fatalf("stop %d left %v", run, left)
			}
			time.Sleep(300 * time.Millisecond)
		}
		t.Logf("[timing %d IEDs] Run address step median %v worst %v; Stop release median %v worst %v (%d runs)",
			size, median(runTimes), worst(runTimes), median(stopTimes), worst(stopTimes), runs)
		t.Logf("[raw %d] run %v stop %v", size, millis(runTimes), millis(stopTimes))
	}
}

func median(list []time.Duration) time.Duration {
	sorted := append([]time.Duration{}, list...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)/2].Round(100 * time.Microsecond)
}

func millis(list []time.Duration) []int64 {
	values := make([]int64, len(list))
	for index, item := range list {
		values[index] = item.Milliseconds()
	}
	return values
}

func worst(list []time.Duration) time.Duration {
	var found time.Duration
	for _, item := range list {
		found = max(found, item)
	}
	return found.Round(100 * time.Microsecond)
}
