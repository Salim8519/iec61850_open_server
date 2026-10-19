package scl

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
)

// basicTypes is type_resolver.TYPES: SCL basic type -> native model type number.
var basicTypes = map[string]int{
	"BOOLEAN": 0, "INT8": 1, "INT16": 2, "INT32": 3, "INT64": 4,
	"INT128": 5, "INT8U": 6, "INT16U": 7, "INT24U": 8, "INT32U": 9,
	"FLOAT32": 10, "FLOAT64": 11, "Enum": 12, "Octet64": 13,
	"Octet6": 14, "Octet8": 15, "VisString32": 16, "VisString64": 17,
	"VisString65": 18, "VisString129": 19, "VisString255": 20,
	"Unicode255": 21, "Timestamp": 22, "Quality": 23, "Check": 24,
	"Dbpos": 25, "Tcmd": 25, "Struct": 27, "EntryTime": 28,
	"Currency": 30, "OptFlds": 31, "TrgOps": 32, "ObjRef": 19, "EntryID": 15, "PhyComAddr": 29,
}

// functionalConstraints is type_resolver.FCS.
var functionalConstraints = func() map[string]int {
	result := map[string]int{}
	for index, name := range strings.Fields("ST MX SP SV CF DC SG SE SR OR BL EX CO US MS RP BR LG GO") {
		result[name] = index
	}
	return result
}()

func isTextKind(kind string) bool {
	return strings.Contains(kind, "String") || kind == "Currency" || kind == "ObjRef"
}

// initialValue is type_resolver.initial_value: bool, string, int64/*big.Int or float64.
func initialValue(kind string, text string, present bool, enums *pyjson.Object) (any, error) {
	if !present || pyrules.Strip(text) == "" {
		switch {
		case kind == "BOOLEAN":
			return false, nil
		case isTextKind(kind):
			return "", nil
		}
		return int64(0), nil
	}
	text = pyrules.Strip(text)
	if kind == "Timestamp" && strings.Contains(text, "T") {
		millis, err := pyrules.ISOTimestampMillis(text)
		if err != nil {
			return nil, err
		}
		return millis, nil
	}
	if kind == "BOOLEAN" {
		lower := pyrules.Lower(text)
		if lower != "true" && lower != "false" && lower != "1" && lower != "0" {
			return nil, apperr.New("Invalid BOOLEAN initial value " + pyrules.Repr(text))
		}
		return lower == "true" || lower == "1", nil
	}
	if isTextKind(kind) {
		return text, nil
	}
	if kind == "Enum" {
		if value, ok := enums.Get(text); ok {
			return value, nil
		}
	}
	if strings.HasPrefix(kind, "FLOAT") {
		number, err := pyrules.Float(text)
		if err != nil {
			return nil, apperr.ValueError(pyrules.FloatMessage(text))
		}
		if math.IsInf(number, 0) || math.IsNaN(number) {
			return nil, apperr.New("Non-finite initial measurement")
		}
		return number, nil
	}
	number, err := pyrules.IntBig(text, 10)
	if err != nil {
		return nil, apperr.New("Unsupported " + kind + " initial value " + pyrules.Repr(text))
	}
	return intValue(number), nil
}

// intValue returns an int64 when it fits, else the big integer (Python ints are unbounded).
func intValue(number *big.Int) any {
	if number.IsInt64() {
		return number.Int64()
	}
	return number
}

var noInitial = map[string]bool{"Struct": true, "Timestamp": true, "Quality": true, "Check": true,
	"Dbpos": true, "Tcmd": true, "EntryTime": true, "EntryID": true, "Octet64": true, "Octet6": true,
	"Octet8": true, "PhyComAddr": true}

// configInitial is type_resolver.config_initial: the "=value" suffix of a DA line.
func configInitial(kind string, value any) (string, error) {
	if noInitial[kind] {
		return "", nil
	}
	switch v := value.(type) {
	case string:
		if strings.ContainsAny(v, "\"\n\r") {
			return "", apperr.New("Quoted/newline string values need native model encoding")
		}
		return "=" + string(pyjson.AppendString(nil, v)), nil
	case bool:
		if v {
			return "=1", nil
		}
		return "=0", nil
	}
	return "=" + pyStr(value), nil
}

// pyStr is Python str() of an int or float model value.
func pyStr(value any) string {
	switch v := value.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	case *big.Int:
		return v.String()
	case float64:
		return pyjson.Repr(v)
	case bool:
		if v {
			return "True"
		}
		return "False"
	case string:
		return v
	case nil:
		return "None"
	}
	return ""
}

// triggerBits is type_resolver.trigger_bits.
func triggerBits(get func(string) (string, bool)) int {
	bits := 0
	for _, pair := range []struct {
		key string
		bit int
	}{{"dchg", 1}, {"qchg", 2}, {"dupd", 4}} {
		if value, ok := get(pair.key); ok && value == "true" {
			bits += pair.bit
		}
	}
	return bits
}
