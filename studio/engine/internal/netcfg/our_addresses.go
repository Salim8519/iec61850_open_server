package netcfg

import (
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// OurAddresses lists the addresses ScadaBench added to a network card and has not removed yet,
// as the journal records them: address -> the GUIDs of the cards it was added to. It reads only the journal (no Windows
// call; a late helper result is journaled first), so the lab page may ask on every refresh
// (ADR-0131).
func (s *NetworkService) OurAddresses() (map[string][]string, error) {
	if err := s.catchUp(); err != nil {
		return nil, err
	}
	outstanding, err := s.Journal.Outstanding()
	if err != nil {
		return nil, err
	}
	ours := map[string][]string{}
	for _, item := range outstanding {
		if get(item, "kind") == "alias" {
			f := fieldsOf(item)
			ours[f.text("ip")] = append(ours[f.text("ip")], f.text("adapter_guid"))
		}
	}
	return ours, nil
}

// RemoveOurAddresses takes the given addresses off the cards ScadaBench put them on, through
// the network helper (one administrator prompt), and journals the result like Clean up. Only
// addresses the journal says ScadaBench added are touched; the rest of Clean up's list stays.
func (s *NetworkService) RemoveOurAddresses(addresses map[string]bool) error {
	outstanding, err := s.Journal.Outstanding()
	if err != nil {
		return err
	}
	adapters, err := s.Adapters(true)
	if err != nil {
		return err
	}
	present := &presence{service: s, adapters: adapters, read: map[string]bool{"adapters": true}}
	removals := []*pyjson.Object{}
	for _, item := range outstanding {
		if get(item, "kind") != "alias" || !addresses[fieldsOf(item).text("ip")] {
			continue
		}
		if there, err := present.present(item); err != nil || !there {
			continue // already gone (Wi-Fi set back to automatic removes them)
		}
		removals = append(removals, copyObject(item).Set("action", "remove"))
	}
	if len(removals) == 0 {
		return nil
	}
	items, err := withDescriptions(removals, DescribeRemoval)
	if err != nil {
		return err
	}
	digest, err := sortedFingerprint(itemsAny(items))
	if err != nil {
		return err
	}
	results, err := s.RunHelper("remove", items, digest)
	s.clearCache()
	if err != nil {
		return err
	}
	failed := []string{}
	for _, raw := range results {
		if line, ok := raw.(*pyjson.Object); ok && !truthy(get(line, "ok")) {
			failed = append(failed, pyStr(get(line, "description"))+": "+pyStr(get(line, "message")))
		}
	}
	if len(failed) > 0 {
		return apperr.New("Windows did not remove every old address: " + strings.Join(failed, "; "))
	}
	return nil
}
