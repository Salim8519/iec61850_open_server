package contract

import (
	"strconv"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// field is one model field: a required field has no fallback; an optional field without
// a fallback stays absent from the output when the input lacks it (see omittable).
type field struct {
	name     string
	check    Validator
	required bool
	fallback func() any
}

func required(name string, check Validator) field {
	return field{name: name, check: check, required: true}
}

func defaulted(name string, check Validator, fallback any) field {
	return field{name: name, check: check, fallback: func() any { return fallback }}
}

// omittable is dumped only when the input has it, so documents without it keep their exact
// bytes (measurements, ADR-0090).
func omittable(name string, check Validator) field {
	return field{name: name, check: check}
}

// model validates a pydantic BaseModel: fields in declaration order, then (extra="forbid")
// one error per unknown key in input order. The output holds every field.
func model(class string, forbid bool, fields ...field) Validator {
	return func(c *collector, at loc, input any) (any, bool) {
		object, isDict := input.(*pyjson.Object)
		if !isDict {
			c.add("model_type", at, "Input should be a valid dictionary or instance of "+class)
			return nil, false
		}
		return fieldsOf(c, at, object, forbid, fields)
	}
}

func fieldsOf(c *collector, at loc, object *pyjson.Object, forbid bool, fields []field) (any, bool) {
	result := pyjson.NewObject()
	ok := true
	known := map[string]bool{}
	for _, f := range fields {
		known[f.name] = true
		value, present := object.Get(f.name)
		if !present {
			if f.required {
				c.add("missing", at.with(f.name), "Field required")
				ok = false
			} else if f.fallback != nil {
				result.Set(f.name, f.fallback())
			}
			continue
		}
		checked, good := f.check(c, at.with(f.name), value)
		ok = ok && good
		result.Set(f.name, checked)
	}
	if forbid {
		for _, key := range object.Keys() {
			if !known[key] {
				c.add("extra_forbidden", at.with(key), "Extra inputs are not permitted")
				ok = false
			}
		}
	}
	return result, ok
}

// list validates a list with a maximum length; a longer list reports only its length.
func list(maxLength int, item Validator) Validator {
	return func(c *collector, at loc, input any) (any, bool) {
		items, isList := input.([]any)
		if !isList {
			c.add("list_type", at, "Input should be a valid list")
			return nil, false
		}
		if len(items) > maxLength {
			c.add("too_long", at, "List should have at most "+strconv.Itoa(maxLength)+
				" items after validation, not "+strconv.Itoa(len(items)))
			return nil, false
		}
		result := make([]any, 0, len(items))
		ok := true
		for index, value := range items {
			checked, good := item(c, at.with(index), value)
			ok = ok && good
			result = append(result, checked)
		}
		return result, ok
	}
}

const identifier = `^[A-Za-z0-9_-]{1,64}$`

var signalBinding = model("SignalBinding", true,
	required("ied_id", text(0, 0, "")),
	required("ref", text(0, 0, "")),
	defaulted("report_ref", optional(text(0, 0, "")), nil))

var controlBinding = model("ControlBinding", true,
	required("ied_id", text(0, 0, "")),
	required("ref", text(0, 0, "")))

var relayStatus = model("RelayStatus", true,
	required("id", text(0, 0, identifier)),
	required("label", text(1, 100, "")),
	required("source", signalBinding))

// measurement is one row of a measurement list (ADR-0090): shown = raw * scale + offset,
// rounded to decimals and followed by unit.
var measurement = model("Measurement", true,
	required("id", text(0, 0, identifier)),
	required("label", text(1, 100, "")),
	required("source", signalBinding),
	defaulted("scale", bounded(1e9), 1.0),
	defaulted("offset", bounded(1e9), 0.0),
	defaulted("unit", text(0, 16, ""), ""),
	defaulted("decimals", integer(true, 0, 6), int64(2)))

var position = model("Position", false,
	required("x", coordinate),
	required("y", coordinate))

var mappings = map[string][]field{
	"direct":          {required("mode", literal("direct")), required("source", signalBinding)},
	"single_to_dbpos": {required("mode", literal("single_to_dbpos")), required("source", signalBinding)},
	"contacts_to_dbpos": {required("mode", literal("contacts_to_dbpos")),
		required("open", signalBinding), required("closed", signalBinding)},
}

// stateMapping is the discriminated union on "mode"; members forbid extra keys.
func stateMapping(c *collector, at loc, input any) (any, bool) {
	object, isDict := input.(*pyjson.Object)
	if !isDict {
		c.add("model_attributes_type", at, "Input should be a valid dictionary or object to extract fields from")
		return nil, false
	}
	tag, present := object.Get("mode")
	if !present {
		c.add("union_tag_not_found", at, "Unable to extract tag using discriminator 'mode'")
		return nil, false
	}
	name, isText := tag.(string)
	fields, known := mappings[name]
	if !isText || !known {
		c.add("union_tag_invalid", at, "Input tag '"+pyval.Str(tag)+
			"' found using 'mode' does not match any of the expected tags: 'direct', 'single_to_dbpos', 'contacts_to_dbpos'")
		return nil, false
	}
	return fieldsOf(c, at.with(name), object, true, fields)
}
