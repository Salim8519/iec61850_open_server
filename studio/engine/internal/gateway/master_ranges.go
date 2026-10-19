package gateway

import (
	"math/big"
	"strconv"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Modbus tables with their per-request value limit, and register widths per data type.
var (
	tables    = map[string]int64{"coils": 2000, "discrete_inputs": 2000, "holding_registers": 125, "input_registers": 125}
	dataTypes = map[string]int64{"bool": 1, "uint16": 1, "int16": 1, "uint32": 2, "int32": 2, "float32": 2}
)

const (
	maxRanges = 20
	maxProbe  = 10000
)

// lookupName is `value in mapping` for a str-keyed dict: TypeError when unhashable.
func lookupName[T any](mapping map[string]T, value any) (string, bool, error) {
	if err := hashable(value); err != nil {
		return "", false, err
	}
	text, isText := value.(string)
	_, found := mapping[text]
	return text, isText && found, nil
}

func isBits(table any) bool { return table == "coils" || table == "discrete_inputs" }

// ValidateRanges is validate_ranges(ranges): the Modbus read list, one request per range.
func ValidateRanges(ranges any) ([]any, error) {
	list, ok := ranges.([]any)
	if !ok || len(list) > maxRanges {
		return nil, valueError("Send at most " + strconv.Itoa(maxRanges) + " Modbus ranges")
	}
	clean := []any{}
	for index, value := range list {
		label := "Range " + itoa(index+1) + ": "
		raw, isDict := value.(*pyjson.Object)
		table, known := "", false
		if isDict {
			var err error
			if table, known, err = lookupName(tables, get(raw, "table")); err != nil {
				return nil, err
			}
		}
		if !known {
			return nil, valueError(label + "choose coils, discrete inputs, holding registers or input registers")
		}
		bits := isBits(table)
		fallback := "uint16"
		if bits {
			fallback = "bool"
		}
		dataType, known, err := lookupName(dataTypes, getOr(raw, "data_type", fallback))
		if err != nil {
			return nil, err
		}
		if !known || (dataType == "bool") != bits {
			return nil, valueError(label + "coils and inputs are read as bool, registers as uint16, int16, uint32, int32 or float32")
		}
		start, count := get(raw, "start"), get(raw, "count")
		limit := tables[table] / dataTypes[dataType]
		if !pytext.IntIn(start, 0, 65535) {
			return nil, valueError(label + "the start address is a whole number from 0 to 65535")
		}
		if !pytext.IntIn(count, 1, limit) {
			return nil, valueError(label + "read 1 to " + strconv.FormatInt(limit, 10) + " values of this type in one range")
		}
		first, _ := pytext.Int64(start)
		number, _ := pytext.Int64(count)
		if first+number*dataTypes[dataType] > 65536 {
			return nil, valueError(label + "the range runs past address 65535")
		}
		swap := getOr(raw, "word_order", "big")
		if swap != "big" && swap != "little" {
			return nil, valueError(label + "the word order is big (high word first) or little")
		}
		clean = append(clean, pyjson.NewObject().Set("table", table).Set("start", start).Set("count", count).
			Set("data_type", dataType).Set("word_order", swap))
	}
	return clean, nil
}

// ValidateProbe is validate_probe(table, start, end): Find registers reads at most 10000
// addresses of one table.
func ValidateProbe(table, start, end any) (string, int64, int64, error) {
	name, known, err := lookupName(tables, table)
	if err != nil {
		return "", 0, 0, err
	}
	if !known {
		return "", 0, 0, valueError("Choose coils, discrete inputs, holding registers or input registers")
	}
	for _, value := range []any{start, end} {
		if !pytext.IntIn(value, 0, 65535) {
			return "", 0, 0, valueError("Addresses are whole numbers from 0 to 65535")
		}
	}
	first, _ := pytext.Int64(start)
	last, _ := pytext.Int64(end)
	if last < first || last-first+1 > maxProbe {
		return "", 0, 0, valueError("Find registers checks 1 to " + strconv.Itoa(maxProbe) + " addresses at a time")
	}
	return name, first, last, nil
}

// RangesFromProbe is ranges_from_probe(found, data_type): answering spans
// [{table, start, count}] as read ranges split to the per-request limit. Spans come from
// the probe; other shapes give the Python crash as a plain error.
func RangesFromProbe(found []any, dataType any) ([]any, error) {
	kindName, known, err := lookupName(dataTypes, dataType)
	if err != nil {
		return nil, err
	}
	if !known || kindName == "bool" {
		return nil, valueError("Read registers as uint16, int16, uint32, int32 or float32")
	}
	ranges := []any{}
	for _, span := range found {
		table, err := subscript(span, "table")
		if err != nil {
			return nil, err
		}
		kind := kindName
		if isBits(table) {
			kind = "bool"
		}
		width := dataTypes[kind]
		name, known, err := lookupName(tables, table)
		if err != nil {
			return nil, err
		}
		if !known {
			return nil, keyError(table)
		}
		limit := tables[name] / width
		countValue, err := subscript(span, "count")
		if err != nil {
			return nil, err
		}
		count, err := spanCount(countValue, width)
		if err != nil {
			return nil, err
		}
		for offset := int64(0); offset < count; offset += limit {
			startValue, err := subscript(span, "start")
			if err != nil {
				return nil, err
			}
			first, err := plusInt(startValue, offset*width)
			if err != nil {
				return nil, err
			}
			ranges = append(ranges, pyjson.NewObject().Set("table", table).Set("start", first).
				Set("count", min(limit, count-offset)).Set("data_type", kind).Set("word_order", "big"))
		}
	}
	return ranges, nil
}

// spanCount is range()'s view of span["count"] // width (probe spans hold int counts).
func spanCount(value any, width int64) (int64, error) {
	switch value.(type) {
	case bool, int, int64:
		total, _ := pytext.Int64(value)
		return floorDiv(total, width), nil
	case float64:
		return 0, crash("TypeError", "'float' object cannot be interpreted as an integer")
	case *big.Int:
		return 0, crash("OverflowError", "the span count is too large") // unreachable: probes read at most 10000
	}
	return 0, crash("TypeError", "unsupported operand type(s) for //: '"+typeName(value)+"' and 'int'")
}

// plusInt is value + number for span["start"].
func plusInt(value any, number int64) (any, error) {
	switch v := value.(type) {
	case bool, int, int64, *big.Int:
		return small(new(big.Int).Add(toBig(v), big.NewInt(number))), nil
	case float64:
		return v + float64(number), nil
	case string, []any:
		name := typeName(value)
		return nil, crash("TypeError", "can only concatenate "+name+" (not \"int\") to "+name)
	}
	return nil, crash("TypeError", "unsupported operand type(s) for +: '"+typeName(value)+"' and 'int'")
}
