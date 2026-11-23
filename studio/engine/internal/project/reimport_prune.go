package project

import (
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// pruner keeps the links of one re-imported IED that its new model still has (ADR-0148). A link
// stays when its reference is in the new file with the same type; a report no longer holding the
// signal is cleared (the link stays). Anything else is removed and recorded in words.
type pruner struct {
	target, name string
	old, fresh   *pyjson.Object
	ieds         *Lookup
	lost         []ReimportLost
	kept         int
}

func (p *pruner) mine(binding *pyjson.Object) bool {
	return binding != nil && text(binding, "ied_id") == p.target
}

func (p *pruner) record(where, ref, why string) {
	p.lost = append(p.lost, ReimportLost{Where: where, Ref: ref, Why: why})
}

// signalWhy is "" when a signal binding can stay, else why not.
func (p *pruner) signalWhy(binding *pyjson.Object) string {
	ref := text(binding, "ref")
	now := p.ieds.Entry(pyval.Lookup(p.fresh, "signals", nil), ref)
	if now == nil {
		return "not in the new file"
	}
	if before := p.ieds.Entry(pyval.Lookup(p.old, "signals", nil), ref); before != nil && text(before, "kind") != text(now, "kind") {
		return "its type changed from " + text(before, "kind") + " to " + text(now, "kind")
	}
	if report := pyval.Lookup(binding, "report_ref", nil); pyval.Truthy(report) && validateMapping(p.fresh, ref, report) != nil {
		binding.Set("report_ref", nil)
	}
	return ""
}

// controlWhy is "" when a command binding can stay; typed allows INC, ENC, APC, BAC, BSC and ISC.
func (p *pruner) controlWhy(binding *pyjson.Object, typed bool) string {
	ref := text(binding, "ref")
	now, before := p.control(p.fresh, ref, typed), p.control(p.old, ref, typed)
	if now == nil {
		return "not in the new file"
	}
	if before != nil && text(before, "cdc") != text(now, "cdc") {
		return "its class changed from " + text(before, "cdc") + " to " + text(now, "cdc")
	}
	return ""
}

func (p *pruner) control(ied *pyjson.Object, ref string, typed bool) *pyjson.Object {
	if found := p.ieds.Entry(pyval.Lookup(ied, "controls", nil), ref); found != nil || !typed {
		return found
	}
	return findByRef(extendedControlsOf(ied), ref)
}

// hasRef says whether the new model has the reference as a signal or a command.
func (p *pruner) hasRef(ref string) bool {
	return p.ieds.Entry(pyval.Lookup(p.fresh, "signals", nil), ref) != nil || p.control(p.fresh, ref, true) != nil
}

// keep checks one binding: true when it stays (counted), false when it was recorded as lost.
func (p *pruner) keep(binding *pyjson.Object, where string, command, typed bool) bool {
	if !p.mine(binding) {
		return true
	}
	why := ""
	if command {
		why = p.controlWhy(binding, typed)
	} else {
		why = p.signalWhy(binding)
	}
	if why != "" {
		p.record(where, text(binding, "ref"), why)
		return false
	}
	p.kept++
	return true
}

// diagram prunes every equipment that uses the IED, then checks each one alone; equipment that
// still does not pass loses all its links to this IED (the equipment itself stays).
func (p *pruner) diagram(diagram *pyjson.Object) {
	ids := map[string]bool{p.target: true}
	for _, item := range items(pyval.Lookup(diagram, "nodes", nil)) {
		node := object(item)
		data := object(pyval.Lookup(node, "data", nil))
		if data == nil || !usesIED(data, ids) {
			continue
		}
		label := text(data, "label")
		if label == "" {
			label = text(node, "id")
		}
		p.node(data, "SLD "+label)
		alone := pyjson.NewObject()
		for _, key := range diagram.Keys() {
			value, _ := diagram.Get(key)
			alone.Set(key, value)
		}
		alone.Set("nodes", []any{node}).Set("edges", []any{})
		if _, err := ValidateDiagramWith(alone, p.ieds); err != nil {
			dropNodeLinks(data, ids)
			p.record("SLD "+label, "all links to "+p.name, err.Error())
		}
	}
}

func (p *pruner) node(data *pyjson.Object, where string) {
	if mapping := object(pyval.Lookup(data, "state_mapping", nil)); mapping != nil {
		for _, key := range []string{"source", "open", "closed"} {
			if binding := object(pyval.Lookup(mapping, key, nil)); !p.keep(binding, where+" position", false, false) {
				data.Set("state_mapping", nil)
				break
			}
		}
	}
	if binding := object(pyval.Lookup(data, "control_binding", nil)); !p.keep(binding, where+" command", true, false) {
		data.Set("control_binding", nil)
	}
	filter := func(key, part string, pick func(entry *pyjson.Object) (*pyjson.Object, bool, bool)) {
		value, present := data.Get(key)
		if !present || value == nil {
			return
		}
		kept := []any{}
		for _, item := range items(value) {
			binding, command, typed := pick(object(item))
			if p.keep(binding, where+" "+part, command, typed) {
				kept = append(kept, item)
			}
		}
		data.Set(key, kept)
	}
	source := func(entry *pyjson.Object) (*pyjson.Object, bool, bool) {
		return object(pyval.Lookup(entry, "source", nil)), false, false
	}
	filter("relay_statuses", "status", source)
	filter("measurements", "measurement", source)
	if kindOf(data) == PointPanelKind && text(data, "ied_id") == p.target {
		filter("points", "point", func(entry *pyjson.Object) (*pyjson.Object, bool, bool) {
			if control := object(pyval.Lookup(entry, "control", nil)); control != nil {
				return control, true, true
			}
			return object(pyval.Lookup(entry, "source", nil)), false, false
		})
	}
	if tap := object(pyval.Lookup(data, "tap_changer", nil)); tap != nil {
		for _, key := range append(append([]string{}, tapSignals...), tapControls...) {
			command := pyval.InTuple(key, tapControls...)
			if binding := object(pyval.Lookup(tap, key, nil)); !p.keep(binding, where+" tap changer", command, true) {
				tap.Set(key, nil)
			}
		}
		settleTapCommand(tap)
	}
	if link := object(pyval.Lookup(data, "goose_link", nil)); p.mine(link) {
		if err := checkGooseLink(data, kindOf(data), p.ieds); err != nil {
			data.Delete("goose_link")
			p.record(where+" GOOSE", text(link, "gocb_ref"), err.Error())
		} else {
			p.kept++
		}
	}
}

// list keeps the entries of another IED and the entries of this IED that check alone.
func (p *pruner) list(value any, where string, check func(entry any) error) []any {
	kept := []any{}
	for _, item := range items(value) {
		if p.mine(object(item)) {
			if err := check(item); err != nil {
				p.record(where, text(object(item), "ref"), err.Error())
				continue
			}
			p.kept++
		}
		kept = append(kept, item)
	}
	return kept
}

// scenarios drops the steps that no longer check; a scenario left without steps goes too.
func (p *pruner) scenarios(value any, ieds []*pyjson.Object) []any {
	kept := []any{}
	for _, item := range items(value) {
		scenario := object(item)
		where := "Scenario " + text(scenario, "name")
		steps := p.list(pyval.Lookup(scenario, "steps", nil), where, func(step any) error {
			alone := pyval.CopyObject(scenario).Set("steps", []any{step})
			_, err := ValidateScenarios([]any{alone}, ieds)
			return err
		})
		if len(steps) == 0 && len(items(pyval.Lookup(scenario, "steps", nil))) > 0 {
			p.record(where, "", "every step used a reference the new file does not have, so the scenario is removed")
			continue
		}
		kept = append(kept, scenario.Set("steps", steps))
	}
	return kept
}
