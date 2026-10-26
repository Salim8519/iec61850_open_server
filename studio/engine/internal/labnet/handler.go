package labnet

import (
	"sort"
)

// Ops is what the service needs from Windows. The service's own registry key names the card;
// nothing a user can write decides which card is changed.
type Ops interface {
	// Card is the lab card's GUID ({...}) and whether it is a KM-TEST card that exists now.
	Card() (string, error)
	// Addresses lists the IPv4 addresses of every adapter by GUID.
	Addresses() (map[string][]string, error)
	// Names maps adapter GUIDs to their Windows names (for messages only).
	Names() map[string]string
	Add(card, ip string) error
	Remove(card, ip string) error
}

// Handle answers one request. Adding an address that is already on the card and removing one
// that is not are both fine (the app may retry after a reboot or a lost answer).
func Handle(ops Ops, request Request, version string) Reply {
	reply := Reply{Version: version, Addresses: []string{}}
	card, err := ops.Card()
	if err != nil {
		reply.Message = err.Error()
		return reply
	}
	reply.Card = card
	all, err := ops.Addresses()
	if err != nil {
		reply.Message = "Windows did not list the network addresses: " + err.Error()
		return reply
	}
	own := all[card]
	if request.Op == OpStatus {
		reply.OK = true
		return answered(reply, own)
	}
	ip, err := ParseAddress(request.IP)
	if err != nil {
		reply.Message = err.Error()
		return answered(reply, own)
	}
	present := contains(own, ip)
	switch {
	case request.Op == OpAdd && !present:
		if other := owner(all, card, ip); other != "" {
			name := ops.Names()[other]
			if name == "" {
				name = other
			}
			reply.Message = ip + " already belongs to “" + name + "” on this PC; nothing was changed"
			return answered(reply, own)
		}
		if err := ops.Add(card, ip); err != nil {
			reply.Message = "Windows did not add " + ip + ": " + err.Error()
			return answered(reply, own)
		}
		own = append(own, ip)
	case request.Op == OpRemove && present:
		if err := ops.Remove(card, ip); err != nil {
			reply.Message = "Windows did not remove " + ip + ": " + err.Error()
			return answered(reply, own)
		}
		own = without(own, ip)
	}
	reply.OK = true
	return answered(reply, own)
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
