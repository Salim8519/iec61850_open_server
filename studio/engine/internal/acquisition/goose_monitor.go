package acquisition

import (
	"sync"
	"time"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

var subscriptionKeys = []string{"key", "gocb_ref", "app_id", "mac", "go_id", "dataset_ref", "conf_rev"}

// GooseMonitor decodes GOOSE from the app's GOOSE engine (through studio_live's inbox) and
// routes it to supervision and live values (acquisition/goose_monitor.py).
type GooseMonitor struct {
	deps   Deps
	engine *GooseEngine
	share  *GooseShare
	values *LiveValues
	inputs *GooseInputs
	refs   refIndex

	mu      sync.Mutex
	process Worker
	watches *pyjson.Object // key -> watch, updated only under mu
	status  *pyjson.Object
}

// NewGooseMonitor returns a stopped monitor.
func NewGooseMonitor(deps Deps, engine *GooseEngine, share *GooseShare, values *LiveValues) *GooseMonitor {
	return &GooseMonitor{deps: deps, engine: engine, share: share, values: values,
		inputs: NewGooseInputs(deps), watches: pyjson.NewObject(), status: offStatus()}
}

func offStatus() *pyjson.Object {
	return pyv.O("state", "off", "message", "", "unknown", []any{})
}

// Start supervises every GoCB of the project; GOOSE runs in every network mode, with or
// without Npcap (ADR-0073, GOOSE-17).
func (g *GooseMonitor) Start(ieds []*pyjson.Object) error {
	watches := goosedom.WatchList(ieds)
	for _, key := range watches.Keys() {
		watch := pyv.GetObj(watches, key)
		watch.Set("state", "stopped").Set("problem", "").Set("frames", int64(0))
	}
	g.mu.Lock()
	g.watches = watches
	if watches.Len() == 0 {
		g.status = pyv.O("state", "nothing_to_watch", "unknown", []any{},
			"message", "No IED in this project publishes or subscribes to GOOSE.")
		g.mu.Unlock()
		return nil
	}
	for _, key := range watches.Keys() {
		watch := pyv.GetObj(watches, key)
		watch.Set("state", "waiting").Set("problem", "").Set("frames", int64(0)).Set("st_num", nil).Set("sq_num", nil)
	}
	g.inputs.Prepare(ieds, watches)
	g.mu.Unlock()
	if err := g.engine.Ensure(); err != nil {
		return err
	}
	process, err := g.deps.Start(g.deps.LiveExecutable, g.onEvent)
	if err != nil {
		return err
	}
	g.mu.Lock()
	g.process = process
	subscriptions := []any{}
	for _, key := range watches.Keys() {
		watch := pyv.GetObj(watches, key)
		entry := pyjson.NewObject()
		for _, name := range subscriptionKeys {
			entry.Set(name, pyv.Get(watch, name))
		}
		subscriptions = append(subscriptions, entry)
	}
	g.mu.Unlock()
	started, err := process.Request("goose_start", 30*time.Second, pyv.O("subscriptions", subscriptions))
	if err != nil {
		return err
	}
	inbox := pyv.Get(pyv.Obj(started), "inbox_port")
	if !pyv.Truthy(inbox) { // an older studio_live listens on an adapter instead
		return apperr.ValueError("The GOOSE listener is outdated (no goose_inbox). Rebuild studio_live.exe.")
	}
	if err := g.engine.Deliver(inbox); err != nil {
		return err
	}
	g.setStatus(pyv.O("state", "inside", "unknown", []any{},
		"message", "GOOSE runs inside ScadaBench. Nothing is sent to the network."))
	return nil
}

func (g *GooseMonitor) setStatus(status *pyjson.Object) {
	g.mu.Lock()
	g.status = status
	g.mu.Unlock()
}

// fail records a start failure like LiveManager.start's except branch.
func (g *GooseMonitor) fail(err error) {
	g.mu.Lock()
	g.status = pyv.Shallow(g.status)
	g.status.Set("state", "failed").Set("message", "GOOSE listener could not start: "+err.Error())
	g.mu.Unlock()
	g.deps.Events.Emit("goose_listener_failed", pyv.O("message", err.Error()))
}

// Snapshot is the GOOSE view: status, watches and the engine state.
func (g *GooseMonitor) Snapshot() *pyjson.Object {
	g.mu.Lock()
	status := pyv.Shallow(g.status)
	status.Set("unknown", append([]any{}, pyv.GetList(g.status, "unknown")...))
	watches := []any{}
	for _, key := range g.watches.Keys() {
		watches = append(watches, pyv.Shallow(pyv.GetObj(g.watches, key)))
	}
	running := g.process != nil
	g.mu.Unlock()
	if running && pyv.GetStr(status, "state") != "failed" {
		state, message := g.share.Status(g.deps.Projects.Current())
		status.Set("state", state).Set("message", message)
	}
	status.Set("watches", watches)
	status.Set("engine", g.engine.Snapshot())
	return status
}

// Watches returns copies of the current watches (the link map joins them).
func (g *GooseMonitor) Watches() []*pyjson.Object {
	g.mu.Lock()
	defer g.mu.Unlock()
	result := make([]*pyjson.Object, 0, g.watches.Len())
	for _, key := range g.watches.Keys() {
		result = append(result, pyv.Shallow(pyv.GetObj(g.watches, key)))
	}
	return result
}

// Stop closes the listener and returns to "off".
func (g *GooseMonitor) Stop() {
	g.mu.Lock()
	process := g.process
	g.process = nil
	g.status = offStatus()
	g.mu.Unlock()
	if process != nil {
		process.Close()
	}
}
