package goosemsg

import (
	"math"
	"strings"

	"studio-engine/internal/apperr"
)

// Value types a GOOSE value can be sent as from the SLD.
const (
	OnOff    = "onoff"    // BOOLEAN: on/off, or a switch's closed/open with one value
	Position = "position" // Dbpos: 0 intermediate, 1 open, 2 closed, 3 bad
	Number   = "number"   // INT*/FLOAT*
)

var numericKinds = map[string]bool{"INT8": true, "INT16": true, "INT32": true, "INT64": true, "INT128": true,
	"INT8U": true, "INT16U": true, "INT24U": true, "INT32U": true, "FLOAT32": true, "FLOAT64": true}

// ValueType is the way a leaf of this basic type is sent, or "" when it cannot be sent from here
// (quality, time stamp, enumerations, text).
func ValueType(kind string) string {
	switch {
	case kind == "BOOLEAN":
		return OnOff
	case kind == "Dbpos":
		return Position
	case numericKinds[kind]:
		return Number
	}
	return ""
}

// Sendable says whether ScadaBench can change this leaf in the simulated IED (the sample path
// refuses read-only and CF attributes) and has a control for its type.
func (l Leaf) Sendable() bool {
	return l.Writable && l.FC != "CF" && ValueType(l.Kind) != ""
}

// Write is one attribute value of one batch.
type Write struct {
	Ref   string
	Value any
}

// wireValue turns a requested value into what the IED holds for this leaf: true/false for an
// on/off value, a whole 0..3 for a position, a number (whole for integer kinds) otherwise.
func wireValue(leaf Leaf, value any) (any, error) {
	name := shortRef(leaf.Ref)
	switch ValueType(leaf.Kind) {
	case OnOff:
		if flag, ok := value.(bool); ok {
			return flag, nil
		}
		return nil, apperr.New(name + " is an on/off value: send ON or OFF (true or false).")
	case Position:
		if number, ok := whole(value); ok && number >= 0 && number <= 3 {
			return number, nil
		}
		return nil, apperr.New(name + " is a position: send 0 (intermediate), 1 (open), 2 (closed) or 3 (bad).")
	case Number:
		number, ok := float(value)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, apperr.New(name + " is a number: enter a number.")
		}
		if strings.HasPrefix(leaf.Kind, "INT") {
			rounded := math.Round(number)
			if math.Abs(number-rounded) > 1e-6*math.Max(1, math.Abs(number)) {
				return nil, apperr.New(name + " holds whole numbers only. Enter a whole number.")
			}
			return int64(rounded), nil
		}
		return number, nil
	}
	return nil, apperr.New(name + " (" + leaf.Kind + ") cannot be changed from here. Choose an on/off value, " +
		"a position or a number.")
}

func whole(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), typed == math.Trunc(typed) && math.Abs(typed) < 1e15
	}
	return 0, false
}

func float(value any) (float64, bool) {
	switch typed := value.(type) {
	case int64:
		return float64(typed), true
	case int:
		return float64(typed), true
	case float64:
		return typed, true
	}
	return 0, false
}

// shortRef drops the domain: "XCBR1.Pos.stVal".
func shortRef(ref string) string {
	if _, rest, found := strings.Cut(ref, "/"); found {
		return rest
	}
	return ref
}
