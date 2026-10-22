package service

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/pytext"
)

// Typed command steps (ADR-0107): a select, an execute after a select, a cancel, and every
// command of the step and set point kinds. Each is logged in the Commands tab like the
// single and double commands, with what was asked, what the IED answered and why.

var (
	phaseWords   = map[Phase]string{PhaseSelect: " (select)", PhaseCancel: " (cancel)"}
	requestWords = map[Phase]string{PhaseSelect: "select received", PhaseExecute: "received",
		PhaseCancel: "cancel received", "": "received"}
)

// typed carries out one command step; refused is a reason found while decoding the request.
func (e *executor) typed(event, point *pyjson.Object, phase Phase, refused string) (bool, string, error) {
	m := e.manager
	id, kind := text(point, "id"), text(point, "kind")
	remote, own := get(point, "source") == "remote", gateway.IsOwn(point)
	var ied *pyjson.Object
	if !remote && !own {
		ied = m.IED(get(point, "ied"))
	}
	scale, _ := pytext.Float(getScale(point))
	value, words, reason := CommandValue(kind, get(event, "value"), scale, text(findControl(ied, get(point, "ref")), "cdc"))
	if refused != "" {
		reason = refused
	}
	label := commandName(point, get(event, "point_id")) + " " + words + phaseWords[phase]
	entry := m.Commands.Add(pyjson.NewObject().Set("point_id", get(event, "point_id")).Set("label", label).
		Set("protocol", get(event, "protocol")).Set("peer", get(event, "peer")).
		Set("request", requestWords[phase]).Set("applied", "").Set("observed", ""))
	seq := get(entry, "seq").(int64)
	switch {
	case reason != "":
		return e.refuse(seq, label, reason)
	case own:
		return e.ownTyped(point, phase, value, seq, label)
	case remote: // the converter relays executes only; its device has no select to keep
		return e.confirm(seq, label, string(phase)+" confirmed by ScadaBench; the device gets the execute")
	}
	if why := Refusal(point, ied, realCommands(m.Config())); why != "" {
		return e.refuse(seq, label, why)
	}
	wire, _ := get(event, "wire").(*pyjson.Object)
	request := TypedRequest{IEDID: text(ied, "id"), Ref: text(point, "ref"), Value: value,
		Test: pytext.Truthy(get(wire, "test")), OrCat: 3, OrIdent: "ScadaBench gateway " + text(event, "protocol")}
	if get(event, "protocol") == "manual" {
		request.OrCat, request.OrIdent = 2, "ScadaBench station"
	}
	switch Route(kind, phase, m.selected.has(id)) {
	case ActSelect:
		return e.selectIED(id, kind, request, seq, label)
	case ActCancel:
		return e.cancelIED(id, request, seq, label)
	}
	return e.operateIED(id, request, seq, label, ied, point)
}

func (e *executor) selectIED(id, kind string, request TypedRequest, seq int64, label string) (bool, string, error) {
	expires, err := e.typedPort().Select(request)
	if SelectFallback(kind, err) { // as before ADR-0107: the IED is selected by the execute itself
		return e.confirm(seq, label, "select confirmed by ScadaBench; the execute operates the IED")
	}
	if err != nil {
		return e.refuse(seq, label, "the IED refused the select: "+err.Error())
	}
	e.manager.selected.put(id, expires)
	return e.confirm(seq, label, "selected in the IED")
}

func (e *executor) cancelIED(id string, request TypedRequest, seq int64, label string) (bool, string, error) {
	if !e.manager.selected.take(id) {
		return e.confirm(seq, label, "cancel confirmed; nothing was selected in the IED")
	}
	if err := e.typedPort().Cancel(request.IEDID, request.Ref); err != nil && !NotAvailable(err) {
		return e.refuse(seq, label, "the IED refused the cancel: "+err.Error())
	}
	return e.confirm(seq, label, "selection cancelled in the IED")
}

func (e *executor) operateIED(id string, request TypedRequest, seq int64, label string, ied,
	point *pyjson.Object) (bool, string, error) {
	m := e.manager
	m.selected.take(id)
	result, err := e.typedPort().Operate(request)
	if err != nil {
		return e.refuse(seq, label, "the IED did not operate: "+err.Error())
	}
	summary := "The IED operated it."
	if _, has := get(result, "ok").(bool); has {
		summary = control.Summary(result)
		if !pytext.Truthy(get(result, "ok")) {
			return e.refuse(seq, label, "refused by the device: "+summary)
		}
	}
	m.Commands.Update(seq, pyjson.NewObject().Set("applied", "operated: "+summary).
		Set("observed", "see the IED value in Points"))
	m.emit("gateway_command", pyjson.NewObject().Set("point", label).Set("ied", get(ied, "name")).
		Set("role", get(ied, "role")).Set("ok", true).Set("result", summary))
	if flag, isBool := request.Value.(bool); isBool {
		controlObject := findControl(ied, get(point, "ref"))
		var expected any = flag
		if get(controlObject, "cdc") == "DPC" {
			expected = map[bool]int64{true: 2, false: 1}[flag]
		}
		if status, found := controlObject.Get("status"); found {
			go e.observe(seq, text(ied, "id"), pytext.Str(status), expected)
		}
	}
	return true, "", nil
}

// typedPort is the composition's typed commands, or one that says they are not available.
func (e *executor) typedPort() TypedCommands {
	if typed := e.manager.Typed; typed != nil {
		return typed
	}
	return unavailable{}
}

func (e *executor) refuse(seq int64, label, reason string) (bool, string, error) {
	e.manager.Commands.Update(seq, pyjson.NewObject().Set("applied", "refused: "+reason))
	e.manager.emit("gateway_command", pyjson.NewObject().Set("point", label).Set("result", "refused").
		Set("reason", reason))
	return false, reason, nil
}

// confirm logs a select or cancel that succeeded. Only executes and refusals send a
// gateway_command event, as before ADR-0107 (a select was confirmed without one).
func (e *executor) confirm(seq int64, label, applied string) (bool, string, error) {
	e.manager.Commands.Update(seq, pyjson.NewObject().Set("applied", applied))
	return true, "", nil
}

// commandName names a point in the Commands tab as the single/double path does.
func commandName(point *pyjson.Object, pointID any) string {
	var name any
	if get(point, "source") == "remote" || gateway.IsOwn(point) {
		name = get(point, "description")
	} else {
		name = get(point, "ref")
	}
	if !pytext.Truthy(name) {
		name = pointID
	}
	return pytext.Str(name)
}

func getScale(point *pyjson.Object) any {
	if scale, found := point.Get("scale"); found {
		return scale
	}
	return 1.0
}

type unavailable struct{}

func (unavailable) Select(TypedRequest) (int, error) { return 0, notAvailable() }
func (unavailable) Operate(TypedRequest) (*pyjson.Object, error) {
	return nil, notAvailable()
}
func (unavailable) Cancel(string, string) error { return notAvailable() }

func notAvailable() error {
	return apperr.Coded("typed IEC 61850 commands are not available in this engine", "not_available")
}
