package acquisition

import (
	"os"
	"path/filepath"
	"sync"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// MissingLive is the refusal when studio_live.exe is not built (manager.py:50).
const MissingLive = "The live client (studio_live.exe) is not built yet; rebuild the native programs"

// NotReal is LiveManager.session's refusal (manager.py:90).
const NotReal = "This IED is not connected as a real device. Set it to 'real' and press Run."

// Deps are the engine services acquisition uses; every field is required unless noted.
type Deps struct {
	LiveExecutable string // path of studio_live.exe; studio_goose.exe is beside it
	Start          StartWorker
	Projects       Projects
	Runtime        Runtime
	Events         Emitter
	Bus            Publisher
	Values         Values
	Samples        Samples
	Equipment      Equipment
	States         IEDStates
	Network        Network
	Clock          Clock
	Members        SubscriptionMembers // optional; nil keeps a subscription as stored
}

// Manager is LiveManager: real-IED sessions and the GOOSE/SV listeners of one Run.
type Manager struct {
	deps   Deps
	values *LiveValues
	Engine *GooseEngine
	Share  *GooseShare
	Goose  *GooseMonitor
	SV     *SvMonitor

	mu       sync.Mutex
	sessions []*RealSession // insertion order of the Python dict
}

// NewManager wires the listeners like LiveManager.__init__.
func NewManager(deps Deps) *Manager {
	m := &Manager{deps: deps}
	m.values = NewLiveValues(deps)
	m.Engine = NewGooseEngine(deps, filepath.Join(filepath.Dir(deps.LiveExecutable), "studio_goose.exe"))
	m.Share = NewGooseShare(deps, m.Engine)
	m.Goose = NewGooseMonitor(deps, m.Engine, m.Share, m.values)
	m.SV = NewSvMonitor(deps)
	return m
}

// Values is the live value writer (used by sessions and the GOOSE monitor).
func (m *Manager) Values() *LiveValues { return m.values }

func (m *Manager) liveExists() bool {
	info, err := os.Stat(m.deps.LiveExecutable)
	return err == nil && !info.IsDir()
}

// Start starts the listeners and connects the real IEDs in selected (all when nil).
func (m *Manager) Start(project *pyjson.Object, selected map[string]bool) error {
	ieds := pyv.Objects(pyv.GetList(project, "ieds"))
	var real []*pyjson.Object
	needsLive := false
	for _, ied := range ieds {
		if pyv.GetStr(ied, "role") == "real" && (selected == nil || selected[pyv.GetStr(ied, "id")]) {
			real = append(real, ied)
		}
		goose, sv := pyv.OrObj(ied, "goose"), pyv.OrObj(ied, "sv")
		if pyv.Truthy(pyv.Get(goose, "publishes")) || pyv.Truthy(pyv.Get(goose, "subscribes")) ||
			pyv.Truthy(pyv.Get(sv, "publishes")) {
			needsLive = true
		}
	}
	if (len(real) > 0 || needsLive) && !m.liveExists() {
		return apperr.ValueError(MissingLive)
	}
	if err := m.Connect(project, real); err != nil {
		return err
	}
	if err := m.startGoose(project, ieds); err != nil {
		m.Goose.fail(err)
	}
	if err := m.SV.Start(project, pyv.Truthy(pyv.Get(m.deps.Network.Npcap(), "installed"))); err != nil {
		m.SV.setStatus(pyv.O("state", "failed", "message", "Sampled Values listener could not start: "+err.Error()))
		m.deps.Events.Emit("sv_listener_failed", pyv.O("message", err.Error()))
	}
	return nil
}

func (m *Manager) startGoose(project *pyjson.Object, ieds []*pyjson.Object) error {
	if err := m.Goose.Start(ieds); err != nil {
		return err
	}
	if m.Engine.Running() {
		return m.Share.Start(project)
	}
	return nil
}

// Connect connects more real IEDs while the run continues (client read only, ADR-0067).
// A session for an IED already connected takes its place (Python overwrote the dict entry;
// Go also stops the replaced session instead of leaving its thread running).
func (m *Manager) Connect(project *pyjson.Object, real []*pyjson.Object) error {
	if len(real) > 0 && !m.liveExists() {
		return apperr.ValueError(MissingLive)
	}
	for _, ied := range real {
		session := NewRealSession(m.deps, m.values, ied, WatchedPoints(project, ied))
		m.mu.Lock()
		replaced := m.replaceLocked(session)
		m.mu.Unlock()
		if replaced != nil {
			replaced.Stop()
		}
		session.Start()
	}
	return nil
}

func (m *Manager) replaceLocked(session *RealSession) *RealSession {
	for index, existing := range m.sessions {
		if existing.ID() == session.ID() {
			m.sessions[index] = session
			return existing
		}
	}
	m.sessions = append(m.sessions, session)
	return nil
}

func (m *Manager) removeLocked(id string) *RealSession {
	for index, session := range m.sessions {
		if session.ID() == id {
			m.sessions = append(m.sessions[:index:index], m.sessions[index+1:]...)
			return session
		}
	}
	return nil
}

// Disconnect stops one real IED's session.
func (m *Manager) Disconnect(iedID string) {
	m.mu.Lock()
	session := m.removeLocked(iedID)
	m.mu.Unlock()
	if session != nil {
		session.Stop()
	}
}

// Stop stops the GOOSE monitor, the engine, SV and then every session (manager.py:79-85).
func (m *Manager) Stop() {
	m.Goose.Stop()
	m.Engine.Stop()
	m.SV.Stop()
	m.mu.Lock()
	sessions := m.sessions
	m.sessions = nil
	m.mu.Unlock()
	for _, session := range sessions {
		session.Stop()
	}
}

// Session returns the IED's session or the ValueError NotReal.
func (m *Manager) Session(iedID string) (*RealSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, session := range m.sessions {
		if session.ID() == iedID {
			return session, nil
		}
	}
	return nil, apperr.ValueError(NotReal)
}

// Sessions returns the sessions in connection order.
func (m *Manager) Sessions() []*RealSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*RealSession(nil), m.sessions...)
}

// Status is GET /live: {"sessions": [status with comparison/health None], "goose": snapshot}.
func (m *Manager) Status() *pyjson.Object {
	sessions := []any{}
	for _, session := range m.Sessions() {
		status := session.Status()
		status.Set("comparison", nil)
		status.Set("health", nil)
		sessions = append(sessions, status)
	}
	return pyv.O("sessions", sessions, "goose", m.Goose.Snapshot())
}
