package acquisition

import (
	"sync"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

type testProjects struct{ project *pyjson.Object }

func (p testProjects) Current() *pyjson.Object { return p.project }
func (p testProjects) RunID() string           { return "test-1" }
func (p testProjects) FindIED(id string) (*pyjson.Object, error) {
	if ied := findIED(p.project, id); ied != nil {
		return ied, nil
	}
	return nil, apperr.Coded("Unknown IED", "not_found")
}
func (p testProjects) SetTestMode(string, *pyjson.Object) (*pyjson.Object, error) { return nil, nil }
func (p testProjects) SaveReach(any, string, string) error                        { return nil }

type testRuntime struct{ state string }

func (r testRuntime) State() string { return r.state }

type testEvents struct {
	mu     sync.Mutex
	events []string
}

func (e *testEvents) Emit(kind string, payload *pyjson.Object) { e.Publish(kind, payload) }
func (e *testEvents) Publish(kind string, payload *pyjson.Object) {
	encoded, _ := pyjson.Dumps(payload)
	e.mu.Lock()
	e.events = append(e.events, kind+" "+string(encoded))
	e.mu.Unlock()
}

func (e *testEvents) all() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.events...)
}

type testValues struct {
	mu     sync.Mutex
	points map[string]map[string][2]any
}

func newTestValues(project *pyjson.Object) *testValues {
	v := &testValues{points: map[string]map[string][2]any{}}
	for _, ied := range pyv.Objects(pyv.GetList(project, "ieds")) {
		table := map[string][2]any{}
		for _, signal := range pyv.Objects(pyv.GetList(ied, "signals")) {
			table[pyv.GetStr(signal, "ref")] = [2]any{pyv.Get(signal, "value"), nil}
		}
		v.points[pyv.GetStr(ied, "id")] = table
	}
	return v
}

func (v *testValues) Has(id string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	_, found := v.points[id]
	return found
}

func (v *testValues) Get(id, ref string) (any, any, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	point, found := v.points[id][ref]
	return point[0], point[1], found
}

func (v *testValues) SetLive(id string, values []RefValue) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, item := range values {
		if _, found := v.points[id][item.Ref]; found {
			v.points[id][item.Ref] = [2]any{item.Value, "fresh"}
		}
	}
}

func (v *testValues) MarkStale(id string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for ref, point := range v.points[id] {
		if point[1] == "fresh" {
			v.points[id][ref] = [2]any{point[0], "stale"}
		}
	}
}

type testIdle struct{}

func (testIdle) Running(string) bool                    { return false }
func (testIdle) Apply(string, []RefValue, string) error { return nil }
func (testIdle) PublishChanged([]PointKey)              {}
func (testIdle) SetState(string, string)                {}
func (testIdle) Adapters() ([]*pyjson.Object, error)    { return nil, nil }
func (testIdle) Npcap() *pyjson.Object                  { return pyv.O("installed", false, "message", "no npcap") }

type testClock struct{ start time.Time }

func (c testClock) Now() time.Time     { return time.Now().UTC() }
func (c testClock) Monotonic() float64 { return time.Since(c.start).Seconds() }
