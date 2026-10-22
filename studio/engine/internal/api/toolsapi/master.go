package toolsapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// commandKinds are the command types each master protocol can send.
// commandKinds is the frozen Python baseline KINDS: it sets the sorted list named in the
// "not available" message, so the Python parity replay keeps matching (ADR-0076).
var commandKinds = map[string][]string{
	"iec104": {"single", "double", "setpoint_float", "setpoint_scaled", "setpoint_normalized", "clock"},
	"dnp3":   {"binary_output", "analog_output_int", "analog_output_float"},
	"modbus": {"coil", "register", "registers_int32", "registers_float"},
}

// goOnlyKinds are the command types added in the Go engine only (change 0169). They are
// accepted, but never listed in the frozen refusal message above.
var goOnlyKinds = map[string][]string{
	"iec104": {"regulating"},
	"dnp3":   {"analog_output_int16"},
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
		route("GET", "/tools/master/cyclic", false, func(*http.Request) (any, error) { return m.CyclicState(), nil }),
		route("POST", "/tools/master/cyclic", false, a.masterCyclic),
		route("POST", "/tools/master/cyclic/stop", false, func(*http.Request) (any, error) { return m.StopCyclic(), nil }),
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

// masterCommand checks a confirmed command against the protocol's command types and sends it once.
func (a *Area) masterCommand(r *http.Request) (any, error) {
	data, err := body(r)
	if err != nil {
		return nil, err
	}
	command, err := a.buildCommand(data)
	if err != nil {
		return nil, err
	}
	return a.deps.Master.Command(command, "You \xc2\xb7 this page")
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
