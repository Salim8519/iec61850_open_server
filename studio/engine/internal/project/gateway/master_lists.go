package gateway

import (
	"strconv"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Modbus tables with the most values one request reads, and data type widths in registers.
var (
	tableLimits = map[string]int64{"coils": 2000, "discrete_inputs": 2000, "holding_registers": 125, "input_registers": 125}
	typeWidths  = map[string]int64{"bool": 1, "uint16": 1, "int16": 1, "uint32": 2, "int32": 2, "float32": 2}
)

func keysOf(table map[string]int64) []string {
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	return keys
}

// ValidateRanges is master_rules.validate_ranges: the Modbus ranges the master reads.
func ValidateRanges(ranges any) ([]any, error) {
	items, isList := ranges.([]any)
	if !isList || len(items) > 20 {
		return nil, apperr.ValueError("Send at most 20 Modbus ranges")
	}
	clean := []any{}
	for index, item := range items {
		label := "Range " + strconv.Itoa(index+1)
		raw, isDict := item.(*pyjson.Object)
		var table any
		known := false
		if isDict {
			table = pyval.Lookup(raw, "table", nil)
			var err error
			if known, err = pyval.InStrings(table, keysOf(tableLimits)...); err != nil {
				return nil, err
			}
		}
		if !known {
			return nil, apperr.ValueError(label + ": choose coils, discrete inputs, holding registers or input registers")
		}
		bits := table == "coils" || table == "discrete_inputs"
		fallback := "uint16"
		if bits {
			fallback = "bool"
		}
		dataType := pyval.Lookup(raw, "data_type", fallback)
		typed, err := pyval.InStrings(dataType, keysOf(typeWidths)...)
		if err != nil {
			return nil, err
		}
		if !typed || (dataType == "bool") != bits {
			return nil, apperr.ValueError(label + ": coils and inputs are read as bool, registers as uint16, int16, uint32, int32 or float32")
		}
		start, count := pyval.Lookup(raw, "start", nil), pyval.Lookup(raw, "count", nil)
		width := typeWidths[dataType.(string)]
		limit := tableLimits[table.(string)] / width
		if !pyval.IsInt(start) || !pyval.Between(start, 0, 65535) {
			return nil, apperr.ValueError(label + ": the start address is a whole number from 0 to 65535")
		}
		if !pyval.IsInt(count) || !pyval.Between(count, 1, float64(limit)) {
			return nil, apperr.ValueError(label + ": read 1 to " + strconv.FormatInt(limit, 10) + " values of this type in one range")
		}
		first, _ := pyval.BigInt(start)
		size, _ := pyval.BigInt(count)
		if first.Int64()+size.Int64()*width > 65536 {
			return nil, apperr.ValueError(label + ": the range runs past address 65535")
		}
		swap := pyval.Lookup(raw, "word_order", "big")
		if !pyval.InTuple(swap, "big", "little") {
			return nil, apperr.ValueError(label + ": the word order is big (high word first) or little")
		}
		clean = append(clean, pyjson.NewObject().Set("table", table).Set("start", start).Set("count", count).
			Set("data_type", dataType).Set("word_order", swap))
	}
	return clean, nil
}

// ValidateRoutes is master_rules.validate_routes: user command routes.
func ValidateRoutes(routes any) ([]any, error) {
	items, isList := routes.([]any)
	if !isList || len(items) > 500 {
		return nil, apperr.ValueError("Send at most 500 command routes as a list")
	}
	clean := []any{}
	for index, item := range items {
		label := "Route " + strconv.Itoa(index+1)
		raw, isDict := item.(*pyjson.Object)
		known := false
		if isDict {
			var err error
			known, err = pyval.InStrings(pyval.Lookup(raw, "kind", nil), "single", "double", "binary_output", "coil")
			if err != nil {
				return nil, err
			}
		}
		if !known {
			return nil, apperr.ValueError(label + ": the device command is single, double, binary_output or coil")
		}
		if !pyval.InTuple(pyval.Lookup(raw, "upstream_kind", nil), "command_single", "command_double") {
			return nil, apperr.ValueError(label + ": the upstream command is a single or double command")
		}
		for _, key := range []string{"upstream", "address"} {
			value := pyval.Lookup(raw, key, nil)
			if !pyval.IsInt(value) || !pyval.Between(value, 0, 16777215) {
				return nil, apperr.ValueError(label + ": " + key + " must be a whole number")
			}
		}
		upstream, _ := raw.Get("upstream")
		upstreamKind, _ := raw.Get("upstream_kind")
		kind, _ := raw.Get("kind")
		address, _ := raw.Get("address")
		clean = append(clean, pyjson.NewObject().Set("upstream", upstream).Set("upstream_kind", upstreamKind).
			Set("kind", kind).Set("address", address).Set("select", pyval.Lookup(raw, "select", true) == true))
	}
	return clean, nil
}
