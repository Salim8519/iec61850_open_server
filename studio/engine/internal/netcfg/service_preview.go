package netcfg

import (
	"studio-engine/internal/pyjson"
)

// pickRounds: re-picks when free addresses turn out to be in use on the network (change 0091).
const pickRounds = 5

// manualOverride is (manual or {}).get(ied_id) when truthy, else nil.
func manualOverride(manual any, iedID string) (*pyjson.Object, error) {
	if !truthy(manual) {
		return nil, nil
	}
	object, ok := manual.(*pyjson.Object)
	if !ok {
		return nil, &pyCrash{"manual addresses are not an object (Python: AttributeError)"}
	}
	value, _ := object.Get(iedID)
	if !truthy(value) {
		return nil, nil
	}
	override, ok := value.(*pyjson.Object)
	if !ok {
		return nil, &pyCrash{"a manual address is not an object (Python: AttributeError)"}
	}
	return override, nil
}

// rows is service._rows: the planner's rows with typed addresses applied.
func (s *NetworkService) rows(ieds []PlanIED, adapter Adapter, strategy string, manual any, avoid map[string]bool) ([]Row, error) {
	all, err := s.Adapters(false)
	if err != nil {
		return nil, err
	}
	rows, err := PlanAddresses(ieds, adapter, all, strategy, avoid)
	if err != nil {
		return nil, err
	}
	for index := range rows {
		override, err := manualOverride(manual, rows[index].IEDID)
		if err != nil {
			return nil, err
		}
		if override == nil {
			continue
		}
		ip, prefix, err := ValidateManual(get(override, "ip"), get(override, "prefix"))
		if err != nil {
			rows[index].Status, rows[index].Hint = "invalid_manual", err.Error()
		} else {
			rows[index].Proposed, rows[index].Prefix = ip, intPointer(prefix)
			rows[index].Status, rows[index].Hint = "ok", "Address typed by you."
		}
	}
	return rows, nil
}

type pickResult struct {
	rows      []Row
	reachable map[string]bool
	inUse     map[string]bool
}

// pick runs the planner up to five times, avoiding picked free addresses that answered.
func (s *NetworkService) pick(ieds []PlanIED, adapter Adapter, strategy string, manual any) (pickResult, error) {
	return s.pickAvoiding(ieds, adapter, strategy, manual, map[string]bool{})
}

// pickAvoiding is pick that never proposes the addresses in avoid (SNMP rows avoid the IEDs').
func (s *NetworkService) pickAvoiding(ieds []PlanIED, adapter Adapter, strategy string, manual any,
	avoid map[string]bool) (pickResult, error) {
	own := adapter.ips()
	var result pickResult
	for round := 0; round < pickRounds; round++ {
		rows, err := s.rows(ieds, adapter, strategy, manual, avoid)
		if err != nil {
			return result, err
		}
		reachable, reachableSet := []string{}, map[string]bool{}
		for _, row := range rows {
			if row.Status != "ok" || own[row.Proposed] {
				continue
			}
			linked, err := OnLink(row.Proposed, adapter)
			if err != nil {
				return result, err
			}
			if linked {
				reachable, reachableSet[row.Proposed] = append(reachable, row.Proposed), true
			}
		}
		inUse, err := s.Probe.InUse(reachable)
		if err != nil {
			return result, err
		}
		result = pickResult{rows: rows, reachable: reachableSet, inUse: inUse}
		clash := false
		for _, row := range rows {
			override, err := manualOverride(manual, row.IEDID)
			if err != nil {
				return result, err
			}
			if override == nil && inUse[row.Proposed] {
				clash = true
			}
		}
		if strategy != StrategyAdapter || !clash {
			break
		}
		for address := range inUse {
			avoid[address] = true
		}
	}
	return result, nil
}

// addAlternatives offers a free address on the adapter's own network for every row that
// cannot be used (change 0098).
func (s *NetworkService) addAlternatives(rows []Row, ieds []PlanIED, adapter Adapter, strategy string, inUse map[string]bool) error {
	allUsable := true
	keep := map[string]bool{}
	for address := range inUse {
		keep[address] = true
	}
	for _, row := range rows {
		if row.usable() {
			keep[row.Proposed] = true
		} else {
			allUsable = false
		}
	}
	if strategy != StrategySCD || allUsable {
		return nil
	}
	all, err := s.Adapters(false)
	if err != nil {
		return err
	}
	free, err := PlanAddresses(ieds, adapter, all, StrategyAdapter, keep)
	if err != nil {
		return err
	}
	byIED := map[string]Row{}
	for _, row := range free {
		byIED[row.IEDID] = row
	}
	for index, row := range rows {
		other, found := byIED[row.IEDID]
		if !row.usable() && found && other.Status == "ok" && other.Proposed != row.Proposed {
			rows[index].Alternative = &Alternative{IP: other.Proposed, Prefix: other.Prefix}
		}
	}
	return nil
}
