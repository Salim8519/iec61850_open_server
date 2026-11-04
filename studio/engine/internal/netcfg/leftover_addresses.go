package netcfg

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Leftover addresses (ADR-0136): addresses ScadaBench's journal says it put on a card with the
// old helper path (`alias`) that are still there. The app asks about them after it opens and
// after Stop; firewall rules, approvals and the project's network mode are left alone (the full
// Clean up on the Network page still removes those).

// LeftoverAddresses lists them with their removal descriptions and a fingerprint.
func (s *NetworkService) LeftoverAddresses() (*pyjson.Object, error) {
	leftovers, err := s.Leftovers()
	if err != nil {
		return nil, err
	}
	removals := []*pyjson.Object{}
	for _, item := range leftovers {
		if get(item, "kind") == "alias" {
			removals = append(removals, copyObject(item).Set("action", "remove"))
		}
	}
	items, err := withDescriptions(removals, DescribeRemoval)
	if err != nil {
		return nil, err
	}
	digest, err := sortedFingerprint(itemsAny(items))
	if err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("items", itemsAny(items)).Set("count", len(items)).
		Set("fingerprint", digest), nil
}

// RemoveLeftoverAddresses removes the listed addresses after the user confirmed exactly that
// list: one Windows prompt for all of them.
func (s *NetworkService) RemoveLeftoverAddresses(fingerprint, confirm any) (*pyjson.Object, error) {
	if !isTrue(confirm) {
		return nil, apperr.New("Confirm the list of leftover addresses first")
	}
	listed, err := s.LeftoverAddresses()
	if err != nil {
		return nil, err
	}
	digest, _ := listed.Get("fingerprint")
	if fingerprint != digest {
		return nil, conflict("The leftover addresses are different now. Look at the list again.", nil)
	}
	raw, _ := listed.Get("items")
	results := []any{}
	if items := objects(raw); len(items) > 0 {
		if results, err = s.RunHelper("remove", items, digest.(string)); err != nil {
			return nil, err
		}
	}
	s.clearCache()
	return pyjson.NewObject().Set("results", results).
		Set("message", "ScadaBench removed the addresses it had left on your network."), nil
}
