package service

import (
	"studio-engine/internal/gateway/masterpoints"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// The SCADA master's own point list (ADR-0110, Go engine only): addresses and types the master
// expects, saved in the master block as "point_list" (one list per protocol) beside the
// connection settings. It changes nothing the device sends; it is compared with it.

// List is the saved point list of the current protocol.
func (m *Master) List() (string, []any) {
	config := m.Config()
	protocol := text(config, "protocol")
	list := masterpoints.StoredList(config, protocol)
	if list == nil {
		list = []any{}
	}
	return protocol, list
}

// SaveList checks and saves the point list of the current protocol (it may change while
// connected: the next Modbus poll reads the new points). It answers the stored list.
func (m *Master) SaveList(raw any) ([]any, error) {
	config := m.Config()
	protocol := text(config, "protocol")
	list, err := masterpoints.Validate(protocol, raw)
	if err != nil {
		return nil, err
	}
	if err := m.Save(masterpoints.WithList(config, protocol, list)); err != nil {
		return nil, err
	}
	m.changed(ChangeMasterStatus)
	return list, nil
}

// SetCheckIED turns the optional "Check against IED" column on or off (saved, off by default).
func (m *Master) SetCheckIED(on bool) error {
	return m.Save(with(m.Config(), "check_ied", on))
}

// CheckIED reports whether the "Check against IED" column is on.
func (m *Master) CheckIED() bool { return masterpoints.CheckIED(m.Config()) }

// Link is what the expected-vs-received comparison needs: connected (also while retrying a
// lost link) and whether a general interrogation, integrity poll or Modbus read finished.
func (m *Master) Link() masterpoints.Link {
	m.mu.Lock()
	defer m.mu.Unlock()
	return masterpoints.Link{Connected: m.process != nil, Asked: m.process != nil && m.lastAsked != nil}
}

// pollRanges are the Modbus reads of one poll: the user's ranges, then the reads the point
// list needs (merged and capped per request); the user's ranges keep their numbers.
func (m *Master) pollRanges(config *pyjson.Object) []any {
	manual, _ := get(config, "modbus_ranges").([]any)
	ranges := append([]any{}, manual...)
	for _, auto := range masterpoints.PollRanges(masterpoints.StoredList(config, "modbus")) {
		if !hasRange(ranges, auto.(*pyjson.Object)) {
			ranges = append(ranges, auto)
		}
	}
	return ranges
}

// hasRange: one of ranges already reads every value of wanted the same way (table, data type
// and word order), so the list's read is not sent twice.
func hasRange(ranges []any, wanted *pyjson.Object) bool {
	first, _ := pytext.Int64(get(wanted, "start"))
	count, _ := pytext.Int64(get(wanted, "count"))
	width := int64(1)
	if dataType := text(wanted, "data_type"); dataType == "int32" || dataType == "uint32" || dataType == "float32" {
		width = 2
	}
	for _, raw := range ranges {
		item, _ := raw.(*pyjson.Object)
		same := true
		for _, key := range []string{"table", "data_type", "word_order"} {
			same = same && pytext.Equal(get(item, key), get(wanted, key))
		}
		start, _ := pytext.Int64(get(item, "start"))
		length, _ := pytext.Int64(get(item, "count"))
		aligned := (first-start)%width == 0
		if same && aligned && start <= first && first+count*width <= start+length*width {
			return true
		}
	}
	return false
}
