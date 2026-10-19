package contract

import (
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"studio-engine/internal/project/pyval"
)

// text validates a str field with optional length limits (code points) and a pattern
// (Rust regex semantics: "$" is the end of the text, as in Go).
func text(minLength, maxLength int, pattern string) Validator {
	var compiled *regexp.Regexp
	if pattern != "" {
		compiled = regexp.MustCompile(pattern)
	}
	return func(c *collector, at loc, input any) (any, bool) {
		value, ok := input.(string)
		if !ok {
			c.add("string_type", at, "Input should be a valid string")
			return nil, false
		}
		length := pyval.Len(value)
		switch {
		case minLength > 0 && length < minLength:
			c.add("string_too_short", at, "String should have at least "+plural(minLength, "character"))
		case maxLength > 0 && length > maxLength:
			c.add("string_too_long", at, "String should have at most "+plural(maxLength, "character"))
		case compiled != nil && !compiled.MatchString(value):
			c.add("string_pattern_mismatch", at, "String should match pattern '"+pattern+"'")
		default:
			return value, true
		}
		return nil, false
	}
}

func plural(count int, noun string) string {
	if count == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(count) + " " + noun + "s"
}

// literal validates a Literal of strings.
func literal(choices ...string) Validator {
	quoted := make([]string, len(choices))
	for index, choice := range choices {
		quoted[index] = "'" + choice + "'"
	}
	message := "Input should be " + quoted[len(quoted)-1]
	if len(quoted) > 1 {
		message = "Input should be " + strings.Join(quoted[:len(quoted)-1], ", ") + " or " + quoted[len(quoted)-1]
	}
	return func(c *collector, at loc, input any) (any, bool) {
		if pyval.InTuple(input, choices...) {
			return input, true
		}
		c.add("literal_error", at, message)
		return nil, false
	}
}

// optional accepts None or what inner accepts (a nullable schema adds no location).
func optional(inner Validator) Validator {
	return func(c *collector, at loc, input any) (any, bool) {
		if input == nil {
			return nil, true
		}
		return inner(c, at, input)
	}
}

// laxInt converts an int field input in lax mode; the error kind is "" on success.
func laxInt(input any) (*big.Int, string) {
	if n, ok := pyval.BigInt(input); ok {
		return n, ""
	}
	switch x := input.(type) {
	case float64:
		return floatAsInt(x)
	case string:
		value, outcome := strAsInt(x)
		switch outcome {
		case intTooLarge:
			return nil, "int_parsing_size"
		case intInvalid:
			return nil, "int_parsing"
		}
		return value, ""
	}
	return nil, "int_type"
}

func floatAsInt(f float64) (*big.Int, string) {
	switch {
	case math.IsInf(f, 0) || math.IsNaN(f):
		return nil, "finite_number"
	case math.Mod(f, 1) != 0:
		return nil, "int_from_float"
	case -9.223372036854775808e18 < f && f < 9.223372036854775807e18:
		return big.NewInt(int64(f)), ""
	}
	return nil, "int_parsing_size"
}

var intMessages = map[string]string{
	"int_type":         "Input should be a valid integer",
	"int_parsing":      "Input should be a valid integer, unable to parse string as an integer",
	"int_parsing_size": "Unable to parse input string as an integer, exceeded maximum size",
	"int_from_float":   "Input should be a valid integer, got a number with a fractional part",
	"finite_number":    "Input should be a finite number",
}

// integer validates an int field, optionally limited to [ge, le].
func integer(limited bool, ge, le int64) Validator {
	return func(c *collector, at loc, input any) (any, bool) {
		value, kind := laxInt(input)
		if kind != "" {
			c.add(kind, at, intMessages[kind])
			return nil, false
		}
		if limited && value.Cmp(big.NewInt(le)) > 0 {
			c.add("less_than_equal", at, "Input should be less than or equal to "+strconv.FormatInt(le, 10))
			return nil, false
		}
		if limited && value.Cmp(big.NewInt(ge)) < 0 {
			c.add("greater_than_equal", at, "Input should be greater than or equal to "+strconv.FormatInt(ge, 10))
			return nil, false
		}
		return pyval.NormInt(value), true
	}
}

// coordinate validates a finite float in [-100000, 100000].
func coordinate(c *collector, at loc, input any) (any, bool) {
	var value float64
	switch x := input.(type) {
	case float64:
		value = x
	case string:
		parsed, ok := strAsFloat(x)
		if !ok {
			c.add("float_parsing", at, "Input should be a valid number, unable to parse string as a number")
			return nil, false
		}
		value = parsed
	default:
		n, ok := pyval.BigInt(input)
		converted, err := 0.0, error(nil)
		if ok {
			converted, err = pyval.IntToFloat(n)
		}
		if !ok || err != nil {
			c.add("float_type", at, "Input should be a valid number")
			return nil, false
		}
		value = converted
	}
	switch {
	case math.IsInf(value, 0) || math.IsNaN(value):
		c.add("finite_number", at, "Input should be a finite number")
	case value > 100000:
		c.add("less_than_equal", at, "Input should be less than or equal to 100000")
	case value < -100000:
		c.add("greater_than_equal", at, "Input should be greater than or equal to -100000")
	default:
		return value, true
	}
	return nil, false
}

// boolean validates a bool field in lax mode.
func boolean(c *collector, at loc, input any) (any, bool) {
	var number *big.Int
	switch x := input.(type) {
	case bool:
		return x, true
	case string:
		switch strings.ToLower(x) {
		case "0", "f", "n", "no", "off", "false":
			return false, true
		case "1", "t", "y", "on", "yes", "true":
			return true, true
		}
		c.add("bool_parsing", at, "Input should be a valid boolean, unable to interpret input")
		return nil, false
	case float64:
		converted, kind := floatAsInt(x)
		if kind != "" {
			c.add("bool_type", at, "Input should be a valid boolean")
			return nil, false
		}
		number = converted
	default:
		converted, ok := pyval.BigInt(input)
		if !ok || !converted.IsInt64() {
			c.add("bool_type", at, "Input should be a valid boolean")
			return nil, false
		}
		number = converted
	}
	if number.IsInt64() && (number.Int64() == 0 || number.Int64() == 1) {
		return number.Int64() == 1, true
	}
	c.add("bool_parsing", at, "Input should be a valid boolean, unable to interpret input")
	return nil, false
}
