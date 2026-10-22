package masterpoints

import (
	"sort"

	"studio-engine/internal/pyjson"
)

// perRequest is how many values one Modbus read may ask for (bits for coils and inputs,
// registers for the register tables), as the manual ranges are capped.
var perRequest = map[string]int64{"coils": 2000, "discrete_inputs": 2000, "holding_registers": 125, "input_registers": 125}

type rangeKey struct{ table, dataType, order string }

type span struct {
	start, count int64 // count is in values of the data type
}

// PollRanges are the Modbus reads the listed points need: one read per run of neighbouring
// points of one table, data type and word order (no unlisted address is read), each split to
// the per-request limit. They are read with the user's own ranges every poll interval.
func PollRanges(list []any) []any {
	starts := map[rangeKey][]int64{}
	widths := map[rangeKey]int64{}
	for _, raw := range list {
		point, ok := raw.(*pyjson.Object)
		width, isInt := field(point, "registers").(int64)
		if !ok || !isInt || text(point, "table") == "" {
			continue
		}
		key := rangeKey{text(point, "table"), text(point, "data_type"), text(point, "word_order")}
		starts[key] = append(starts[key], Address(point))
		widths[key] = width
	}
	keys := make([]rangeKey, 0, len(starts))
	for key := range starts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.table != b.table {
			return a.table < b.table
		}
		if a.dataType != b.dataType {
			return a.dataType < b.dataType
		}
		return a.order < b.order
	})
	ranges := []any{}
	for _, key := range keys {
		width := widths[key]
		for _, part := range runs(starts[key], width, perRequest[key.table]/width) {
			ranges = append(ranges, pyjson.NewObject().Set("table", key.table).Set("start", part.start).
				Set("count", part.count).Set("data_type", key.dataType).Set("word_order", key.order))
		}
	}
	return ranges
}

// runs merges sorted addresses into consecutive spans of at most limit values.
func runs(addresses []int64, width, limit int64) []span {
	sort.Slice(addresses, func(i, j int) bool { return addresses[i] < addresses[j] })
	result := []span{}
	for _, address := range addresses {
		if n := len(result); n > 0 {
			last := &result[n-1]
			if end := last.start + last.count*width; address == end && last.count < limit {
				last.count++
				continue
			} else if address < end {
				continue // a repeated address
			}
		}
		result = append(result, span{start: address, count: 1})
	}
	return result
}
