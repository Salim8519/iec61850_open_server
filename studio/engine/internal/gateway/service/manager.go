package service

import (
	"os"
	"sort"
	"strings"
	"sync"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/numlog"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/worker"
)

// Required capabilities of studio_gateway.
var required = []string{"gateway_iec104", "gateway_modbus", "gateway_dnp3", "commands_via_backend"}

// Manager is GatewayManager: the gateway worker, its point values and its logs.
type Manager struct {
	projects   Projects
	runtime    Runtime
	control    control.Operator
	Executable string
	spawn      worker.Spawner
	Traffic    *numlog.Log
	Commands   *numlog.Log
	Converter  *Converter // protocol converter role; set by the composition
	Typed      TypedCommands // typed IEC 61850 commands (ADR-0104/0107); set by the composition
	changes               // OnChange: what the push stream re-reads

	lifecycle sync.Mutex // Start/Stop one at a time (Python's RLock around them)
	mu        sync.Mutex // the fields below; never held across a worker request
	process   worker.Process
	points    map[string]*pyjson.Object
	order     []string
	state     string
	message   string
	opened    *pyjson.Object
	problems  []any
	clients   map[string]bool
	sync      *syncer
	executor  *executor
	own       ownStore // values of the own points the running gateway serves (lane G)
	selected  selections // IED selections a SCADA select made, until execute or cancel (ADR-0107)
}

// NewManager returns a stopped gateway for studio_gateway at executable.
func NewManager(projects Projects, runtime Runtime, operator control.Operator, executable string,
	spawn worker.Spawner) *Manager {
	if spawn == nil {
		spawn = worker.Spawn
	}
	m := &Manager{projects: projects, runtime: runtime, control: operator, Executable: executable, spawn: spawn,
		Traffic: numlog.New(numlog.TrafficLines), Commands: numlog.New(numlog.CommandLines), state: "stopped",
		points: map[string]*pyjson.Object{}, clients: map[string]bool{}, problems: []any{}}
	m.executor = newExecutor(m)
	m.Traffic.Watch(func() { m.changed(ChangeGatewayTraffic); m.changed(ChangeGatewayStatus) }) // traffic_last
	m.Commands.Watch(func() { m.changed(ChangeGatewayCommands) })
	return m
}

func (m *Manager) emit(kind string, payload *pyjson.Object) { m.runtime.Emit(kind, payload) }

// Config is the stored gateway section, or the defaults.
func (m *Manager) Config() *pyjson.Object {
	if config := m.projects.Gateway(); config != nil {
		return config
	}
	return gateway.EmptyGateway()
}

// IED returns the project IED with id, or nil.
func (m *Manager) IED(id any) *pyjson.Object {
	for _, ied := range m.projects.IEDs() {
		if get(ied, "id") == id {
			return ied
		}
	}
	return nil
}

func (m *Manager) iedIndex() map[string]*pyjson.Object {
	index := map[string]*pyjson.Object{}
	for _, ied := range m.projects.IEDs() {
		if id, ok := get(ied, "id").(string); ok {
			if _, seen := index[id]; !seen {
				index[id] = ied
			}
		}
	}
	return index
}

// Point is a point of the running gateway, else of the saved list (manual commands work while stopped).
func (m *Manager) Point(id any) *pyjson.Object {
	m.mu.Lock()
	key, _ := id.(string)
	point, found := m.points[key]
	m.mu.Unlock()
	if found {
		return point
	}
	config := m.Config()
	for _, raw := range pointsOf(config, text(config, "protocol")) {
		if candidate, ok := raw.(*pyjson.Object); ok && get(candidate, "id") == id {
			return candidate
		}
	}
	return nil
}

func pointsOf(config *pyjson.Object, protocol string) []any {
	byProtocol, _ := get(config, "points").(*pyjson.Object)
	points, _ := get(byProtocol, protocol).([]any)
	return points
}

func realCommands(config *pyjson.Object) bool {
	value, found := config.Get("real_commands")
	return !found || pytext.Truthy(value)
}

// MonitorOnly is _monitor_only: no commands to an IED that is off, or real while real commands are off.
func (m *Manager) MonitorOnly(point, config *pyjson.Object, ied *pyjson.Object) bool {
	if get(point, "source") == "remote" {
		return !realCommands(config)
	}
	if gateway.IsOwn(point) { // an own point is a simulation: its commands are always taken
		return false
	}
	role := roleOf(ied)
	return role == "off" || (role == "real" && !realCommands(config))
}

// roleOf is (ied or {}).get("role", "off").
func roleOf(ied *pyjson.Object) any {
	if ied != nil {
		if role, found := ied.Get("role"); found {
			return role
		}
	}
	return "off"
}

// State is the gateway state word (running, stopped, failed).
func (m *Manager) State() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Status is the page's view of the gateway.
func (m *Manager) Status() *pyjson.Object {
	config := m.Config()
	m.mu.Lock()
	var opened any
	if m.process != nil && m.opened != nil {
		opened = m.opened
	}
	clients := make([]string, 0, len(m.clients))
	for peer := range m.clients {
		clients = append(clients, peer)
	}
	state, message, problems := m.state, m.message, m.problems
	m.mu.Unlock()
	sort.Strings(clients)
	peers := make([]any, len(clients))
	for index, peer := range clients {
		peers[index] = peer
	}
	_, err := os.Stat(m.Executable)
	autoStart, found := config.Get("auto_start")
	if !found {
		autoStart = false
	}
	role, found := config.Get("role")
	if !found {
		role = "gateway"
	}
	realValue, found := config.Get("real_commands")
	if !found {
		realValue = true
	}
	return pyjson.NewObject().Set("state", state).Set("message", message).Set("opened", opened).
		Set("protocol", get(config, "protocol")).Set("listen", get(config, "listen")).Set("clients", peers).
		Set("problems", problems).Set("run_state", m.runtime.RunState()).Set("available", err == nil).
		Set("traffic_last", m.Traffic.Sequence()).Set("real_commands", realValue).Set("auto_start", autoStart).
		Set("role", role)
}

// Table is the saved point list of one protocol with what each point carries now.
func (m *Manager) Table(protocol string) ([]any, error) {
	config := m.Config()
	if get(config, "role") == "converter" && m.Converter != nil {
		return m.Converter.Table(protocol)
	}
	runState := m.runtime.RunState()
	ieds := m.iedIndex()
	rows := []any{}
	for _, raw := range pointsOf(config, protocol) {
		point := raw.(*pyjson.Object)
		if gateway.IsOwn(point) {
			rows = append(rows, m.ownRow(point))
			continue
		}
		iedID, _ := get(point, "ied").(string)
		ied := ieds[iedID]
		name := any("?")
		if ied != nil {
			name = get(ied, "name")
		}
		row := copyObject(point).Set("ied_name", name).Set("ied_role", roleOf(ied)).
			Set("monitor_only", m.MonitorOnly(point, config, ied))
		if kind, _ := get(point, "kind").(string); !strings.HasPrefix(kind, "command_") {
			current := SampleOf(text(point, "id"), iedID, text(point, "ref"), ied, runState, m.runtime)
			row.Set("value_text", Display(point, current)).Set("online", current.SourceOK)
		}
		rows = append(rows, row)
	}
	return rows, nil
}
