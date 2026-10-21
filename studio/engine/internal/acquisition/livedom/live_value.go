// Package livedom holds the pure rules of reading real IEDs: value conversion by SCD type,
// model comparison, the report/polling access choice and the MMS client list (ports of
// domain/live_value.py, model_compare.py, access_policy.py and mms_access.py).
package livedom

import (
	"errors"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

var expectedMMS = map[string][]int64{
	"BOOLEAN": {2}, "INT8": {4}, "INT16": {4}, "INT32": {4}, "INT64": {4}, "INT128": {4},
	"INT8U": {5, 4}, "INT16U": {5, 4}, "INT24U": {5, 4}, "INT32U": {5, 4},
	"FLOAT32": {6}, "FLOAT64": {6}, "Enum": {4}, "Dbpos": {3}, "Tcmd": {3}, "Quality": {3},
	"Check": {3}, "Timestamp": {14}, "EntryTime": {10}, "Octet64": {7}, "Octet6": {7}, "Octet8": {7},
	"EntryID": {7}, "VisString32": {8}, "VisString64": {8}, "VisString65": {8}, "VisString129": {8},
	"VisString255": {8}, "ObjRef": {8}, "Currency": {8}, "Unicode255": {13}, "OptFlds": {3},
	"TrgOps": {3},
}

var mmsNames = map[int64]string{0: "array", 1: "structure", 2: "boolean", 3: "bit string", 4: "integer",
	5: "unsigned", 6: "float", 7: "octet string", 8: "visible string",
	9: "generalized time", 10: "binary time", 11: "BCD", 12: "object id",
	13: "unicode string", 14: "UTC time", 15: "access error"}

// Convert returns the ScadaBench value for one device value, or an error whose text is the
// Python ValueError message.
func Convert(kind string, raw any) (any, error) {
	if raw == nil {
		return nil, errors.New("the device returned no value")
	}
	if object, ok := raw.(*pyjson.Object); ok {
		switch {
		case pyv.Has(object, "access_error"):
			return nil, errors.New("the device refused the read (access error " +
				pyv.PyStr(pyv.Get(object, "access_error")) + ")")
		case pyv.Has(object, "bits"):
			if kind == "Dbpos" || kind == "Tcmd" {
				return pyv.Get(object, "be"), nil
			}
			return pyv.Get(object, "le"), nil
		case pyv.Has(object, "octets"):
			return pyv.Get(object, "octets"), nil
		}
		return nil, errors.New("the device sent a value type ScadaBench cannot show yet")
	}
	_, isBool := raw.(bool)
	switch {
	case kind == "BOOLEAN":
		if !isBool {
			return nil, errors.New("expected true/false from the device")
		}
		return raw, nil
	case strings.HasPrefix(kind, "FLOAT"):
		if f, ok := asFloat(raw); ok && !isBool {
			return f, nil
		}
		return nil, errors.New("expected a number from the device")
	case strings.HasPrefix(kind, "INT") || kind == "Enum" || kind == "Timestamp" || kind == "EntryTime":
		if !pyv.IsInt(raw) {
			return nil, errors.New("expected a whole number from the device")
		}
		return raw, nil
	}
	if text, ok := raw.(string); ok {
		return text, nil
	}
	return nil, errors.New("the device value does not match the SCD type")
}

func asFloat(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case *big.Int:
		f, _ := new(big.Float).SetInt(v).Float64()
		return f, true
	}
	return 0, false
}

// TypeMatches reports whether the device MMS type fits the SCD basic type.
func TypeMatches(kind string, mmsType any) bool {
	expected, found := expectedMMS[kind]
	if !found {
		return true
	}
	for _, code := range expected {
		if pyv.Equal(code, mmsType) {
			return true
		}
	}
	return false
}

// DescribeMMS names an MMS type code, e.g. "integer" or "MMS type 99".
func DescribeMMS(mmsType any) string {
	if code, ok := pyv.Int64(mmsType); ok {
		if name, found := mmsNames[code]; found {
			return name
		}
		return "MMS type " + strconv.FormatInt(code, 10)
	}
	if f, ok := mmsType.(float64); ok && f == float64(int64(f)) { // Python: MMS_NAMES.get(4.0) finds 4
		if name, found := mmsNames[int64(f)]; found {
			return name
		}
	}
	if b, ok := mmsType.(bool); ok {
		if name, found := mmsNames[map[bool]int64{false: 0, true: 1}[b]]; found {
			return name
		}
	}
	return "MMS type " + pyv.PyStr(mmsType)
}
