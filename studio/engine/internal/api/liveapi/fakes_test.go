package liveapi

import (
	"sync"
	"time"

	"studio-engine/internal/acquisition"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// fakeProjects stands in for stream B's project actor: an immutable tree replaced on change.
type fakeProjects struct {
	mu      sync.Mutex
	project *pyjson.Object
	runID   string
}

func (p *fakeProjects) Current() *pyjson.Object {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.project
}

func (p *fakeProjects) RunID() string { return p.runID }

func (p *fakeProjects) FindIED(id string) (*pyjson.Object, error) {
	for _, ied := range pyv.Objects(pyv.GetList(p.Current(), "ieds")) {
		if pyv.GetStr(ied, "id") == id {
			return ied, nil
		}
	}
	return nil, apperr.Coded("Unknown IED", "not_found")
}

// SetTestMode publishes a new project whose IED carries the settings (copy on write).
func (p *fakeProjects) SetTestMode(iedID string, settings *pyjson.Object) (*pyjson.Object, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	next := pyv.Shallow(p.project)
	ieds := []any{}
	var changed *pyjson.Object
	for _, ied := range pyv.Objects(pyv.GetList(p.project, "ieds")) {
		if pyv.GetStr(ied, "id") == iedID {
			ied = pyv.Shallow(ied)
			ied.Set("test_mode", settings)
			changed = ied
		}
		ieds = append(ieds, ied)
	}
	next.Set("ieds", ieds)
	p.project = next
	return changed, nil
}

func (p *fakeProjects) SaveReach(reach any, uplinkGUID, shareGUID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	network := pyv.Shallow(pyv.GetObj(p.project, "network"))
	network.Set("goose_reach", reach).Set("goose_uplink_guid", uplinkGUID).Set("goose_share_guid", shareGUID)
	next := pyv.Shallow(p.project)
	next.Set("network", network)
	p.project = next
	return nil
}

type fakeRuntime struct {
	mu    sync.Mutex
	state string
}

func (r *fakeRuntime) State() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}

// recorder captures emitted and published events as [kind, json.dumps(payload)].
type recorder struct {
	mu     sync.Mutex
	events [][2]string
}

func (r *recorder) Emit(kind string, payload *pyjson.Object) { r.Publish(kind, payload) }

func (r *recorder) Publish(kind string, payload *pyjson.Object) {
	encoded, err := pyjson.Dumps(payload)
	if err != nil {
		panic(err)
	}
	r.mu.Lock()
	r.events = append(r.events, [2]string{kind, string(encoded)})
	r.mu.Unlock()
}

func (r *recorder) take() [][2]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	events := r.events
	r.events = nil
	return events
}

type point struct {
	value any
	state any
}

// fakeValues is the runtime value table (stream C), seeded from the model at Run.
type fakeValues struct {
	mu   sync.Mutex
	ieds map[string]map[string]*point
}

func (v *fakeValues) seed(project *pyjson.Object) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.ieds = map[string]map[string]*point{}
	for _, ied := range pyv.Objects(pyv.GetList(project, "ieds")) {
		table := map[string]*point{}
		for _, signal := range pyv.Objects(pyv.GetList(ied, "signals")) {
			table[pyv.GetStr(signal, "ref")] = &point{value: pyv.Get(signal, "value"), state: pyv.Get(signal, "live_state")}
		}
		v.ieds[pyv.GetStr(ied, "id")] = table
	}
}

func (v *fakeValues) Has(iedID string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	_, found := v.ieds[iedID]
	return found
}

func (v *fakeValues) Get(iedID, ref string) (any, any, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	entry := v.ieds[iedID][ref]
	if entry == nil {
		return nil, nil, false
	}
	return entry.value, entry.state, true
}

func (v *fakeValues) SetLive(iedID string, values []acquisition.RefValue) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, item := range values {
		if entry := v.ieds[iedID][item.Ref]; entry != nil {
			entry.value, entry.state = item.Value, "fresh"
		}
	}
}

func (v *fakeValues) MarkStale(iedID string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, entry := range v.ieds[iedID] {
		if entry.state == "fresh" {
			entry.state = "stale"
		}
	}
}

type idle struct{}

func (idle) Running(string) bool                                { return false }
func (idle) Apply(string, []acquisition.RefValue, string) error { return nil }
func (idle) PublishChanged([]acquisition.PointKey)              {}
func (idle) SetState(string, string)                            {}

type fakeNetwork struct {
	adapters []*pyjson.Object
	npcap    *pyjson.Object
}

func (n fakeNetwork) Adapters() ([]*pyjson.Object, error) {
	copies := make([]*pyjson.Object, len(n.adapters))
	for index, adapter := range n.adapters {
		copies[index] = pyv.CopyObject(adapter)
	}
	return copies, nil
}

func (n fakeNetwork) Npcap() *pyjson.Object { return n.npcap }

type fixedClock struct {
	now       time.Time
	monotonic float64
}

func (c fixedClock) Now() time.Time     { return c.now }
func (c fixedClock) Monotonic() float64 { return c.monotonic }
