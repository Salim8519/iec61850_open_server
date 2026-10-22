package service

import (
	"os"
	"strconv"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/worker"
)

func (m *Manager) setMessage(message string) {
	m.mu.Lock()
	m.message = message
	m.mu.Unlock()
	m.changed(ChangeGatewayStatus)
}

// prepare proposes points when the list is empty (Run stays the user's choice).
func (m *Manager) prepare() (*pyjson.Object, error) {
	config := m.Config()
	protocol := text(config, "protocol")
	if len(pointsOf(config, protocol)) == 0 {
		ieds := m.projects.IEDs()
		proposal, _, err := gateway.Propose(ieds, protocol)
		if err != nil {
			return nil, err
		}
		clean, _, err := gateway.ValidatePoints(protocol, proposal, ieds)
		if err != nil {
			return nil, err
		}
		points := copyObject(get(config, "points").(*pyjson.Object)).Set(protocol, clean)
		if err := m.projects.SaveGateway(with(config, "points", points)); err != nil {
			return nil, err
		}
	}
	return m.Config(), nil
}

// Autostart starts again after the app restarts if the gateway was running when it closed.
func (m *Manager) Autostart() {
	m.mu.Lock()
	idle := m.process == nil
	m.mu.Unlock()
	if pytext.Truthy(get(m.Config(), "auto_start")) && idle {
		if _, err := m.Start(); err != nil {
			m.mu.Lock()
			m.state, m.message = "failed", "Automatic start failed: "+err.Error()
			m.mu.Unlock()
			m.changed(ChangeGatewayStatus)
		}
	}
}

// Start validates the point list, starts studio_gateway and the value push.
func (m *Manager) Start() (*pyjson.Object, error) {
	defer m.changed(ChangeGatewayStatus)
	m.lifecycle.Lock()
	m.mu.Lock()
	running := m.process != nil
	m.mu.Unlock()
	if running {
		m.lifecycle.Unlock()
		return nil, apperr.ValueError("The gateway is already running; stop it first")
	}
	opened, protocol, count, err := m.launch()
	m.lifecycle.Unlock()
	if err != nil {
		return nil, err
	}
	if config := m.Config(); !pytext.Truthy(get(config, "auto_start")) {
		if err := m.projects.SaveGateway(with(config, "auto_start", true)); err != nil {
			return nil, err
		}
	}
	m.emit("gateway_started", pyjson.NewObject().Set("protocol", protocol).Set("port", get(opened, "port")).
		Set("points", int64(count)))
	return m.Status(), nil
}

func (m *Manager) launch() (*pyjson.Object, string, int, error) {
	if _, err := os.Stat(m.Executable); err != nil {
		return nil, "", 0, apperr.RuntimeError("The gateway program (studio_gateway.exe) is not built yet; rebuild the native programs")
	}
	role, found := m.Config().Get("role")
	if !found {
		role = "gateway"
	}
	if role == "master" {
		return nil, "", 0, apperr.ValueError("The tool is set to SCADA master; switch it to Station gateway first")
	}
	config := m.Config()
	if role == "gateway" {
		var err error
		if config, err = m.prepare(); err != nil {
			return nil, "", 0, err
		}
	}
	protocol := text(config, "protocol")
	var points []any
	var problems []string
	var err error
	if role == "converter" && m.Converter != nil {
		points, problems, err = m.Converter.Points(protocol)
	} else {
		points, problems, err = gateway.ValidatePoints(protocol, pointsOf(config, protocol), m.projects.IEDs())
	}
	if err != nil {
		return nil, "", 0, err
	}
	if len(points) == 0 {
		return nil, "", 0, apperr.ValueError("The point list is empty. Press 'Propose from IEDs' or import a CSV point list first.")
	}
	settingsByProtocol, _ := get(config, "settings").(*pyjson.Object)
	settings, err := gateway.ValidateSettings(protocol, get(settingsByProtocol, protocol))
	if err != nil {
		return nil, "", 0, err
	}
	bind, err := gateway.ListenAddress(get(config, "listen"))
	if err != nil {
		return nil, "", 0, err
	}
	settings = with(settings, "bind", bind)
	ieds := m.iedIndex()
	native, typed := make([]any, len(points)), make([]*pyjson.Object, len(points))
	for index, raw := range points {
		point := raw.(*pyjson.Object)
		typed[index] = point
		ied := ieds[text(point, "ied")]
		native[index] = nativePoint(point, m.MonitorOnly(point, config, ied))
	}
	process, err := m.spawn(m.Executable, m.event)
	if err != nil {
		return nil, "", 0, err
	}
	opened, err := m.open(process, protocol, settings, native)
	if err != nil {
		process.Close()
		return nil, "", 0, err
	}
	problemList := make([]any, len(problems))
	for index, problem := range problems {
		problemList[index] = problem
	}
	endpoint := pytext.Str(get(opened, "bind")) + ":" + pytext.Str(get(opened, "port"))
	m.own.reset(typed)
	sync := newSyncer(m, process, typed)
	m.mu.Lock()
	m.process, m.opened, m.problems, m.clients = process, opened, problemList, map[string]bool{}
	m.points, m.order = map[string]*pyjson.Object{}, nil
	for _, point := range typed {
		m.points[text(point, "id")] = point
		m.order = append(m.order, text(point, "id"))
	}
	m.state, m.message, m.sync = "running", "Listening on "+endpoint, sync
	m.mu.Unlock()
	m.Traffic.Add(pyjson.NewObject().Set("dir", "info").Set("peer", "").Set("text", "Gateway started: "+protocol+
		" on "+endpoint+" with "+strconv.Itoa(len(points))+" points"))
	sync.start()
	return opened, protocol, len(points), nil
}

func (m *Manager) open(process worker.Process, protocol string, settings *pyjson.Object, native []any) (*pyjson.Object, error) {
	hello, err := process.Request("hello", worker.DefaultTimeout, nil)
	if err != nil {
		return nil, err
	}
	for _, capability := range required {
		if !control.HasCapability(hello, capability) {
			return nil, apperr.RuntimeError("The gateway program is outdated; rebuild the native programs")
		}
	}
	typed := control.HasCapability(hello, typedCapability)
	if !typed && hasTypedPoints(native) {
		return nil, apperr.RuntimeError("The gateway program is too old for tap position, step and set point points; rebuild the native programs")
	}
	answer, err := process.Request("start", 15*time.Second, startRequest(protocol, settings, native, typed))
	if err != nil {
		return nil, err
	}
	opened, _ := answer.(*pyjson.Object)
	if opened == nil {
		opened = pyjson.NewObject()
	}
	return opened, nil
}
