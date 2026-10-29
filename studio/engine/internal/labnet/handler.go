package labnet

import (
	"sort"
)

// Ops is what the service needs from Windows. The service's own registry key names the card;
// nothing a user can write decides which card is changed.
type Ops interface {
	// Read lists Windows' adapters once (ADR-0128): the lab card, which must be a KM-TEST card
	// that exists now (ErrNoCard otherwise), and every adapter's IPv4 addresses and name.
	Read() (View, error)
	// Add and Remove change one address on the card that view names.
	Add(view View, ip string) error
	Remove(view View, ip string) error
}

// View is one reading of Windows' adapters, taken once per request.
type View struct {
	Card      string              // the lab card's GUID ({...})
	Index     uint32              // the card's interface index, for Add and Remove
	LUID      uint64              // the card's LUID, for Add and Remove
	Addresses map[string][]string // IPv4 addresses of every adapter by GUID
	Names     map[string]string   // Windows adapter names by GUID (for messages only)
}

// Handle answers one request. Adding an address that is already on the card and removing one
// that is not are both fine (the app may retry after a reboot or a lost answer). A request that
// names several addresses changes each with the same checks; Failed says which did not change.
func Handle(ops Ops, request Request, version string) Reply {
	reply := Reply{Version: version, Addresses: []string{}}
	view, err := ops.Read()
	if err != nil {
		reply.Message = err.Error()
		return reply
	}
	reply.Card = view.Card
	own := view.Addresses[view.Card]
	if request.Op == OpStatus {
		reply.OK = true
		return answered(reply, own)
	}
	failed := map[string]string{}
	for _, wanted := range request.Wanted() {
		var message string
		if own, message = change(ops, view, request.Op, wanted, own); message != "" {
			if reply.Message == "" {
				reply.Message = message
			}
			failed[wanted] = message
		}
	}
	reply.OK = len(failed) == 0
	if request.IPs != nil && len(failed) > 0 {
		reply.Failed = failed
	}
	return answered(reply, own)
}

// change adds or removes one address and returns the card's addresses after it, with the reason
// when it was refused or Windows failed.
func change(ops Ops, view View, op, wanted string, own []string) ([]string, string) {
	ip, err := ParseAddress(wanted)
	if err != nil {
		return own, err.Error()
	}
	present := contains(own, ip)
	switch {
	case op == OpAdd && !present:
		if other := owner(view.Addresses, view.Card, ip); other != "" {
			name := view.Names[other]
			if name == "" {
				name = other
			}
			return own, ip + " already belongs to “" + name + "” on this PC; nothing was changed"
		}
		if err := ops.Add(view, ip); err != nil {
			return own, "Windows did not add " + ip + ": " + err.Error()
		}
		return append(own, ip), ""
	case op == OpRemove && present:
		if err := ops.Remove(view, ip); err != nil {
			return own, "Windows did not remove " + ip + ": " + err.Error()
		}
		return without(own, ip), ""
	}
	return own, ""
}

func answered(reply Reply, addresses []string) Reply {
	sorted := append([]string{}, addresses...)
	sort.Strings(sorted)
	reply.Addresses = sorted
	return reply
}

// owner is the GUID of another adapter that already has ip, or "".
func owner(all map[string][]string, card, ip string) string {
	guids := make([]string, 0, len(all))
	for guid := range all {
		guids = append(guids, guid)
	}
	sort.Strings(guids)
	for _, guid := range guids {
		if guid != card && contains(all[guid], ip) {
			return guid
		}
	}
	return ""
}

func contains(list []string, wanted string) bool {
	for _, item := range list {
		if item == wanted {
			return true
		}
	}
	return false
}

func without(list []string, unwanted string) []string {
	kept := []string{}
	for _, item := range list {
		if item != unwanted {
			kept = append(kept, item)
		}
	}
	return kept
}
