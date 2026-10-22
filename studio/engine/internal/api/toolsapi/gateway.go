package toolsapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/gateway/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/numlog"
)

// gatewayProtocol is api/gateway.protocol_arg.
func gatewayProtocol(value any) (string, error) {
	for _, protocol := range gateway.Protocols {
		if value == protocol {
			return protocol, nil
		}
	}
	return "", apperr.ValueError("Choose IEC 104 (iec104), Modbus TCP (modbus) or DNP3 (dnp3)")
}

func (a *Area) gatewayConfig() *pyjson.Object {
	current := a.deps.Gateway.Config()
	return pyjson.NewObject().Set("protocol", value(current, "protocol", nil)).Set("listen", value(current, "listen", nil)).
		Set("settings", value(current, "settings", nil)).Set("real_commands", value(current, "real_commands", true)).
		Set("auto_start", value(current, "auto_start", false))
}

func copyWith(source *pyjson.Object, key string, item any) *pyjson.Object {
	result := pyjson.NewObject()
	for _, name := range source.Keys() {
		found, _ := source.Get(name)
		result.Set(name, found)
	}
	return result.Set(key, item)
}

func (a *Area) gatewayRoutes() []api.Route {
	d := a.deps
	return []api.Route{
		route("GET", "/tools/gateway", true, func(*http.Request) (any, error) { return d.Gateway.Status(), nil }),
		route("GET", "/tools/gateway/config", false, func(*http.Request) (any, error) { return a.gatewayConfig(), nil }),
		route("PUT", "/tools/gateway/config", false, a.saveGatewayConfig),
		route("POST", "/tools/gateway/start", false, func(*http.Request) (any, error) { return d.Gateway.Start() }),
		route("POST", "/tools/gateway/stop", false, func(*http.Request) (any, error) { return d.Gateway.Stop(true) }),
		route("POST", "/tools/gateway/command", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			if value(data, "confirm", nil) != true {
				return nil, apperr.ValueError("Confirm the command first")
			}
			pointID, isText := value(data, "point_id", nil).(string)
			sent := value(data, "value", nil) // also "higher"/"lower" or a number (ADR-0107)
			kind := ""
			if point := d.Gateway.Point(pointID); isText && point != nil {
				kind, _ = value(point, "kind", nil).(string)
			}
			if !isText || !service.CommandInput(kind, sent) {
				return nil, apperr.ValueError("Send point_id and value (true = ON / close, false = OFF / open)")
			}
			return d.Gateway.Send(pointID, sent), nil
		}),
		route("GET", "/tools/gateway/traffic", true, func(r *http.Request) (any, error) { return after(r, d.Gateway.Traffic, 300) }),
		route("GET", "/tools/gateway/commands", true, func(r *http.Request) (any, error) { return after(r, d.Gateway.Commands, 200) }),
		route("POST", "/tools/gateway/role", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			role, err := gateway.ValidateRole(value(data, "role", nil))
			if err != nil {
				return nil, err
			}
			if d.Master.Busy() || d.Gateway.State() == "running" {
				return nil, apperr.ValueError("Stop the gateway and disconnect the master before changing this")
			}
			if err := d.Projects.SaveGateway(copyWith(d.Gateway.Config(), "role", role)); err != nil {
				return nil, err
			}
			return pyjson.NewObject().Set("role", role), nil
		}),
	}
}

func after(r *http.Request, log *numlog.Log, limit int) (any, error) {
	seq, err := afterArg(r)
	if err != nil {
		return nil, err
	}
	return log.After(seq, limit), nil
}

// saveGatewayConfig: protocol, listen choice, real-IED commands and settings (not points).
func (a *Area) saveGatewayConfig(r *http.Request) (any, error) {
	data, err := body(r)
	if err != nil {
		return nil, err
	}
	if a.deps.Gateway.State() == "running" {
		return nil, apperr.ValueError("Stop the gateway before changing its settings")
	}
	protocol, err := gatewayProtocol(value(data, "protocol", nil))
	if err != nil {
		return nil, err
	}
	listen := value(data, "listen", "this_pc")
	if _, err := gateway.ListenAddress(listen); err != nil {
		return nil, err
	}
	current := a.deps.Gateway.Config()
	checked, err := gateway.ValidateSettings(protocol, value(data, "settings", nil))
	if err != nil {
		return nil, err
	}
	stored, _ := value(current, "settings", nil).(*pyjson.Object)
	if stored == nil {
		stored = pyjson.NewObject()
	}
	settings := copyWith(stored, protocol, checked)
	realCommands := value(data, "real_commands", value(current, "real_commands", true))
	if _, isBool := realCommands.(bool); !isBool {
		return nil, apperr.ValueError("Commands to real IEDs are on or off")
	}
	saved := copyWith(copyWith(copyWith(copyWith(current, "protocol", protocol), "listen", listen), "settings", settings),
		"real_commands", realCommands)
	if err := a.deps.Projects.SaveGateway(saved); err != nil {
		return nil, err
	}
	return a.gatewayConfig(), nil
}
