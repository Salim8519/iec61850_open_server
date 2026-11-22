package netcfg

import "testing"

// Change 0233: the address the user set is planned before the SCD's, on any card.
func TestPlanProposesTheSavedAddress(t *testing.T) {
	adapter := Adapter{GUID: "g", Name: "LAN", Addresses: []Address{{IP: "172.26.0.5", Prefix: 24}}}
	ieds := []PlanIED{{ID: "a", Name: "A", SCLIP: "192.168.1.5", LabIP: "172.26.0.34", Port: 102},
		{ID: "b", Name: "B", SCLIP: "172.26.0.40", Port: 102}}
	for _, strategy := range []string{StrategySCD, StrategyAdapter} {
		rows, err := PlanAddresses(ieds, adapter, []Adapter{adapter}, strategy, nil)
		if err != nil {
			t.Fatal(err)
		}
		if rows[0].Proposed != "172.26.0.34" || !rows[0].Saved || rows[0].SCDIP != "192.168.1.5" {
			t.Fatalf("%s: saved address not proposed: %+v", strategy, rows[0])
		}
		if rows[1].Proposed != "172.26.0.40" || rows[1].Saved {
			t.Fatalf("%s: SCD address row changed: %+v", strategy, rows[1])
		}
		if saved, _ := rows[0].Object().Get("saved"); saved != true {
			t.Fatalf("%s: saved flag missing in JSON", strategy)
		}
	}
}
