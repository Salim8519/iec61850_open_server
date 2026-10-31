package labsetup

import (
	"strings"

	"studio-engine/internal/netcfg"
)

// StatusOurs is a row whose address ScadaBench itself put on another card of this PC (an earlier
// "Put your published IEDs on a network" on Ethernet or Wi-Fi). It does not block: Use takes the
// address off that card first (ADR-0131).
const StatusOurs = "ours_elsewhere"

// markOurs turns "other_card" rows into StatusOurs when the journal says ScadaBench added that
// address to the card that has it. ours maps address -> card GUIDs. An address the user or
// another program set keeps blocking.
func markOurs(rows []Row, adapters []netcfg.Adapter, ours map[string][]string) {
	for index, row := range rows {
		guids, found := ours[row.IP]
		if row.Status != "other_card" || !found {
			continue
		}
		for _, adapter := range adapters {
			if !addedTo(guids, adapter.GUID) || !hasAddress(adapter, row.IP) {
				continue
			}
			rows[index].Status = StatusOurs
			rows[index].Hint = "ScadaBench put " + row.IP + " on “" + adapter.Name + "” earlier. Use takes it off " +
				adapter.Name + " (one Windows prompt) and puts it on the lab card."
		}
	}
}

func addedTo(guids []string, guid string) bool {
	for _, item := range guids {
		if strings.EqualFold(item, guid) {
			return true
		}
	}
	return false
}

func hasAddress(adapter netcfg.Adapter, address string) bool {
	for _, item := range adapter.Addresses {
		if item.IP == address {
			return true
		}
	}
	return false
}

// freeOurs takes ScadaBench's own addresses off the other cards before Use switches to the lab
// card, so the IEDs answer only on the card.
func (s *Service) freeOurs(rows []Row) error {
	addresses := map[string]bool{}
	for _, row := range rows {
		if row.Status == StatusOurs {
			addresses[row.IP] = true
		}
	}
	if len(addresses) == 0 || s.Windows.FreeOurs == nil {
		return nil
	}
	return s.Windows.FreeOurs(addresses)
}
