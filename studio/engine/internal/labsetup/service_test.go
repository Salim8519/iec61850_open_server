package labsetup

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
)

// pc is a fake Windows with a working card and service, and a project with two IEDs.
type pc struct {
	cardIPs  map[string]bool
	ieds     []IED
	mode     string
	guid     string
	running  []string
	calls    []string
	version  string // the service's protocol ("" answers like protocol 1)
	reserved map[string]string
	lab      *Service // Start and Stop call it as the coordinator does (ADR-0127)
}

func newPC() *pc {
	return &pc{cardIPs: map[string]bool{}, mode: "loopback",
		ieds: []IED{{ID: "a", Name: "P545", SCLIP: "192.168.216.1"}, {ID: "b", Name: "D60", SCLIP: "10.20.0.5"}}}
}

func (p *pc) SimulatedIEDs() ([]IED, error) { return append([]IED{}, p.ieds...), nil }
func (p *pc) Network() (string, string)     { return p.mode, p.guid }
func (p *pc) UseCard(addresses map[string]string, guid, _ string) error {
	for index := range p.ieds {
		p.ieds[index].LabIP = addresses[p.ieds[index].ID]
	}
	p.mode, p.guid = "lab", guid
	return nil
}
func (p *pc) SetLabIP(id, address string) error {
	for index := range p.ieds {
		if p.ieds[index].ID == id {
			p.ieds[index].LabIP = address
		}
	}
	p.calls = append(p.calls, "store "+address)
	return nil
}
func (p *pc) Reserved() map[string]string { return p.reserved }
func (p *pc) Running() []string           { return append([]string{}, p.running...) }
func (p *pc) Start(ids []string) error {
	p.calls = append(p.calls, "start "+strings.Join(ids, ","))
	p.lab.EnsureForRun(card, ids)
	p.running = append(p.running, ids...)
	p.lab.Release()
	return nil
}
func (p *pc) Stop(ids []string) error {
	p.calls = append(p.calls, "stop "+strings.Join(ids, ","))
	p.running = nil
	p.lab.Release()
	return nil
}

func (p *pc) service() *Service {
	windows := Windows{
		Adapters: func() ([]netcfg.Adapter, error) {
			adapters := pcAdapters()
			adapters[0].Addresses = nil
			for address := range p.cardIPs {
				adapters[0].Addresses = append(adapters[0].Addresses, netcfg.Address{IP: address, Prefix: 32})
			}
			return adapters, nil
		},
		State: func() (bool, bool) { return true, true },
		Card:  func() string { return card },
		Call: func(request labnet.Request) (labnet.Reply, error) {
			p.calls = append(p.calls, request.Op+" "+strings.Join(request.Wanted(), ","))
			for _, address := range request.Wanted() {
				switch request.Op {
				case labnet.OpAdd:
					p.cardIPs[address] = true
				case labnet.OpRemove:
					delete(p.cardIPs, address)
				}
			}
			addresses := []string{}
			for address := range p.cardIPs {
				addresses = append(addresses, address)
			}
			sort.Strings(addresses)
			return labnet.Reply{OK: true, Card: card, Addresses: addresses, Version: p.version}, nil
		},
	}
	p.lab = &Service{Projects: p, Runtime: p, Windows: windows, Asker: NewAsker(func(string, string) bool { return false })}
	return p.lab
}

func TestUseSwitchesTheProjectButPutsNothingOnTheCard(t *testing.T) {
	p := newPC()
	p.cardIPs["10.9.9.9"] = true // left by an earlier session
	status, err := p.service().Use(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Active || p.mode != "lab" || p.guid != card || len(p.cardIPs) != 0 {
		t.Fatalf("status %+v pc %+v", status, p)
	}
	if p.ieds[0].LabIP != "192.168.216.1" || p.ieds[1].LabIP != "10.20.0.5" {
		t.Fatalf("lab_ip not stored: %+v", p.ieds)
	}
}

func TestUseRefusesAClashAndChangesNothing(t *testing.T) {
	p := newPC()
	if _, err := p.service().Use(map[string]string{"b": "192.168.216.1"}); err == nil || !strings.Contains(err.Error(), "also uses") {
		t.Fatalf("duplicate accepted: %v", err)
	}
	if len(p.cardIPs) != 0 || p.mode != "loopback" {
		t.Fatalf("something changed: %+v", p)
	}
	p.running = []string{"a"}
	if _, err := p.service().Use(nil); err == nil {
		t.Fatal("Use while running was accepted")
	}
}

func TestARunningIEDMovesAndStartsAgain(t *testing.T) {
	p := newPC()
	service := p.service()
	if _, err := service.Use(nil); err != nil {
		t.Fatal(err)
	}
	service.Runtime.Start([]string{"b"})
	p.calls = nil
	status, err := service.SetAddress("b", "10.20.0.50")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"stop b", "status ", "remove 10.20.0.5", "store 10.20.0.50", "start b", "status ",
		"add 10.20.0.50", "status "}
	if !reflect.DeepEqual(p.calls, want) {
		t.Fatalf("calls %v, want %v", p.calls, want)
	}
	if !reflect.DeepEqual(p.cardIPs, map[string]bool{"10.20.0.50": true}) || !reflect.DeepEqual(p.running, []string{"b"}) {
		t.Fatalf("card %v running %v", p.cardIPs, p.running)
	}
	for _, row := range status.Rows {
		if row.IEDID == "b" && (row.IP != "10.20.0.50" || !row.OnCard || !row.Yours) {
			t.Fatalf("row %+v", row)
		}
	}
}

func TestAStoppedIEDMovesWithoutTouchingTheCard(t *testing.T) {
	p := newPC()
	service := p.service()
	service.Use(nil)
	p.calls = nil
	if _, err := service.SetAddress("b", "10.20.0.50"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.calls, []string{"store 10.20.0.50"}) || len(p.cardIPs) != 0 {
		t.Fatalf("calls %v card %v", p.calls, p.cardIPs)
	}
}

func TestAddressesFollowStartAndStop(t *testing.T) {
	p := newPC()
	p.reserved = map[string]string{"10.10.0.100": "Station gateway"}
	p.cardIPs["10.10.0.100"] = true
	service := p.service()
	service.Use(nil)
	service.Runtime.Start([]string{"a"})
	if !reflect.DeepEqual(p.cardIPs, map[string]bool{"192.168.216.1": true, "10.10.0.100": true}) {
		t.Fatalf("after Start a: %v", p.cardIPs)
	}
	service.Runtime.Stop(nil)
	if !reflect.DeepEqual(p.cardIPs, map[string]bool{"10.10.0.100": true}) {
		t.Fatalf("after Stop the gateway's address must stay alone: %v", p.cardIPs)
	}
	p.mode = "loopback" // This PC only: nothing of the project stays either
	p.cardIPs["192.168.216.1"] = true
	service.Release()
	if !reflect.DeepEqual(p.cardIPs, map[string]bool{"10.10.0.100": true}) {
		t.Fatalf("after This PC only: %v", p.cardIPs)
	}
}

func TestAMoveOntoAnotherIEDsAddressIsRefused(t *testing.T) {
	p := newPC()
	service := p.service()
	service.Use(nil)
	p.calls = nil
	if _, err := service.SetAddress("b", "192.168.216.1"); err == nil {
		t.Fatal("duplicate accepted")
	}
	if len(p.calls) != 0 {
		t.Fatalf("Windows was asked: %v", p.calls)
	}
}

func TestRunPutsTheAddressesBackAfterARestart(t *testing.T) {
	p := newPC()
	service := p.service()
	service.Use(nil)
	p.cardIPs = map[string]bool{} // Windows restarted
	service.EnsureForRun(card, []string{"a", "b"})
	if !p.cardIPs["192.168.216.1"] || !p.cardIPs["10.20.0.5"] {
		t.Fatalf("card %v", p.cardIPs)
	}
	p.cardIPs = map[string]bool{}
	service.EnsureForRun("{OTHER}", []string{"a", "b"})
	if len(p.cardIPs) != 0 {
		t.Fatal("addresses were added for a project on another card")
	}
}
