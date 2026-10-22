package masterpoints

import (
	"strconv"

	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
)

// MaxRange is the most points one "Add a range" plans.
const MaxRange = 1000

// One plans one point typed in the form (type, address, name, scale, offset, unit, select,
// value, word order); the stored shape and the address are checked against the list.
func One(protocol string, raw *pyjson.Object, current []any) []Result {
	point, problem := newPlanner(protocol, current).add(raw)
	if point == nil {
		point = copyObject(raw)
	}
	return []Result{{Number: 1, Point: point, Problem: problem}}
}

// Range plans count points of one type at consecutive addresses from the first one (a Modbus
// float32 or int32 steps by two registers). Names are "<name> 1", "<name> 2", ... or, without a
// name, the type and its address. A used address stops nothing silently: that row says why.
func Range(protocol, typed string, from int64, count int, name string, current []any) []Result {
	entry, known := addpoints.MasterEntry(protocol, typed)
	if !known {
		return []Result{{Number: 1, Point: pyjson.NewObject().Set("type", typed),
			Problem: "Choose one of the " + ProtocolName(protocol) + " types"}}
	}
	if count < 1 || count > MaxRange {
		return []Result{{Number: 1, Point: pyjson.NewObject().Set("type", typed),
			Problem: "Add 1 to " + strconv.Itoa(MaxRange) + " points at a time"}}
	}
	plan, results := newPlanner(protocol, current), make([]Result, 0, count)
	for at := 0; at < count; at++ {
		address := from + int64(at)*entry.Registers()
		label := entry.Name + " " + strconv.FormatInt(address, 10)
		if name != "" {
			label = name + " " + strconv.Itoa(at+1)
		}
		raw := pyjson.NewObject().Set("type", entry.Key).Set("address", address).Set("name", label)
		point, problem := plan.add(raw)
		if point == nil {
			point = raw
		}
		results = append(results, Result{Number: at + 1, Point: point, Problem: rowProblem(at+1, problem)})
	}
	return results
}
