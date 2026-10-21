package toolsapi

import (
	"net/http"
	"sort"
	"strings"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// commandKinds are the command types each master protocol can send.
var commandKinds = map[string][]string{
	"iec104": {"single", "double", "setpoint_float", "setpoint_scaled", "setpoint_normalized", "clock"},
	"dnp3":   {"binary_output", "analog_output_int", "analog_output_float"},
	"modbus": {"coil", "register", "registers_int32", "registers_float"},
}

func (a *Area) masterConfig() *pyjson.Object {
	current := a.deps.Master.Config()
	role := value(a.deps.Gateway.Config(), "role", "gateway")
	result := pyjson.NewObject().Set("role", role)
	for _, key := range []string{"protocol", "host", "settings", "modbus_ranges", "routes", "auto_connect"} {
		result.Set(key, value(current, key, nil))
	}
	return result
}

func (a *Area) masterRoutes() []api.Route {
	m := a.deps.Master
	return []api.Route{
		route("GET", "/tools/master", true, func(*http.Request) (any, error) { return m.Status(), nil }),
		route("GET", "/tools/master/config", false, func(*http.Request) (any, error) { return a.masterConfig(), nil }),
		route("PUT", "/tools/master/config", false, a.saveMasterConfig),
		route("POST", "/tools/master/connect", false, func(*http.Request) (any, error) { return m.Connect() }),
		route("POST", "/tools/master/disconnect", false, func(*http.Request) (any, error) { return m.Disconnect(true) }),
		route("POST", "/tools/master/interrogate", false, func(*http.Request) (any, error) { return m.Interrogate() }),
		route("POST", "/tools/master/command", false, a.masterCommand),
		route("GET", "/tools/master/traffic", true, func(r *http.Request) (any, error) { return after(r, m.Traffic, 300) }),
		route("GET", "/tools/master/commands", true, func(r *http.Request) (any, error) { return after(r, m.Commands, 200) }),
	}
}

func (a *Area) saveMasterConfig(r *http.Request) (any, error) {
	data, err := body(r)
	if err != nil {
		return nil, err
	}
	m := a.deps.Master
	if m.Busy() {
		return nil, apperr.ValueError("Disconnect before changing the connection settings")
	}
	current := m.Config()
	protocol, err := gateway.ProtocolArg(value(data, "protocol", value(current, "protocol", nil)))
	if err != nil {
		return nil, err
	}
	host := ""
	if raw := value(data, "host", nil); truthy(raw) {
		if host, err = gateway.ValidateHost(raw); err != nil {
			return nil, err
		}
	}
	checked, err := gateway.ValidateMasterSettings(protocol, value(data, "settings", nil))
	if err != nil {
		return nil, err
	}
	stored, _ := value(current, "settings", nil).(*pyjson.Object)
	ranges, err := gateway.ValidateRanges(value(data, "modbus_ranges", value(current, "modbus_ranges", nil)))
	if err != nil {
		return nil, err
	}
	saved := copyWith(copyWith(copyWith(copyWith(current, "protocol", protocol), "host", host), "settings",
		copyWith(orObject(stored), protocol, checked)), "modbus_ranges", ranges)
	if err := m.Save(saved); err != nil {
		return nil, err
	}
	return a.masterConfig(), nil
}

// masterCommand checks a confirmed command against the protocol's command types.
func (a *Area) masterCommand(r *http.Request) (any, error) {
	data, err := body(r)
	if err != nil {
		return nil, err
	}
	if value(data, "confirm", nil) != true {
		return nil, apperr.ValueError("Confirm the command first")
	}
	m := a.deps.Master
	kind := value(data, "kind", nil)
	protocol := pytext.Str(value(m.Config(), "protocol", nil))
	allowed := commandKinds[protocol]
	name, isText := kind.(string)
	if !isText || !contains(allowed, name) {
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
		return m.Command(pyjson.NewObject().Set("kind", name), "You \xc2\xb7 this page")
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
	}
	return m.Command(command, "You \xc2\xb7 this page")
}

type pythonError string

func (e pythonError) Error() string { return string(e) }

var errUnhashable = pythonError("TypeError: unhashable type")

func contains(items []string, name string) bool {
	for _, item := range items {
		if item == name {
			return true
		}
	}
	return false
}
