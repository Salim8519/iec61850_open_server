package runtime

import (
	"math"
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// typedValue checks what the browser sent for one control against its class (ADR-0108) and
// returns the TypedCommand value: true/false for SPC and DPC, a whole number for INC, a number
// for APC, one of the enum labels for ENC and "higher" or "lower" for BAC, BSC and ISC. min, max and
// step of the typed payload bound INC and APC values. Messages say what to send instead.
func typedValue(control *pyjson.Object, value any, label string) (any, error) {
	cdc := pyval.Str(control, "cdc")
	switch cdc {
	case "SPC", "DPC":
		if flag, isBool := value.(bool); isBool {
			return flag, nil
		}
		return nil, apperr.New(label + " is sent as true or false: on or off for a single command, " +
			"close or open for a double command.")
	case "INC":
		number, ok := wholeNumber(value)
		if !ok || pyval.IsBool(value) {
			return nil, apperr.New(label + " takes a whole number" + rangeWords(control) + ".")
		}
		if err := inRange(control, float64(number), label); err != nil {
			return nil, err
		}
		return number, nil
	case "APC":
		number, isNumber := pyval.Float(value)
		if pyval.IsBool(value) || !isNumber || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, apperr.New(label + " takes a number" + rangeWords(control) + ".")
		}
		if err := inRange(control, number, label); err != nil {
			return nil, err
		}
		return number, nil
	case "ENC":
		labels := enumLabels(control)
		if text, isText := value.(string); isText {
			for _, known := range labels {
				if known == text {
					return text, nil
				}
			}
		}
		if len(labels) == 0 {
			return nil, apperr.New(label + " has no list of values in this IED's model, so nothing can be chosen.")
		}
		return nil, apperr.New(label + " takes one of: " + strings.Join(labels, ", ") + ".")
	case "BAC", "BSC", "ISC":
		if value == "higher" || value == "lower" {
			return value, nil
		}
		return nil, apperr.New(label + " is a raise/lower command: send \"higher\" (Raise) or \"lower\" (Lower).")
	}
	return nil, apperr.New(label + " uses a " + cdc + " control, which ScadaBench cannot command.")
}

// enumLabels are the labels of an ENC control in the model's order: the keys of `labels`
// (label to ordinal, ADR-0104).
func enumLabels(control *pyjson.Object) []string {
	if labels, ok := pyval.Get(control, "labels").(*pyjson.Object); ok {
		return labels.Keys()
	}
	return nil
}

// bound reads min, max or step; ok is false when the model has none.
func bound(control *pyjson.Object, key string) (float64, bool) {
	value := pyval.Get(control, key)
	if value == nil || pyval.IsBool(value) {
		return 0, false
	}
	return pyval.Float(value)
}

// rangeWords is " from 1 to 17 in steps of 1" (each part only when the model has it).
func rangeWords(control *pyjson.Object) string {
	words := ""
	if low, ok := bound(control, "min"); ok {
		words += " from " + numberWords(low)
	}
	if high, ok := bound(control, "max"); ok {
		words += " up to " + numberWords(high)
	}
	if step, ok := bound(control, "step"); ok && step > 0 {
		words += " in steps of " + numberWords(step)
	}
	return words
}

// inRange refuses a value outside min..max or off the step grid (counted from min, else 0).
func inRange(control *pyjson.Object, number float64, label string) error {
	low, hasLow := bound(control, "min")
	high, hasHigh := bound(control, "max")
	if (hasLow && number < low) || (hasHigh && number > high) {
		return apperr.New(label + " takes values" + rangeWords(control) + ", not " + numberWords(number) + ".")
	}
	if step, ok := bound(control, "step"); ok && step > 0 {
		steps := (number - low) / step
		if math.Abs(steps-math.Round(steps)) > 1e-9*math.Max(1, math.Abs(steps)) {
			return apperr.New(label + " takes values" + rangeWords(control) + ", so " + numberWords(number) +
				" is between two steps.")
		}
	}
	return nil
}

// numberWords writes 17, 0.5 or 1e-06 the way a person reads them (no trailing .0).
func numberWords(number float64) string { return strconv.FormatFloat(number, 'g', -1, 64) }
