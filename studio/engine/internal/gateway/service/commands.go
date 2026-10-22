package service

import (
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/worker"
)

type job struct {
	process worker.Process
	event   *pyjson.Object
}

// executor is CommandExecutor: one goroutine runs SCADA commands in arrival order.
type executor struct {
	manager *Manager
	jobs    chan job
}

func newExecutor(manager *Manager) *executor {
	e := &executor{manager: manager, jobs: make(chan job, 1024)}
	go e.loop()
	return e
}

func (e *executor) submit(process worker.Process, event *pyjson.Object) {
	select {
	case e.jobs <- job{process, event}:
	default:
		go func() { e.jobs <- job{process, event} }() // never block the worker's reader
	}
}

func (e *executor) loop() {
	for next := range e.jobs {
		ok, reason := e.run(next.event)
		if next.process != nil {
			next.process.Request("command_result", worker.DefaultTimeout, pyjson.NewObject().
				Set("token", get(next.event, "token")).Set("ok", ok).Set("reason", reason))
		}
	}
}

// run executes one command and never fails: (ok, plain reason).
func (e *executor) run(event *pyjson.Object) (ok bool, reason string) {
	defer func() {
		if failure := recover(); failure != nil {
			ok, reason = false, pytext.Str(failure)
		}
	}()
	ok, reason, err := e.execute(event)
	if err != nil {
		return false, err.Error()
	}
	return ok, reason
}

func (e *executor) execute(event *pyjson.Object) (bool, string, error) {
	m := e.manager
	pointID := get(event, "point_id")
	point := m.Point(pointID)
	wire, _ := get(event, "wire").(*pyjson.Object)
	phase, refused := PhaseOf(text(event, "protocol"), wire)
	if point != nil && (refused != "" || phase != PhaseExecute || gateway.IsTypedKind(text(point, "kind")) ||
		m.selected.has(text(point, "id"))) {
		return e.typed(event, point, phase, refused) // ADR-0107
	}
	value := pytext.Truthy(get(event, "value"))
	var name any
	if point != nil && (get(point, "source") == "remote" || gateway.IsOwn(point)) {
		name = get(point, "description")
	} else if point != nil {
		name = get(point, "ref")
	}
	if !pytext.Truthy(name) {
		name = pointID
	}
	label := pytext.Str(name) + " " + map[bool]string{true: "ON / close", false: "OFF / open"}[value]
	entry := m.Commands.Add(pyjson.NewObject().Set("point_id", pointID).Set("label", label).
		Set("protocol", get(event, "protocol")).Set("peer", get(event, "peer")).Set("request", "received").
		Set("applied", "").Set("observed", ""))
	seq := get(entry, "seq").(int64)
	config := m.Config()
	allowed := realCommands(config)
	if gateway.IsOwn(point) {
		return e.ownCommand(point, value, seq, label)
	}
	if point != nil && get(point, "source") == "remote" {
		ok, answer := m.Converter.Relay(point, value, allowed)
		verb := "refused"
		if ok {
			verb = "operated"
		}
		m.Commands.Update(seq, pyjson.NewObject().Set("applied", verb+" by the device: "+answer).
			Set("observed", "see the device value in Points"))
		m.emit("gateway_command", pyjson.NewObject().Set("point", label).Set("result", "relayed").Set("ok", ok).
			Set("reason", answer))
		if ok {
			return true, "", nil
		}
		return false, answer, nil
	}
	var ied *pyjson.Object
	if point != nil {
		ied = m.IED(get(point, "ied"))
	}
	if reason := Refusal(point, ied, allowed); reason != "" {
		m.Commands.Update(seq, pyjson.NewObject().Set("applied", "refused: "+reason))
		m.emit("gateway_command", pyjson.NewObject().Set("point", label).Set("result", "refused").Set("reason", reason))
		return false, reason, nil
	}
	host, port, err := e.endpoint(ied)
	if err != nil {
		return false, "", err
	}
	controlObject := findControl(ied, get(point, "ref"))
	originator := "remote"
	if get(event, "protocol") == "manual" {
		originator = "station"
	}
	result, err := m.control.Operate(host, port, text(point, "ref"), value,
		pyjson.NewObject().Set("originator", originator))
	if err != nil {
		return false, "", err
	}
	summary := control.Summary(result)
	m.emit("gateway_command", pyjson.NewObject().Set("point", label).Set("ied", get(ied, "name")).
		Set("role", get(ied, "role")).Set("ok", get(result, "ok")).Set("result", summary))
	if !pytext.Truthy(get(result, "ok")) {
		m.Commands.Update(seq, pyjson.NewObject().Set("applied", "refused by the device: "+summary))
		return false, summary, nil
	}
	m.Commands.Update(seq, pyjson.NewObject().Set("applied", "operated: "+summary))
	var expected any = value
	if get(controlObject, "cdc") == "DPC" {
		expected = map[bool]int64{true: 2, false: 1}[value]
	}
	status, found := controlObject.Get("status")
	if !found {
		return false, "", errorText("'status'") // Python's KeyError text after the command was sent
	}
	go e.observe(seq, text(ied, "id"), pytext.Str(status), expected)
	return true, "", nil
}

type errorText string

func (e errorText) Error() string { return string(e) }

func findControl(ied *pyjson.Object, ref any) *pyjson.Object {
	controls, _ := get(ied, "controls").([]any)
	// Typed controls (BSC, ISC, APC, INC; ADR-0104) live in the stored SCL, not in "controls".
	for _, raw := range append(controls, gateway.TypedControlsOf(ied)...) {
		if item, ok := raw.(*pyjson.Object); ok && get(item, "ref") == ref {
			return item
		}
	}
	return pyjson.NewObject()
}

// endpoint: the real device's address, or where the simulated IED listens.
func (e *executor) endpoint(ied *pyjson.Object) (string, any, error) {
	if get(ied, "role") == "real" {
		host, port := RealEndpoint(ied)
		return host, port, nil
	}
	return e.manager.runtime.Endpoint(text(ied, "id"))
}

// RealEndpoint is network_settings.real_endpoint: the user's override, else the SCD address.
func RealEndpoint(ied *pyjson.Object) (string, any) {
	live, _ := get(ied, "live").(*pyjson.Object)
	if !pytext.Truthy(live) {
		live = pyjson.NewObject().Set("ip", "").Set("port", int64(102))
	}
	address, _ := get(ied, "scl_address").(*pyjson.Object)
	scd, _ := get(address, "ip").(string)
	host, _ := get(live, "ip").(string)
	if host == "" {
		host = scd
	}
	return host, get(live, "port")
}

// Refusal is commands.refusal: why this SCADA command is not sent, or "" when it may be.
func Refusal(point, ied *pyjson.Object, realCommands bool) string {
	if point == nil {
		return "this point is not in the gateway point list"
	}
	if !gateway.IsCommandKind(text(point, "kind")) {
		return "this point is not a command"
	}
	if ied == nil {
		return "its IED is not in the project"
	}
	role, found := ied.Get("role")
	if !found {
		role = "simulated"
	}
	name := pytext.Str(get(ied, "name"))
	switch {
	case role == "off":
		return name + " is switched off in Devices"
	case role == "real" && !realCommands:
		return "commands to real IEDs are switched off in the gateway settings"
	case role == "real":
		if host, _ := RealEndpoint(ied); host == "" {
			return name + " has no address; type it in Devices"
		}
	case role == "simulated" && get(ied, "state") != "running":
		return name + " is not running; press Run first"
	}
	if get(findControl(ied, get(point, "ref")), "ref") == nil {
		return name + " has no controllable object " + pytext.Str(get(point, "ref"))
	}
	return ""
}

// PositionText is position_text: ON/OFF for a single point, closed/open for a position.
func PositionText(value any) string {
	if flag, ok := value.(bool); ok {
		return map[bool]string{true: "ON", false: "OFF"}[flag]
	}
	if key, ok := dictKey(value); ok {
		if word, found := map[int64]string{0: "intermediate", 1: "open (OFF)", 2: "closed (ON)",
			3: "bad state"}[key]; found {
			return word
		}
	}
	return pytext.Str(value)
}
