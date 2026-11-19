package netcfg

import "studio-engine/internal/pyjson"

// firewallKinds are the journaled firewall changes: rules ScadaBench added and Windows' own rules
// it switched off ("unblock"). Addresses and approvals are not among them.
var firewallKinds = map[string]bool{"firewall": true, "gateway_firewall": true, "snmp_firewall": true,
	HostKeyKind: true, "unblock": true}

// FirewallCleanup is the uninstaller's clean-up (fix plan B-3), run inside the elevated lab network
// program: the journaled firewall changes that still exist are undone with the same commands as
// Clean up, and each result is journaled. Only items the journal records as ScadaBench's own and
// the helper's validation accepts are touched. It returns how many were undone and the first
// reason one was not ("" when all were).
func (s *NetworkService) FirewallCleanup(executor Executor) (int, string, error) {
	if err := s.catchUp(); err != nil {
		return 0, "", err
	}
	outstanding, err := s.Journal.Outstanding()
	if err != nil {
		return 0, "", err
	}
	check := &presence{service: s, read: map[string]bool{}, unblock: []string{}}
	for _, item := range outstanding {
		if get(item, "kind") == "unblock" {
			check.unblock = append(check.unblock, fieldsOf(item).text("name"))
		}
	}
	items, problem := []*pyjson.Object{}, ""
	for _, item := range outstanding {
		if !firewallKinds[pyStr(get(item, "kind"))] {
			continue
		}
		present, err := check.present(item)
		if err != nil {
			return 0, "", err
		}
		if !present {
			continue
		}
		if err := validateItem(item, s.Program); err != nil {
			if problem == "" {
				problem = err.Error() + "; that change was left as it is"
			}
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return 0, problem, nil
	}
	return s.undoFirewall(items, problem, executor)
}

// undoFirewall removes the checked items in one PowerShell batch and journals each result.
func (s *NetworkService) undoFirewall(items []*pyjson.Object, problem string, executor Executor) (int, string, error) {
	digest, err := sortedFingerprint(itemsAny(items))
	if err != nil {
		return 0, "", err
	}
	request := pyjson.NewObject().Set("action", "remove").Set("fingerprint", digest).Set("items", itemsAny(items))
	results, err := ExecuteBatch(request, executor)
	if err != nil {
		return 0, "", err
	}
	byKey, undone := map[string]*pyjson.Object{}, 0
	for _, raw := range results {
		entry, _ := raw.(*pyjson.Object)
		if entry == nil {
			continue
		}
		byKey[pyStr(get(entry, "key"))] = entry
		if truthy(get(entry, "ok")) {
			undone++
		} else if problem == "" {
			if problem = pyStr(get(entry, "message")); problem == "" {
				problem = "Windows did not undo one of the changes"
			}
		}
	}
	if _, err := s.journalResults("remove", items, digest, byKey); err != nil {
		return undone, problem, err
	}
	return undone, problem, nil
}
