package service

import (
	"os"
	"sync"

	"studio-engine/internal/gateway"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/numlog"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/worker"
)

// RetrySeconds is how often a dropped link is tried again.
const RetrySeconds = 10

var capability = map[string]string{"iec104": "master_iec104", "dnp3": "master_dnp3", "modbus": "master_modbus"}
var protocolNames = map[string]string{"iec104": "IEC 104", "dnp3": "DNP3", "modbus": "Modbus TCP"}

// askWords is what "asking for every value" is called in each protocol.
var askWords = map[string]string{"iec104": "General interrogation", "dnp3": "Integrity poll", "modbus": "Reading the ranges"}

// Master is MasterManager: ScadaBench as SCADA master toward a gateway, RTU or outstation.
type Master struct {
	projects   Projects
	emit       func(kind string, payload *pyjson.Object)
	Executable string
	spawn      worker.Spawner
	Traffic    *numlog.Log
	Commands   *numlog.Log
	Probe      *Probe
	cyclic     *cyclicSender
	// OnValues runs after device values were merged (the converter pushes them upstream).
	OnValues func()
	// Adapters lists this PC's adapters for the master's "Network" choice (ADR-0123).
	Adapters func() ([]netcfg.Adapter, error)
	changes  // OnChange: what the push stream re-reads

	lifecycle sync.Mutex // connect/disconnect
	mu        sync.Mutex // the fields below; never held across a worker request
	process   worker.Process
	state     string
	message   string
	linkOpen  bool
	lastAsked any // time.time() of the last interrogation, or None
	failures  []any
	stopping  chan struct{}

	tableMu sync.Mutex // the discovered table; values arrive while requests run
	table   *pyjson.Object
	device  []any // (protocol, host, port) the table belongs to; nil before the first connect
}

// NewMaster returns a disconnected master for studio_master at executable.
func NewMaster(projects Projects, emit func(string, *pyjson.Object), executable string, spawn worker.Spawner) *Master {
	if spawn == nil {
		spawn = worker.Spawn
	}
	if emit == nil {
		emit = func(string, *pyjson.Object) {}
	}
	m := &Master{projects: projects, emit: emit, Executable: executable, spawn: spawn, state: "disconnected",
		Traffic: numlog.New(numlog.TrafficLines), Commands: numlog.New(numlog.CommandLines), Probe: &Probe{},
		failures: []any{}, table: pyjson.NewObject(), stopping: make(chan struct{})}
	m.cyclic = newCyclic(m)
	m.Traffic.Watch(func() { m.changed(ChangeMasterTraffic); m.changed(ChangeMasterStatus) }) // traffic_last
	m.Commands.Watch(func() { m.changed(ChangeMasterCommands) })
	return m
}

// StartCyclic repeats a command on a timer (address step and value toggle optional).
func (m *Master) StartCyclic(command *pyjson.Object, interval, step int64, toggle bool) (*pyjson.Object, error) {
	return m.cyclic.Start(command, interval, step, toggle)
}

// StopCyclic ends the running cycle.
func (m *Master) StopCyclic() *pyjson.Object {
	m.cyclic.Stop()
	return m.cyclic.Snapshot()
}

// CyclicState is the cyclic sender's current state for the page (Go-only route).
func (m *Master) CyclicState() *pyjson.Object { return m.cyclic.Snapshot() }

func (m *Master) gatewaySection() *pyjson.Object {
	if section := m.projects.Gateway(); section != nil {
		return section
	}
	return gateway.EmptyGateway()
}

// Config is the stored master block, or the defaults.
func (m *Master) Config() *pyjson.Object {
	if master, ok := get(m.projects.Gateway(), "master").(*pyjson.Object); ok && pytext.Truthy(master) {
		return master
	}
	return gateway.EmptyMaster()
}

// Save stores the master block in the gateway section.
func (m *Master) Save(master *pyjson.Object) error {
	return m.projects.SaveGateway(with(m.gatewaySection(), "master", master))
}

func settingsOf(config *pyjson.Object) (string, *pyjson.Object) {
	protocol := text(config, "protocol")
	byProtocol, _ := get(config, "settings").(*pyjson.Object)
	settings, _ := get(byProtocol, protocol).(*pyjson.Object)
	return protocol, settings
}

// Busy reports a running worker (the role switch and settings need it stopped).
func (m *Master) Busy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.process != nil
}

// LinkOpen reports whether the device link is up.
func (m *Master) LinkOpen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.linkOpen
}

// tableFor: discovered points belong to one device; another protocol, host or port shows none.
func (m *Master) tableFor(config *pyjson.Object) *pyjson.Object {
	protocol, settings := settingsOf(config)
	if m.device != nil && !pytext.Equal(m.device, []any{protocol, get(config, "host"), get(settings, "port")}) {
		return pyjson.NewObject()
	}
	return m.table
}

// Points is the discovered table as display rows (an error only for data Python crashes on).
func (m *Master) Points() ([]any, error) {
	config := m.Config()
	protocol := text(config, "protocol")
	unticked, _ := get(get(config, "unticked").(*pyjson.Object), protocol).([]any)
	labels, _ := get(get(config, "labels").(*pyjson.Object), protocol).(*pyjson.Object)
	if labels == nil {
		labels = pyjson.NewObject()
	}
	hidden := make([]string, 0, len(unticked))
	for _, key := range unticked {
		hidden = append(hidden, pytext.Str(key))
	}
	open := m.LinkOpen()
	m.tableMu.Lock()
	defer m.tableMu.Unlock()
	return gateway.Rows(m.tableFor(config), protocol, hidden, labels, open)
}

// Status is the page's view of the master.
func (m *Master) Status() *pyjson.Object {
	config := m.Config()
	protocol, settings := settingsOf(config)
	m.tableMu.Lock()
	count := m.tableFor(config).Len()
	m.tableMu.Unlock()
	_, err := os.Stat(m.Executable)
	autoConnect, found := config.Get("auto_connect")
	if !found {
		autoConnect = false
	}
	ranges, _ := get(config, "modbus_ranges").([]any)
	m.mu.Lock()
	defer m.mu.Unlock()
	return pyjson.NewObject().Set("state", m.state).Set("message", m.message).Set("link_open", m.linkOpen).
		Set("protocol", protocol).Set("host", get(config, "host")).Set("port", get(settings, "port")).
		Set("points", int64(count)).Set("last_interrogation", m.lastAsked).Set("available", err == nil).
		Set("traffic_last", m.Traffic.Sequence()).Set("auto_connect", autoConnect).Set("probe", m.Probe.Snapshot()).
		Set("poll_failures", m.failures).Set("ranges", int64(len(ranges)))
}

func (m *Master) setMessage(message string) {
	m.mu.Lock()
	m.message = message
	m.mu.Unlock()
	m.changed(ChangeMasterStatus)
}

// activeProcess is _process: the worker while the link is open.
func (m *Master) activeProcess() (worker.Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.process == nil || !m.linkOpen {
		return nil, notConnected
	}
	return m.process, nil
}
