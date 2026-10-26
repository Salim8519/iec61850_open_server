// Package labsetup runs the "This PC lab network" page (ADR-0120): which address each simulated
// IED answers at on the ScadaBench Lab card, the clash checks against this PC's real cards and
// their networks, creating and removing the card through the one administrator prompt, and
// moving an IED to another address at any time through the lab network service. Every decision
// is made here; the browser only shows the rows.
package labsetup

import (
	"sort"

	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
)

// IED is a simulated IED as the page needs it.
type IED struct {
	ID, Name, SCLIP, LabIP string
}

// Row is one IED on the page. Status is one of ok, same_range, real_device, checking, duplicate,
// other_card, invalid and missing; the last four block "Use" and "Change".
type Row struct {
	IEDID       string `json:"ied_id"`
	Name        string `json:"name"`
	SCLIP       string `json:"scl_ip"`
	IP          string `json:"ip"`
	Yours       bool   `json:"yours"`
	OnCard      bool   `json:"on_card"`
	Status      string `json:"status"`
	Hint        string `json:"hint"`
	Card        string `json:"card,omitempty"`
	Alternative string `json:"alternative,omitempty"`
}

// Blocking reports whether the row's address cannot be used.
func (r Row) Blocking() bool {
	switch r.Status {
	case "duplicate", "other_card", "invalid", "missing":
		return true
	}
	return false
}

// Real is a real card of this PC (Ethernet or Wi-Fi) whose network the lab addresses may overlap.
type Real struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Connected bool     `json:"connected"`
	Networks  []string `json:"networks"`
	Source    string   `json:"-"` // its own first address, the source of ARP questions
}

// Answers knows which (address, source) pairs a device answered on; pending ones are unknown.
type Answers interface {
	// Answer is (answered, known).
	Answer(address, source string) (bool, bool)
}

// RealCards are this PC's Ethernet and Wi-Fi cards with their IPv4 networks.
func RealCards(adapters []netcfg.Adapter, card string) []Real {
	reals := []Real{}
	for _, adapter := range adapters {
		if adapter.GUID == card || (adapter.Kind != "ethernet" && adapter.Kind != "wifi") {
			continue
		}
		real := Real{Name: adapter.Name, Kind: adapter.Kind, Connected: adapter.Status == "Up", Networks: []string{}}
		for _, address := range adapter.Addresses {
			if labnet.InNetwork(address.IP, "169.254.0.0", 16) {
				continue
			}
			real.Networks = append(real.Networks, address.IP+"/"+itoa(address.Prefix))
			if real.Source == "" {
				real.Source = address.IP
			}
		}
		reals = append(reals, real)
	}
	return reals
}

// Rows decides every IED's address and status. chosen holds addresses typed on the page (by IED
// id); active says the project already uses the card, so its stored lab_ip is the address.
func Rows(ieds []IED, chosen map[string]string, active bool, adapters []netcfg.Adapter, card string,
	answers Answers) []Row {
	elsewhere, onCard := map[string]string{}, map[string]bool{}
	for _, adapter := range adapters {
		for _, address := range adapter.Addresses {
			if adapter.GUID == card {
				onCard[address.IP] = true
			} else {
				elsewhere[address.IP] = adapter.Name
			}
		}
	}
	reals := RealCards(adapters, card)
	rows := make([]Row, len(ieds))
	users := map[string][]int{}
	for index, ied := range ieds {
		row := Row{IEDID: ied.ID, Name: ied.Name, SCLIP: ied.SCLIP, IP: ied.SCLIP}
		if active && ied.LabIP != "" {
			row.IP = ied.LabIP
		}
		if typed, found := chosen[ied.ID]; found {
			row.IP = typed
		}
		row.Yours = row.IP != ied.SCLIP
		rows[index] = row
		if parsed, err := labnet.ParseAddress(row.IP); err == nil {
			rows[index].IP = parsed
			users[parsed] = append(users[parsed], index)
		}
	}
	for index := range rows {
		judge(&rows[index], rows, users, elsewhere, reals, answers)
		rows[index].OnCard = onCard[rows[index].IP]
	}
	return rows
}

func judge(row *Row, rows []Row, users map[string][]int, elsewhere map[string]string, reals []Real, answers Answers) {
	if row.IP == "" {
		row.Status, row.Hint = "missing", "This IED has no address in its file. Type the address it should answer at."
		return
	}
	if _, err := labnet.ParseAddress(row.IP); err != nil {
		row.Status, row.Hint = "invalid", "Not an address ScadaBench can use: "+err.Error()+"."
		return
	}
	if others := users[row.IP]; len(others) > 1 {
		for _, other := range others {
			if rows[other].IEDID != row.IEDID {
				row.Status = "duplicate"
				row.Hint = rows[other].Name + " also uses " + row.IP + ". Each IED needs its own address."
				return
			}
		}
	}
	if name, found := elsewhere[row.IP]; found {
		row.Status = "other_card"
		row.Hint = row.IP + " is the address of “" + name + "” on this PC. Pick another address."
		return
	}
	row.Status, row.Hint = "ok", "Only on this PC. Ping, MMS and GOOSE work here like on a real network."
	for _, real := range reals {
		network := within(row.IP, real.Networks)
		if network == "" {
			continue
		}
		row.Card = real.Name
		if !real.Connected || real.Source == "" {
			row.Status = "same_range"
			row.Hint = "In the same range as " + real.Name + " (" + network + "). That's fine: it stays on this PC."
			continue
		}
		answered, known := answers.Answer(row.IP, real.Source)
		switch {
		case !known:
			row.Status, row.Hint = "checking", "Asking "+real.Name+" whether a real device uses "+row.IP+"…"
		case answered:
			row.Status = "real_device"
			row.Hint = "A real device on " + real.Name + " uses " + row.IP + ". While ScadaBench uses it, this PC " +
				"reaches the simulator, not that device."
			row.Alternative = free(row.IP, users, elsewhere, real.Source, answers)
		default:
			row.Status = "same_range"
			row.Hint = "In the same range as " + real.Name + " (" + network + "); no real device answered there. " +
				"It stays on this PC."
		}
		return
	}
}

// free is the first address of the same /24 that no IED and no card uses and that no device
// answered on (only addresses already asked about count as free).
func free(address string, users map[string][]int, elsewhere map[string]string, source string, answers Answers) string {
	for _, candidate := range labnet.SameSlash24(address) {
		if len(users[candidate]) > 0 || elsewhere[candidate] != "" {
			continue
		}
		if answered, known := answers.Answer(candidate, source); known && !answered {
			return candidate
		}
	}
	return ""
}

func within(address string, networks []string) string {
	for _, network := range networks {
		ip, prefix := split(network)
		if labnet.InNetwork(address, ip, prefix) {
			return network
		}
	}
	return ""
}

// Questions are the (address, source) pairs Rows needs answered: each row in a connected real
// card's network, plus the first free candidates of rows whose own address is taken.
func Questions(rows []Row, reals []Real, answers Answers) [][2]string {
	asked := map[[2]string]bool{}
	for _, row := range rows {
		for _, real := range reals {
			if !real.Connected || real.Source == "" || within(row.IP, real.Networks) == "" {
				continue
			}
			asked[[2]string{row.IP, real.Source}] = true
			if answered, known := answers.Answer(row.IP, real.Source); known && answered {
				for _, candidate := range labnet.SameSlash24(row.IP)[:4] {
					asked[[2]string{candidate, real.Source}] = true
				}
			}
		}
	}
	pairs := make([][2]string, 0, len(asked))
	for pair := range asked {
		pairs = append(pairs, pair)
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i][0]+pairs[i][1] < pairs[j][0]+pairs[j][1] })
	return pairs
}
