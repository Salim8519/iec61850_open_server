package project

import (
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// legacyFields are the flat mapping fields of formats 1-3.
var legacyFields = []string{"ied_id", "status_ref", "control_ref", "report_ref", "status_kind", "position_values"}

// relayEntries turns a format-8 relay state mapping into one named relay status.
func (m *migration) relayEntries() error {
	nodes, err := m.list("nodes")
	if err != nil {
		return err
	}
	for _, node := range nodes {
		data, err := nodeData(node)
		if err != nil {
			return err
		}
		if pyval.Lookup(data, "kind", nil) != "RELAY" {
			continue
		}
		mapping, _ := pyval.Pop(data, "state_mapping", nil)
		if !pyval.Truthy(mapping) {
			continue
		}
		id, _ := pyval.Get(node, "id", "relay")
		label := pyval.Lookup(data, "label", id)
		mode, err := pyval.Get(mapping, "mode", nil)
		if err != nil {
			return err
		}
		var source any
		if direct, err := pyval.InStrings(mode, "direct", "single_to_dbpos"); err != nil {
			return err
		} else if direct {
			source, _ = pyval.Get(mapping, "source", nil)
		}
		if pyval.Truthy(source) {
			entry := pyjson.NewObject().Set("id", "s1").Set("label", label).Set("source", source)
			data.Set("relay_statuses", []any{entry})
			m.notice(pyval.Str(label) + ": existing state mapping became one named relay status")
		} else {
			m.notice(pyval.Str(label) + ": removed unsupported state mapping while upgrading relays")
		}
	}
	return nil
}

// indexKey hashes the (ied id, ref) tuple key of _indexes; only two strings can match.
func indexKey(iedID, ref any) (signalKey, bool, error) {
	return lookupKey(iedID, ref)
}

func controlIndex(ieds []*pyjson.Object) map[signalKey]*pyjson.Object {
	index := map[signalKey]*pyjson.Object{}
	for _, ied := range ieds {
		for _, item := range items(pyval.Lookup(ied, "controls", nil)) {
			control := object(item)
			index[signalKey{text(ied, "id"), text(control, "ref")}] = control
		}
	}
	return index
}

// flatMappings converts the flat format 1-3 fields into state and control bindings.
func (m *migration) flatMappings() error {
	signals, controls := signalIndex(m.ieds), controlIndex(m.ieds)
	nodes, err := m.list("nodes")
	if err != nil {
		return err
	}
	for _, node := range nodes {
		data, err := nodeData(node)
		if err != nil {
			return err
		}
		kind := pyval.Lookup(data, "kind", nil)
		label := labelOf(data, node, "equipment")
		iedID, statusRef := pyval.Lookup(data, "ied_id", nil), pyval.Lookup(data, "status_ref", nil)
		key, textual, err := indexKey(iedID, statusRef)
		if err != nil {
			return err
		}
		if signal := signals[key]; textual && pyval.Truthy(signal) {
			if err := m.flatState(data, kind, label, signal); err != nil {
				return err
			}
		}
		controlRef := pyval.Lookup(data, "control_ref", nil)
		key, textual, err = indexKey(iedID, controlRef)
		if err != nil {
			return err
		}
		control := controls[key]
		matched := false
		if textual && pyval.Truthy(control) {
			if matched, err = pyval.InStrings(kind, switches...); err != nil {
				return err
			}
			matched = matched && pyval.Lookup(control, "cdc", nil) == "DPC"
		}
		if matched {
			data.Set("control_binding", pyjson.NewObject().Set("ied_id", iedID).Set("ref", controlRef))
		} else if pyval.Truthy(controlRef) {
			m.notice(label + ": removed incompatible control mapping")
		}
		for _, field := range legacyFields {
			data.Delete(field)
		}
	}
	return nil
}

func (m *migration) flatState(data *pyjson.Object, kind any, label string, signal *pyjson.Object) error {
	binding := pyjson.NewObject().Set("ied_id", pyval.Lookup(data, "ied_id", nil)).
		Set("ref", pyval.Lookup(data, "status_ref", nil)).
		Set("report_ref", pyval.Or(pyval.Lookup(data, "report_ref", nil), nil))
	isSwitchKind, err := pyval.InStrings(kind, switches...)
	if err != nil {
		return err
	}
	signalKind := pyval.Lookup(signal, "kind", nil)
	switch {
	case isSwitchKind && signalKind == "BOOLEAN":
		data.Set("state_mapping", pyjson.NewObject().Set("mode", "single_to_dbpos").Set("source", binding))
	case !isSwitchKind || signalKind == "Dbpos":
		data.Set("state_mapping", pyjson.NewObject().Set("mode", "direct").Set("source", binding))
	default:
		m.notice(label + ": removed unsupported " + pyval.Str(signalKind) + " switch state mapping")
	}
	return nil
}

// tripPolarity sets the trip command and state polarity of formats before 7.
func (m *migration) tripPolarity(version int64) error {
	nodes, err := m.list("nodes")
	if err != nil {
		return err
	}
	fallback := "operate_false"
	if version <= 4 {
		fallback = "operate_true"
	}
	for _, node := range nodes {
		data, err := nodeData(node)
		if err != nil {
			return err
		}
		old, _ := pyval.Pop(data, "trip_polarity", fallback)
		if pyval.Lookup(data, "kind", nil) != "TRIP_RELAY" {
			continue
		}
		label := labelOf(data, node, "equipment")
		if version == 6 {
			data.Set("trip_command_polarity", old)
		} else {
			data.Set("trip_command_polarity", "operate_false")
		}
		data.Set("trip_state_polarity", "operated_true")
		if version <= 5 && old == "operate_true" {
			m.notice(label + ": Trip command polarity changed to Operate=0 / Reset=1")
		}
		if version == 6 {
			m.notice(label + ": Trip state polarity set to Operated=1 / Reset=0")
		}
	}
	return nil
}
