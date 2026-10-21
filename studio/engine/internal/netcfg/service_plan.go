package netcfg

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

func (s *NetworkService) rowHints(picked pickResult) {
	for index, row := range picked.rows {
		if picked.inUse[row.Proposed] {
			picked.rows[index].Status = "in_use_on_network"
			picked.rows[index].Hint = "Another device already answers on " + row.Proposed + ". " +
				"Pick a different address so the two do not clash."
		} else if row.Status == "ok" && row.Hint == "" {
			picked.rows[index].Hint = "Not used on this PC. Devices in this new network can only be checked " +
				"after the address is added; Windows reports a clash then."
			if picked.reachable[row.Proposed] {
				picked.rows[index].Hint = "No other device answered on this address."
			}
		}
	}
}

// Preview is service.preview: the address rows, the exact Windows changes and their
// fingerprint. The preview is remembered so Apply can re-check it.
func (s *NetworkService) Preview(guid any, strategy string, manual any) (*pyjson.Object, error) {
	result, _, err := s.preview(guid, strategy, manual)
	return result, err
}

func (s *NetworkService) preview(guid any, strategy string, manual any) (*pyjson.Object, previewState, error) {
	s.Progress.begin()
	defer s.Progress.finish()
	s.Progress.step("cards")
	adapter, err := s.Adapter(guid)
	if err != nil {
		return nil, previewState{}, err
	}
	ieds, err := s.Projects.SimulatedIEDs()
	if err != nil {
		return nil, previewState{}, err
	}
	devices, err := s.snmpDevices()
	if err != nil {
		return nil, previewState{}, err
	}
	if len(ieds) == 0 && len(devices) == 0 {
		return nil, previewState{}, apperr.New("No IED is set to 'Simulate', so there is nothing to put on " +
			"the network. Set an IED to 'Simulate' in Devices first.")
	}
	s.Progress.step("addresses")
	picked, err := s.pick(ieds, adapter, strategy, manual)
	if err != nil {
		return nil, previewState{}, err
	}
	s.rowHints(picked)
	rows := picked.rows
	if err := s.addAlternatives(rows, ieds, adapter, strategy, picked.inUse); err != nil {
		return nil, previewState{}, err
	}
	snmp := SNMPRule{Program: snmpWorker(s.Program)}
	if len(devices) > 0 {
		snmpRows, err := s.snmpRows(devices, adapter, manual, rows)
		if err != nil {
			return nil, previewState{}, err
		}
		rows = append(rows, snmpRows...)
		for _, device := range devices {
			snmp.Ports = append(snmp.Ports, device.Port)
		}
	}
	s.Progress.step("firewall")
	if len(devices) > 0 {
		if snmp.Existing, err = s.SNMPRules(); err != nil {
			return nil, previewState{}, err
		}
	}
	ports := make([]int, len(ieds))
	for index, ied := range ieds {
		ports[index] = ied.Port
	}
	existing, err := s.Rules()
	if err != nil {
		return nil, previewState{}, err
	}
	s.Progress.step("plan")
	plan, err := BuildPlanWithSNMP(adapter, rows, ports, s.Program, existing, snmp)
	if err != nil {
		return nil, previewState{}, err
	}
	items, err := withDescriptions(plan.Items, Describe)
	if err != nil {
		return nil, previewState{}, err
	}
	state := previewState{adapter: adapter, rows: rows, items: items, fingerprint: plan.Fingerprint, ready: true}
	for _, row := range rows {
		state.ready = state.ready && row.usable()
	}
	needsAdmin := false
	for _, item := range plan.Items {
		needsAdmin = needsAdmin || get(item, "action") == "add"
	}
	if !truthy(manual) {
		manual = pyjson.NewObject()
	}
	s.mu.Lock()
	if s.previews == nil {
		s.previews = map[string]storedPreview{}
	}
	s.previews[plan.Fingerprint] = storedPreview{guid: adapter.GUID, strategy: strategy, manual: manual}
	s.mu.Unlock()
	result := pyjson.NewObject().Set("adapter", adapter.Object()).Set("rows", RowsObject(rows)).
		Set("items", itemsAny(items)).Set("fingerprint", plan.Fingerprint).Set("ready", state.ready).
		Set("needs_admin", needsAdmin)
	return result, state, nil
}

// previewState is the typed part of a preview Apply needs.
type previewState struct {
	adapter     Adapter
	rows        []Row
	items       []*pyjson.Object
	fingerprint string
	ready       bool
}
