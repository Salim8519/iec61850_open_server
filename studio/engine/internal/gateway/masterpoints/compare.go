package masterpoints

import (
	"strconv"

	"studio-engine/internal/gateway"
	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
)

// The states of a listed point, compared with what the device reported.
const (
	StateReceived  = "received"   // the device sent this address with this type
	StateMissing   = "missing"    // nothing at this address after asking for every value
	StateWrongType = "wrong_type" // the address arrived, but as another type
	StateWaiting   = "waiting"    // not connected yet, or the first ask has not finished
	StateCommand   = "command"    // a command point: devices do not report commands
)

// Link is what the comparison needs to know about the connection.
type Link struct {
	Connected bool // the master runs (also while the link is being tried again)
	Asked     bool // a general interrogation, integrity poll or Modbus read finished
}

var askWords = map[string]string{"iec104": "the general interrogation", "dnp3": "the integrity poll",
	"modbus": "reading"}

type cellKey struct {
	space   string
	address int64
}

// Compare classifies every listed point as received, missing, wrong type, waiting or command,
// and lists the reported points that no listed point claims ("Not in your list"). The master
// compares addresses and types only; it never asks an IED what it should have received.
func Compare(protocol string, list []any, rows []any, link Link) *pyjson.Object {
	reported := ReadReceived(rows)
	byCell := map[cellKey][]int{}
	for at, item := range reported {
		key := cellKey{groupSpace(protocol, item.Group), item.Address}
		byCell[key] = append(byCell[key], at)
	}
	claimed := make([]bool, len(reported))
	counts := map[string]int64{StateReceived: 0, StateMissing: 0, StateWrongType: 0, StateWaiting: 0, StateCommand: 0}
	items := []any{}
	for _, raw := range list {
		point, ok := raw.(*pyjson.Object)
		entry, known := addpoints.MasterEntry(protocol, text(point, "type"))
		if !ok || !known {
			continue
		}
		item := classify(protocol, point, entry, reported, byCell, claimed, link)
		counts[text(item, "state")]++
		items = append(items, item)
	}
	extra := []any{}
	for at, item := range reported {
		if !claimed[at] {
			extra = append(extra, extraRow(protocol, item))
		}
	}
	summary := pyjson.NewObject()
	for _, state := range []string{StateReceived, StateMissing, StateWrongType, StateWaiting, StateCommand} {
		summary.Set(state, counts[state])
	}
	summary.Set("not_in_list", int64(len(extra))).Set("total", int64(len(items)))
	return pyjson.NewObject().Set("rows", items).Set("not_in_list", extra).Set("counts", summary).
		Set("asked", link.Asked).Set("connected", link.Connected)
}

func classify(protocol string, point *pyjson.Object, entry addpoints.MasterType, reported []Received,
	byCell map[cellKey][]int, claimed []bool, link Link) *pyjson.Object {
	item := pyjson.NewObject().Set("id", text(point, "id")).Set("point", point).Set("command", CommandOf(protocol, point))
	group, space := entry.Group, Space(protocol, entry)
	if group == "" && entry.StatusGroup != "" {
		group, space = entry.StatusGroup, groupSpace(protocol, entry.StatusGroup)
	}
	if group == "" {
		return item.Set("state", StateCommand).Set("reason", "A command point: devices do not report commands. Use Send.")
	}
	here := byCell[cellKey{space, Address(point)}]
	for _, at := range here {
		if found := reported[at]; found.Group == group && (protocol != "modbus" || found.Type == entry.DataType) {
			claimed[at] = true
			return withReceived(item.Set("state", StateReceived).Set("reason", ""), point, found)
		}
	}
	var other *Received
	for _, at := range here {
		if !claimed[at] && other == nil {
			claimed[at], other = true, &reported[at] // the same address, sent as another type
		}
	}
	if other != nil {
		return withReceived(item.Set("state", StateWrongType).Set("reason", wrongTypeReason(protocol, entry, *other)),
			point, *other)
	}
	if entry.Group == "" { // a command whose output status is not reported
		return item.Set("state", StateCommand).Set("reason", "A command point: no output status at this "+
			AddressWord[protocol]+" yet. Use Send.")
	}
	where := AddressWord[protocol] + " " + strconv.FormatInt(Address(point), 10)
	switch {
	case !link.Connected:
		return item.Set("state", StateWaiting).Set("reason", "Not connected. Connect to compare.")
	case !link.Asked:
		return item.Set("state", StateWaiting).Set("reason", "Waiting for "+askWords[protocol]+" to finish.")
	}
	return item.Set("state", StateMissing).Set("reason", "Nothing received at "+where+" after "+askWords[protocol]+".")
}

func wrongTypeReason(protocol string, entry addpoints.MasterType, found Received) string {
	where := AddressWord[protocol] + " " + strconv.FormatInt(found.Address, 10)
	if protocol == "modbus" && found.Group == entry.Group {
		return "Read as " + found.Type + " at " + where + ", but the list says " + entry.DataType +
			". Check the ranges in What to read."
	}
	sent := gateway.GroupName(protocol, found.Group)
	if found.Type != "" {
		sent += " (" + found.Type + ")"
	}
	return "The device sends a " + sent + " at " + where + ", not a " + entry.Name + "."
}

// withReceived adds what arrived: the value in words, the value after scale and offset, its
// quality, time and whether it is a last-known value while the link is down.
func withReceived(item, point *pyjson.Object, found Received) *pyjson.Object {
	row := found.Row
	return item.Set("received_key", found.Key).Set("received_group", found.Group).Set("received_type", found.Type).
		Set("value", field(row, "value")).Set("value_text", text(row, "value_text")).
		Set("engineering", engineering(point, found)).Set("quality_text", text(row, "quality_text")).
		Set("good", field(row, "good") == true).Set("stale", field(row, "stale") == true).
		Set("time_ms", field(row, "time_ms")).Set("cause", text(row, "cause"))
}

func extraRow(protocol string, item Received) *pyjson.Object {
	row := item.Row
	result := pyjson.NewObject().Set("key", item.Key).Set("group", item.Group).
		Set("group_name", gateway.GroupName(protocol, item.Group)).Set("address", item.Address).Set("type", item.Type).
		Set("label", text(row, "label")).Set("value_text", text(row, "value_text")).
		Set("quality_text", text(row, "quality_text")).Set("good", field(row, "good") == true)
	if entry, known := TypeOfReceived(protocol, item); known {
		result.Set("list_type", entry.Key)
	} else {
		result.Set("list_type", nil)
	}
	return result
}
