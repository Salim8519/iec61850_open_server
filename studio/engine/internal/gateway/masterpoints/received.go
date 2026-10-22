package masterpoints

import (
	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
)

// Received is one point the device reported, as the master's discovered table shows it
// (gateway.Rows): key "<group>:<address>", group, address, type (the protocol's type id, or
// the Modbus data type), value and its words.
type Received struct {
	Row     *pyjson.Object
	Key     string
	Group   string
	Address int64
	Type    string
}

// ReadReceived reads the display rows of the discovered table.
func ReadReceived(rows []any) []Received {
	result := make([]Received, 0, len(rows))
	for _, raw := range rows {
		row, ok := raw.(*pyjson.Object)
		address, isInt := field(row, "address").(int64)
		if ok && isInt {
			result = append(result, Received{Row: row, Key: text(row, "key"), Group: text(row, "group"),
				Address: address, Type: text(row, "type")})
		}
	}
	return result
}

// groupSpace is the address space a reported group belongs to (as Space for a list type).
func groupSpace(protocol, group string) string {
	switch protocol {
	case "iec104":
		return "ioa"
	case "modbus":
		return group + "s"
	}
	return group
}

// TypeOfReceived is the master type a reported point is: its group (and Modbus data type).
func TypeOfReceived(protocol string, item Received) (addpoints.MasterType, bool) {
	for _, entry := range addpoints.MasterTypes(protocol) {
		if entry.Group == item.Group && (protocol != "modbus" || entry.DataType == item.Type) {
			return entry, true
		}
	}
	return addpoints.MasterType{}, false
}

// FromReceived plans list points from reported points (the "Not in your list" rows): the
// type the device sent, its address and the name given in the received list, if any.
func FromReceived(protocol string, rows []any, keys []string, current []any) []Result {
	wanted := map[string]bool{}
	for _, key := range keys {
		wanted[key] = true
	}
	plan, results := newPlanner(protocol, current), []Result{}
	for _, item := range ReadReceived(rows) {
		if !wanted[item.Key] {
			continue
		}
		number := len(results) + 1
		entry, known := TypeOfReceived(protocol, item)
		raw := pyjson.NewObject().Set("name", text(item.Row, "label")).Set("address", item.Address)
		if !known {
			results = append(results, Result{Number: number, Point: raw.Set("type", item.Group),
				Problem: rowProblem(number, "ScadaBench has no list type for "+item.Group+" "+item.Type)})
			continue
		}
		point, problem := plan.add(raw.Set("type", entry.Key))
		if point == nil {
			point = raw
		}
		results = append(results, Result{Number: number, Point: point, Problem: rowProblem(number, problem)})
	}
	return results
}
