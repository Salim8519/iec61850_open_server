package labsetup

import (
	"sort"
	"strings"

	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

// Real-card addresses ScadaBench added (fix plan A6, ADR-0132): Stop, launch and exit take an
// address off a real card only when the lab journal records that ScadaBench added it there and
// has not taken it off since. An approved address the user or DHCP gave the card is the PC's own
// and stays.

// realKey names one address on one real card.
func realKey(guid, address string) string { return strings.ToUpper(guid) + " " + address }

// outstandingReal is the real-card addresses the journal says ScadaBench added and has not taken
// off yet: the newest "requested", "applied", "removed" or "skipped" record of each wins; "failed"
// changes nothing. "requested" counts as added (change 0221): the service may have added the
// address although its answer was lost, and only an address really on the card, never-source and
// approved is ever taken off. An unreadable journal means none (nothing is removed).
func (s *Service) outstandingReal() map[string]bool { return s.outstanding("real_address") }

// outstanding is outstandingReal for any address kind ("real_address", "lab_address"), keyed by
// realKey(card, address).
func (s *Service) outstanding(kind string) map[string]bool {
	found := map[string]bool{}
	if s.Journal == nil {
		return found
	}
	records, err := s.Journal.Records()
	if err != nil {
		return found
	}
	for _, record := range records {
		raw, _ := record.Get("item")
		item, _ := raw.(*pyjson.Object)
		if item == nil || text(item, "kind") != kind {
			continue
		}
		key := realKey(text(item, "card"), text(item, "ip"))
		switch text(record, "event") {
		case "applied", "requested":
			found[key] = true
		case "removed", "skipped":
			delete(found, key)
		}
	}
	return found
}

// settleGone journals as settled the outstanding addresses no longer on this card: Windows
// forgets the service's addresses when it restarts, and an address the user gives the card
// afterwards must never be taken for ScadaBench's.
func (s *Service) settleGone(adapter netcfg.Adapter, outstanding map[string]bool) {
	present := map[string]bool{}
	for _, item := range adapter.Addresses {
		present[item.IP] = true
	}
	gone := []string{}
	for key := range outstanding {
		if address, found := strings.CutPrefix(key, realKey(adapter.GUID, "")); found && !present[address] {
			gone = append(gone, address)
		}
	}
	sort.Strings(gone)
	for _, address := range gone {
		delete(outstanding, realKey(adapter.GUID, address))
		s.journal("skipped", "real_address", address, "", adapter.GUID,
			"Not on the card (never added, Windows restarted or the card was reset); nothing to take off.")
	}
}

// addressesOn is the addresses the real card has now.
func (s *Service) addressesOn(guid string) map[string]bool {
	found := map[string]bool{}
	adapters, err := s.Windows.Adapters()
	if err != nil {
		return found
	}
	for _, adapter := range adapters {
		if strings.EqualFold(adapter.GUID, guid) {
			for _, item := range adapter.Addresses {
				found[item.IP] = true
			}
		}
	}
	return found
}

func text(object *pyjson.Object, key string) string {
	value, _ := object.Get(key)
	found, _ := value.(string)
	return found
}
