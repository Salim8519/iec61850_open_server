package acquisition

import (
	"os"
	"strings"
	"sync"
	"time"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Texts of acquisition/goose_engine.py.
const (
	GooseMissing = "The GOOSE engine (studio_goose.exe) is not built yet; run studio/tools/build_goose_engine.py"
	GooseStale   = "The GOOSE engine is outdated. Rebuild it with studio/tools/build_goose_engine.py before Run"
	engineCall   = 10 * time.Second
)

func closedLink() *pyjson.Object {
	return pyv.O("adapter_guid", "", "uplink", "closed", "share", "off", "message", "")
}

// GooseEngine owns studio_goose.exe for one Run (ADR-0073). calls serialises requests;
// mu guards the state events update and is never held while waiting for a reply.
type GooseEngine struct {
	deps       Deps
	executable string

	calls     sync.Mutex
	mu        sync.Mutex
	process   Worker
	port      any
	sources   *pyjson.Object
	link      *pyjson.Object
	conflicts *pyjson.Object
}

// NewGooseEngine returns a stopped engine for executable.
func NewGooseEngine(deps Deps, executable string) *GooseEngine {
	return &GooseEngine{deps: deps, executable: executable, sources: pyjson.NewObject(),
		link: closedLink(), conflicts: pyjson.NewObject()}
}

// Running reports whether the engine process runs.
func (e *GooseEngine) Running() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.process != nil
}

func (e *GooseEngine) current() Worker {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.process
}

// Ensure starts the engine once per Run and refuses a missing or outdated binary.
func (e *GooseEngine) Ensure() error {
	e.calls.Lock()
	defer e.calls.Unlock()
	return e.ensureLocked()
}

func (e *GooseEngine) ensureLocked() error {
	if e.current() != nil {
		return nil
	}
	if info, err := os.Stat(e.executable); err != nil || info.IsDir() {
		return apperr.ValueError(GooseMissing)
	}
	process, err := e.deps.Start(e.executable, e.onEvent)
	if err != nil {
		return err
	}
	hello, err := process.Request("hello", engineCall, pyjson.NewObject())
	if err != nil {
		process.Close()
		return err
	}
	if !hasAll(pyv.GetList(pyv.Obj(hello), "capabilities"), goosedom.EngineCapabilities()) {
		process.Close()
		return apperr.ValueError(GooseStale)
	}
	e.mu.Lock()
	e.process = process
	e.mu.Unlock()
	opened, err := process.Request("open", engineCall, pyv.O("ieds", []any{}))
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.port = pyv.Get(pyv.Obj(opened), "port")
	e.mu.Unlock()
	return nil
}

// BusFor registers the publishing simulated IEDs and returns {id: "udp:PORT:MAC"}.
func (e *GooseEngine) BusFor(ieds []*pyjson.Object) (*pyjson.Object, error) {
	entries := GoosePublishers(ieds)
	if len(entries) == 0 {
		return pyjson.NewObject(), nil
	}
	e.calls.Lock()
	defer e.calls.Unlock()
	if err := e.ensureLocked(); err != nil {
		return nil, err
	}
	answer, err := e.current().Request("open", engineCall, pyv.O("ieds", entries))
	if err != nil {
		return nil, err
	}
	opened := pyv.Obj(answer)
	sources := pyv.OrObj(opened, "sources")
	e.mu.Lock()
	pyv.Update(e.sources, sources)
	e.mu.Unlock()
	result := pyjson.NewObject()
	for _, id := range sources.Keys() {
		mac, _ := sources.Get(id)
		result.Set(id, "udp:"+pyv.PyStr(pyv.Get(opened, "port"))+":"+strings.ReplaceAll(pyv.PyStr(mac), "-", ""))
	}
	return result, nil
}

// GoosePublishers lists the simulated IEDs whose GOOSE publishing is on, with their gocbRefs.
func GoosePublishers(ieds []*pyjson.Object) []any {
	result := []any{}
	for _, ied := range ieds {
		goose := pyv.OrObj(ied, "goose")
		if !pyv.Truthy(pyv.Get(goose, "publishes")) || !pyv.Truthy(pyv.GetOr(ied, "goose_publish", true)) ||
			!pyv.Equal(pyv.GetOr(ied, "role", "simulated"), "simulated") {
			continue
		}
		refs := []any{}
		for _, control := range pyv.Objects(pyv.GetList(goose, "publishes")) {
			refs = append(refs, pyv.Get(control, "gocb_ref"))
		}
		result = append(result, pyv.O("id", pyv.Get(ied, "id"), "gocb_refs", refs))
	}
	return result
}

// Deliver tells the engine where studio_live's GOOSE inbox listens.
func (e *GooseEngine) Deliver(inboxPort any) error {
	e.calls.Lock()
	defer e.calls.Unlock()
	if err := e.ensureLocked(); err != nil {
		return err
	}
	_, err := e.current().Request("deliver", engineCall, pyv.O("port", inboxPort))
	return err
}

// Uplink listens on the adapter ("" = none) and shares outside only when share is true;
// without confirmReal the engine holds sharing next to a real device (ADR-0119).
func (e *GooseEngine) Uplink(adapterGUID string, share, confirmReal bool) (*pyjson.Object, error) {
	e.calls.Lock()
	defer e.calls.Unlock()
	process := e.current()
	if process == nil {
		e.mu.Lock()
		defer e.mu.Unlock()
		return pyv.Shallow(e.link), nil
	}
	request := pyv.O("adapter_guid", adapterGUID, "share", share)
	if !goosedom.PythonTwin() {
		request.Set("confirm_real", confirmReal)
	}
	result, err := process.Request("uplink", engineCall, request)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if object := pyv.Obj(result); object != nil {
		pyv.Update(e.link, object)
	}
	return pyv.Shallow(e.link), nil
}

// Gate answers a conflict for an IED: "stop" or "continue".
func (e *GooseEngine) Gate(iedID, choice string) (any, error) {
	e.calls.Lock()
	defer e.calls.Unlock()
	process := e.current()
	if process == nil {
		return nil, apperr.ValueError("Press Run first: GOOSE is not running")
	}
	result, err := process.Request("gate", engineCall, pyv.O("ied_id", iedID, "choice", choice))
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	if conflict := pyv.GetObj(e.conflicts, iedID); conflict != nil {
		updated := pyv.Shallow(conflict)
		updated.Set("choice", choice)
		e.conflicts.Set(iedID, updated)
	}
	e.mu.Unlock()
	return result, nil
}

// Stats returns the engine counters, {} when stopped.
func (e *GooseEngine) Stats() (*pyjson.Object, error) {
	e.calls.Lock()
	defer e.calls.Unlock()
	process := e.current()
	if process == nil {
		return pyjson.NewObject(), nil
	}
	result, err := process.Request("stats", engineCall, pyjson.NewObject())
	if err != nil {
		return nil, err
	}
	if object := pyv.Obj(result); object != nil {
		return object, nil
	}
	return pyjson.NewObject(), nil
}
