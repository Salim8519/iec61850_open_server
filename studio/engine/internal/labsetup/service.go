package labsetup

import (
	"strings"
	"sync"

	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
)

// Projects is what the page needs from the open project.
type Projects interface {
	SimulatedIEDs() ([]IED, error)
	// Network is the stored mode and adapter GUID.
	Network() (mode, guid string)
	// UseCard stores lab_ip per IED and switches the network to lab mode on the card.
	UseCard(addresses map[string]string, guid, label string) error
	// SetLabIP stores one IED's lab_ip.
	SetLabIP(id, ip string) error
	// Reserved maps the card addresses other devices of this PC use (the station gateway, ADR-0125)
	// to their names; no IED may take them and Use keeps them on the card.
	Reserved() map[string]string
}

// Runtime is the coordinator: which IEDs run, and starting or stopping some of them.
type Runtime interface {
	Running() []string
	Start(ids []string) error
	Stop(ids []string) error
}

// Windows is everything that reads or changes Windows; tests replace it.
type Windows struct {
	Adapters func() ([]netcfg.Adapter, error) // fresh list (netcfg names, kinds and states)
	State    func() (installed, running bool) // the lab network service
	Card     func() string                    // the card GUID the installer stored
	Call     func(labnet.Request) (labnet.Reply, error)
	Ping     func(address string, timeoutMS uint32) (labnet.Pinged, error)
}

// Service is the lab network page's logic.
type Service struct {
	Projects  Projects
	Runtime   Runtime
	Windows   Windows
	Asker     *Asker
	Elevator  netcfg.Elevator
	Program   string // scadabench_labnet.exe beside the native workers
	Directory string // the data folder
	Journal   *netcfg.Journal
	NewID     func() string

	change sync.Mutex // one address change at a time
	card   sync.Mutex // Run's adds and Release's removals never overlap
}

// Status is GET /network/lab.
type Status struct {
	Installed   bool     `json:"installed"`    // the service exists
	Running     bool     `json:"running"`      // the service answers
	Card        string   `json:"card"`         // the card GUID ("" before Create)
	CardPresent bool     `json:"card_present"` // Windows lists that card now
	CardName    string   `json:"card_name"`
	Ready       bool     `json:"ready"`  // card and service work: addresses can be added
	Active      bool     `json:"active"` // the project uses the card (lab mode on it)
	IEDsRunning bool     `json:"ieds_running"`
	Reals       []Real   `json:"real_cards"`
	Rows        []Row    `json:"rows"`
	Checking    bool     `json:"checking"`
	Problem     string   `json:"problem,omitempty"`
	CardIPs     []string `json:"card_addresses"`
}

// Status reads Windows and the project and decides every row; ARP answers that are missing are
// asked in the background (the page refreshes by itself).
func (s *Service) Status(chosen map[string]string) (Status, error) {
	status := Status{CardName: labnet.CardName, Reals: []Real{}, Rows: []Row{}, CardIPs: []string{}}
	status.Installed, status.Running = s.Windows.State()
	status.Card = s.Windows.Card()
	adapters, err := s.Windows.Adapters()
	if err != nil {
		return status, err
	}
	for _, adapter := range adapters {
		if status.Card != "" && strings.EqualFold(adapter.GUID, status.Card) {
			status.Card, status.CardPresent = adapter.GUID, true
			for _, address := range adapter.Addresses {
				if _, err := labnet.ParseAddress(address.IP); err == nil {
					status.CardIPs = append(status.CardIPs, address.IP)
				}
			}
		}
	}
	status.Ready = status.Running && status.CardPresent
	mode, guid := s.Projects.Network()
	status.Active = mode == "lab" && status.Card != "" && strings.EqualFold(guid, status.Card)
	status.IEDsRunning = len(s.Runtime.Running()) > 0
	switch {
	case status.Installed && !status.CardPresent:
		status.Problem = "The ScadaBench Lab card is missing (removed in Device Manager?). Press Repair."
	case status.Installed && !status.Running:
		status.Problem = "The lab network service is not running. Press Repair."
	}
	ieds, err := s.Projects.SimulatedIEDs()
	if err != nil {
		return status, err
	}
	card := ""
	if status.CardPresent {
		card = status.Card
	}
	status.Reals = RealCards(adapters, card)
	status.Rows = Rows(ieds, chosen, status.Active, adapters, card, s.Asker)
	s.Asker.Start(Questions(status.Rows, status.Reals, s.Asker))
	reserve(status.Rows, s.Projects.Reserved())
	for _, row := range status.Rows {
		status.Checking = status.Checking || row.Status == "checking"
	}
	return status, nil
}

// settled is Status after every ARP question it needs has been answered.
func (s *Service) settled(chosen map[string]string) (Status, error) {
	status, err := s.Status(chosen)
	for round := 0; err == nil && status.Checking && round < 3; round++ {
		s.Asker.Wait(Questions(status.Rows, status.Reals, s.Asker))
		status, err = s.Status(chosen)
	}
	return status, err
}

// Ping is POST /network/lab/ping: one echo, as Windows ping does.
func (s *Service) Ping(address string) (labnet.Pinged, error) {
	parsed, err := labnet.ParseAddress(address)
	if err != nil {
		return labnet.Pinged{}, plain(err.Error())
	}
	return s.Windows.Ping(parsed, 1000)
}
