package livedom

import (
	"sort"
	"strconv"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// ShortName is the last dotted part of a report block reference (short_name).
func ShortName(rcbRef string) string {
	return rcbRef[strings.LastIndex(rcbRef, ".")+1:]
}

// ChooseAccess decides how ScadaBench reads a real IED: free report blocks first, polling
// otherwise (choose_access). blockStatus holds the device answers for every SCD report
// block keyed by reference; access is the IED's live access mode ("auto_reports" default).
// It returns {"reports": [rcb refs], "polling_datasets": [...], "notes": [...]}.
func ChooseAccess(ied, blockStatus *pyjson.Object, access string) *pyjson.Object {
	notes, chosen, polling := []any{}, []any{}, []any{}
	byDataset := map[string][]any{}
	var datasets []string
	for _, report := range pyv.Objects(pyv.List(pyv.GetOr(ied, "reports", []any{}))) {
		dataset := pyv.Get(report, "dataset_ref")
		if !pyv.Truthy(dataset) {
			continue
		}
		key := pyv.PyStr(dataset)
		if _, found := byDataset[key]; !found {
			datasets = append(datasets, key)
		}
		byDataset[key] = append(byDataset[key], pyv.Get(report, "ref"))
	}
	sort.Strings(datasets)
	if access == "polling" {
		notes = append(notes, "Read-only polling is selected, so ScadaBench will not use any report block.")
		return pyv.O("reports", []any{}, "polling_datasets", pyv.Strings(datasets), "notes", notes)
	}
	for _, dataset := range datasets {
		blocks := byDataset[dataset]
		name := dataset[strings.LastIndex(dataset, "$")+1:]
		var free []any
		anyReadable := false
		for _, ref := range blocks {
			state := blockState(blockStatus, ref)
			readable := pyv.Truthy(pyv.Get(state, "readable"))
			anyReadable = anyReadable || readable
			if readable && pyv.Truthy(pyv.Get(state, "free")) {
				free = append(free, ref)
			}
		}
		if len(free) > 0 {
			chosen = append(chosen, free[0])
			notes = append(notes, "ScadaBench will use report block "+ShortName(pyv.PyStr(free[0]))+" for '"+name+
				"' because the IED shows it as free. It is given back when you press Stop.")
			continue
		}
		polling = append(polling, dataset)
		if !anyReadable {
			notes = append(notes, "The IED did not answer for the report blocks of '"+name+
				"', so ScadaBench will poll these values instead.")
		} else {
			notes = append(notes, "All "+strconv.Itoa(len(blocks))+" report block(s) for '"+name+
				"' are in use by other clients (for example SCADA or a gateway). ScadaBench will not take "+
				"them and will poll these values instead.")
		}
	}
	return pyv.O("reports", chosen, "polling_datasets", polling, "notes", notes)
}

// blockState is `block_status.get(ref) or {"readable": False}`. Device answers are objects;
// any other truthy answer has no readable/free keys here (Python would fail on .get).
func blockState(blockStatus *pyjson.Object, ref any) *pyjson.Object {
	text, ok := ref.(string)
	if !ok {
		return nil
	}
	return pyv.OrObj(blockStatus, text)
}
