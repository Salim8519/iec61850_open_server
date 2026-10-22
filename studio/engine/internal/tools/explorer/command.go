package explorer

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/typedcmd"
)

// CommandRoute is the Go-only Explorer route of typed commands (ADR-0104).
const CommandRoute = "POST /api/v1/tools/explorer/command"

var typedOps = map[string]string{"select": "typed_select", "operate": "typed_operate", "cancel": "typed_cancel"}

// Command is one typed step on the connected device (ADR-0104): select, operate or cancel,
// with a value of kind bool, int, enum, float or step and the command options. Select and
// Operate need the same small confirmation as Operate; Cancel never does. The answer is
// {result, summary}; a refusal is in result (ok false, error, add_cause_text), not an error.
func (s *Service) Command(action, reference, kind string, value, options any, confirmed bool) (*pyjson.Object, error) {
	op, known := typedOps[action]
	if !known {
		return nil, apperr.ValueError("Action must be select, operate or cancel")
	}
	if action != "cancel" && !confirmed {
		return nil, apperr.ValueError("Confirm the command first")
	}
	operator, typed := s.control.(control.TypedOperator)
	if s.control == nil || !typed {
		return nil, apperr.RuntimeError("Commands are not available in this copy of ScadaBench")
	}
	settings, err := typedcmd.ParseOptions(options)
	if err != nil {
		return nil, err
	}
	if settings.OrCat == 0 {
		settings.OrCat = typedcmd.DefaultOrCat
	}
	s.mu.Lock()
	state, host, port := s.state, s.host, s.port
	s.mu.Unlock()
	if state != "connected" || port == nil {
		return nil, apperr.ValueError("Connect to a device first")
	}
	fields := settings.Fields(pyjson.NewObject().Set("ip", host).Set("port", port).Set("ref", reference))
	if action == "operate" || value != nil {
		// Explorer has no SCL: the value is checked for its kind; the device checks its limits.
		check := kind
		if check == "enum" {
			check = "int"
		}
		_, wire, err := typedcmd.Wire(typedcmd.Control{Ref: reference, ValueKind: check}, value, nil)
		if err != nil {
			return nil, err
		}
		fields.Set("value", wire).Set("kind", kind)
	}
	if action == "operate" {
		fields.Set("require_select", false)
	}
	result, err := operator.Typed(op, fields)
	if err != nil {
		return nil, err
	}
	summary := typedSummary(action, result)
	ok, _ := result.Get("ok")
	s.emit("explorer_command", pyjson.NewObject().Set("host", host).Set("ref", reference).Set("value", value).
		Set("action", action).Set("ok", ok).Set("result", summary))
	return pyjson.NewObject().Set("result", result).Set("summary", summary), nil
}

func typedSummary(action string, result *pyjson.Object) string {
	ok, _ := result.Get("ok")
	if ok != true {
		return control.Summary(result)
	}
	switch action {
	case "select":
		if selected, _ := result.Get("selected"); selected != true {
			return "This control operates directly; no select is needed."
		}
		return "Selected. Send Operate before the time runs out, or Cancel."
	case "cancel":
		if canceled, _ := result.Get("canceled"); canceled == true {
			return "The selection was canceled."
		}
		return "Nothing was selected, so there was nothing to cancel."
	}
	return control.Summary(result)
}
