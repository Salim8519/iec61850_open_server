package toolsapi

import (
	"net/http"
	"sort"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// masterCyclic validates a confirmed command and starts the cyclic sender for it. It takes the
// same command fields as the one-shot route plus interval_ms, address_step and toggle.
func (a *Area) masterCyclic(r *http.Request) (any, error) {
	data, err := body(r)
	if err != nil {
		return nil, err
	}
	command, err := a.buildCommand(data)
	if err != nil {
		return nil, err
	}
	if value(command, "kind", nil) == "clock" {
		return nil, apperr.ValueError("Clock synchronisation cannot be sent on a cycle")
	}
	interval, ok := pytext.Int64(value(data, "interval_ms", int64(1000)))
	if !ok {
		return nil, apperr.ValueError("The cycle time is milliseconds from 100 to 600000")
	}
	step, ok := pytext.Int64(value(data, "address_step", int64(0)))
	if !ok {
		return nil, apperr.ValueError("The address step is a whole number from 0 to 65535")
	}
	return a.deps.Master.StartCyclic(command, interval, step, value(data, "toggle", nil) == true)
}

// buildCommand validates one confirmed command for the current protocol and returns the
// request for the worker. It is shared by the one-shot command route and the cyclic sender,
// so both apply exactly the same checks before anything reaches a real device.
func (a *Area) buildCommand(data *pyjson.Object) (*pyjson.Object, error) {
	if value(data, "confirm", nil) != true {
		return nil, apperr.ValueError("Confirm the command first")
	}
	kind := value(data, "kind", nil)
	protocol := pytext.Str(value(a.deps.Master.Config(), "protocol", nil))
	allowed := commandKinds[protocol]
	name, isText := kind.(string)
	if !isText || !(contains(allowed, name) || contains(goOnlyKinds[protocol], name)) {
		if _, unhashable := kind.([]any); unhashable {
			return nil, errUnhashable // `kind not in set` hashes the value: TypeError, HTTP 500
		}
		if _, unhashable := kind.(*pyjson.Object); unhashable {
			return nil, errUnhashable
		}
		sorted := append([]string(nil), allowed...)
		sort.Strings(sorted)
		return nil, apperr.ValueError("This command type is not available for " + protocol + ": choose one of " +
			strings.Join(sorted, ", "))
	}
	if name == "clock" {
		return pyjson.NewObject().Set("kind", name), nil
	}
	address, number := value(data, "address", nil), value(data, "value", nil)
	limit := int64(65535)
	if protocol == "iec104" {
		limit = 16777215
	}
	if !pytext.IntIn(address, 0, limit) {
		return nil, apperr.ValueError("The address must be a whole number")
	}
	if flag, isBool := number.(bool); isBool && name == "single" {
		number = map[bool]int64{true: 1, false: 0}[flag]
	}
	if !pytext.IsNumber(number) {
		return nil, apperr.ValueError("The value must be a number")
	}
	command := pyjson.NewObject().Set("kind", name).Set("address", address).Set("value", number).
		Set("select", value(data, "select", nil) == true)
	return a.commandFields(name, protocol, data, command)
}

// commandFields adds the per-protocol extras (Modbus word order, DNP3 CROB code and timing,
// IEC 104 qualifier and deactivate) to an already value-checked command.
func (a *Area) commandFields(name, protocol string, data, command *pyjson.Object) (*pyjson.Object, error) {
	if strings.HasPrefix(name, "registers_") {
		order := "big"
		if value(data, "word_order", nil) == "little" {
			order = "little"
		}
		command.Set("word_order", order)
	}
	if name == "binary_output" {
		code := value(data, "code", "latch")
		if code != "latch" && code != "pulse" && code != "trip_close" {
			if _, unhashable := code.([]any); unhashable {
				return nil, errUnhashable
			}
			return nil, apperr.ValueError("A CROB is latch, pulse or trip_close")
		}
		command.Set("code", code)
		if err := crobTiming(data, command); err != nil {
			return nil, err
		}
	}
	if protocol == "iec104" {
		if err := iec104Qualifier(data, command); err != nil {
			return nil, err
		}
	}
	return command, nil
}

// crobTiming threads the optional DNP3 CROB count and on/off pulse times (Go-only addition).
func crobTiming(data, command *pyjson.Object) error {
	limits := []struct {
		key  string
		low  int64
		high int64
		text string
	}{{"count", 1, 255, "The CROB count is a whole number from 1 to 255"},
		{"on_time_ms", 0, 3600000, "The on time is milliseconds from 0 to 3600000"},
		{"off_time_ms", 0, 3600000, "The off time is milliseconds from 0 to 3600000"}}
	for _, field := range limits {
		if raw := value(data, field.key, nil); raw != nil {
			if !pytext.IntIn(raw, field.low, field.high) {
				return apperr.ValueError(field.text)
			}
			command.Set(field.key, raw)
		}
	}
	return nil
}

// iec104Qualifier threads the optional QU qualifier (0-31) and the deactivate flag (COT 8).
func iec104Qualifier(data, command *pyjson.Object) error {
	if raw := value(data, "qualifier", nil); raw != nil {
		if !pytext.IntIn(raw, 0, 31) {
			return apperr.ValueError("The command qualifier (QU) is a whole number from 0 to 31")
		}
		command.Set("qualifier", raw)
	}
	if value(data, "deactivate", nil) == true {
		command.Set("deactivate", true)
	}
	return nil
}
