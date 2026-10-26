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
	cardIPs map[string]bool
	ieds    []IED
	mode    string
	guid    string
	running []string
	calls   []string
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
func (p *pc) Running() []string { return append([]string{}, p.running...) }
func (p *pc) Start(ids []string) error {
	p.calls = append(p.calls, "start "+strings.Join(ids, ","))
	p.running = append(p.running, ids...)
	return nil
}
func (p *pc) Stop(ids []string) error {
	p.calls = append(p.calls, "stop "+strings.Join(ids, ","))
	p.running = nil
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
			p.calls = append(p.calls, request.Op+" "+request.IP)
			switch request.Op {
			case labnet.OpAdd:
				p.cardIPs[request.IP] = true
			case labnet.OpRemove:
				delete(p.cardIPs, request.IP)
			}
			addresses := []string{}
			for address := range p.cardIPs {
				addresses = append(addresses, address)
			}
			sort.Strings(addresses)
			return labnet.Reply{OK: true, Card: card, Addresses: addresses}, nil
		},
	}
	return &Service{Projects: p, Runtime: p, Windows: windows, Asker: NewAsker(func(string, string) bool { return false })}
}

func TestUsePutsEveryIEDOnTheCardAndSwitchesTheProject(t *testing.T) {
	p := newPC()
	status, err := p.service().Use(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Active || p.mode != "lab" || p.guid != card || !p.cardIPs["192.168.216.1"] || !p.cardIPs["10.20.0.5"] {
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
	p.running, p.calls = []string{"b"}, nil
	status, err := service.SetAddress("b", "10.20.0.50")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"stop b", "add 10.20.0.50", "store 10.20.0.50", "remove 10.20.0.5", "start b"}
	if !reflect.DeepEqual(p.calls, want) {
		t.Fatalf("calls %v, want %v", p.calls, want)
	}
	if p.cardIPs["10.20.0.5"] || !p.cardIPs["10.20.0.50"] || !reflect.DeepEqual(p.running, []string{"b"}) {
		t.Fatalf("card %v running %v", p.cardIPs, p.running)
	}
	for _, row := range status.Rows {
		if row.IEDID == "b" && (row.IP != "10.20.0.50" || !row.OnCard || !row.Yours) {
			t.Fatalf("row %+v", row)
		}
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
	service.EnsureForRun(card)
	if !p.cardIPs["192.168.216.1"] || !p.cardIPs["10.20.0.5"] {
		t.Fatalf("card %v", p.cardIPs)
	}
	p.cardIPs = map[string]bool{}
	service.EnsureForRun("{OTHER}")
	if len(p.cardIPs) != 0 {
		t.Fatal("addresses were added for a project on another card")
	}
}
