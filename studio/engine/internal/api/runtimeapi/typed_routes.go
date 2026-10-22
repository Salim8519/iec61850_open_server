package runtimeapi

import (
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/typedcmd"
)

// Go-only typed command routes (ADR-0104, docs/reference/API.md "Typed commands"). They act on
// simulated IEDs of the running project only, exactly like POST /controls/operate.

// GoOnlyRoutes are the routes the frozen Python app does not have: typed commands (ADR-0104) and
// select before operate for a breaker, disconnector or earth switch (ADR-0108).
var GoOnlyRoutes = map[string]bool{"POST /api/v1/controls/select": true, "POST /api/v1/controls/cancel": true,
	"POST /api/v1/controls/checks": true, "GET /api/v1/controls/typed": true,
	"POST /api/v1/equipment/{node_id}/select": true, "POST /api/v1/equipment/{node_id}/cancel": true} // ADR-0108

// typedRequest reports whether a /controls/operate body asks for the typed path: command
// options, or a value that is not on/off. {ied_id, ref, value: bool} keeps the old path.
func typedRequest(data any) bool {
	object, ok := data.(*pyjson.Object)
	if !ok {
		return false
	}
	if _, present := object.Get("options"); present {
		return true
	}
	value, present := object.Get("value")
	_, isBool := value.(bool)
	return present && value != nil && !isBool
}

func typedCommand(data any) (runtime.TypedCommand, error) {
	object, ok := data.(*pyjson.Object)
	if !ok {
		return runtime.TypedCommand{}, apperr.ValueError("Send a JSON object with ied_id and ref")
	}
	iedID, okID := pyval.Get(object, "ied_id").(string)
	ref, okRef := pyval.Get(object, "ref").(string)
	if !okID || !okRef || iedID == "" || ref == "" {
		return runtime.TypedCommand{}, apperr.ValueError("Send the IED (ied_id) and the control reference (ref)")
	}
	command := runtime.TypedCommand{IEDID: iedID, Ref: ref, Value: pyval.Get(object, "value")}
	options, err := typedcmd.ParseOptions(pyval.Get(object, "options"))
	command.Options = options
	return command, err
}

// typedOperate is POST /controls/operate on the typed path: the device's answer.
func (a Area) typedOperate(data any) (any, int, error) {
	command, err := typedCommand(data)
	if err != nil {
		return nil, 0, err
	}
	result, err := a.Runtime.OperateTyped(command)
	if err != nil {
		return nil, 0, err
	}
	return result, http.StatusOK, nil
}

// controlSelect is POST /controls/select: {ok, model, expires_ms}.
func (a Area) controlSelect(r *http.Request) (any, int, error) {
	data, err := getJSON(r)
	if err != nil {
		return nil, 0, err
	}
	command, err := typedCommand(data)
	if err != nil {
		return nil, 0, err
	}
	result, err := a.Runtime.SelectTyped(command)
	if err != nil {
		return nil, 0, err
	}
	return pyval.Obj("ok", true, "model", result.Model, "expires_ms", int64(result.ExpiresMs)), http.StatusOK, nil
}

// controlCancel is POST /controls/cancel {ied_id, ref}: {ok}.
func (a Area) controlCancel(r *http.Request) (any, int, error) {
	data, err := getJSON(r)
	if err != nil {
		return nil, 0, err
	}
	command, err := typedCommand(data)
	if err != nil {
		return nil, 0, err
	}
	if err := a.Runtime.CancelTyped(command.IEDID, command.Ref); err != nil {
		return nil, 0, err
	}
	return pyval.Obj("ok", true), http.StatusOK, nil
}

// controlChecks is POST /controls/checks {ied_id, ref, blocked?, interlock_blocked?, synchro_blocked?}.
func (a Area) controlChecks(r *http.Request) (any, int, error) {
	data, err := getJSON(r)
	if err != nil {
		return nil, 0, err
	}
	command, err := typedCommand(data)
	if err != nil {
		return nil, 0, err
	}
	if err := a.Runtime.SetControlChecks(command.IEDID, command.Ref, data.(*pyjson.Object)); err != nil {
		return nil, 0, err
	}
	return pyval.Obj("ok", true), http.StatusOK, nil
}

// controlList is GET /controls/typed?ied_id=: {controls, extended_controls}.
func (a Area) controlList(r *http.Request) (any, int, error) {
	result, err := a.Runtime.TypedControls(r.URL.Query().Get("ied_id"))
	if err != nil {
		return nil, 0, err
	}
	return result, http.StatusOK, nil
}
