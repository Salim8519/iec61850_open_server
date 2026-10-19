package runtime

import (
	"os"
	"path/filepath"
	"sync"

	"studio-engine/internal/apperr"
	"studio-engine/internal/native"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// startKinds are the signal kinds whose stored value is written right after start
// (supervisor._launch), so a worker starts with the last known positions.
var startKinds = map[string]bool{"Dbpos": true, "Tcmd": true, "Quality": true, "Timestamp": true}

// gooseRequest is worker_settings.goose_request.
func gooseRequest(ied *pyjson.Object, bus string) *pyjson.Object {
	publish := bus != "" && pyval.Truthy(getOr(ied, "goose_publish", true)) &&
		pyval.Truthy(pyval.Get(pyval.Map(ied, "goose"), "publishes"))
	interfaceName := ""
	if publish {
		interfaceName = bus
	}
	return pyval.Obj("publish", publish, "interface", interfaceName)
}

func getOr(object *pyjson.Object, key string, fallback any) any {
	if value, ok := object.Get(key); ok {
		return value
	}
	return fallback
}

// addWorkers is Supervisor.add: endpoints, preflight, GOOSE bus, then every worker started
// in parallel (0111 rule 9). On any failure the whole batch is stopped and the first error
// in project order is returned, as the serial Python loop would.
func (c *Coordinator) addWorkers(batch []*running, network *pyjson.Object, current []*running) error {
	executable := filepath.Join(c.opt.NativeDir, "studio_ied.exe")
	if _, err := os.Stat(executable); err != nil {
		return apperr.ValueError("Native IED engine is not built yet")
	}
	everyone := make([]*pyjson.Object, 0, len(batch)+len(current))
	for _, item := range current {
		everyone = append(everyone, item.ied)
	}
	ieds := make([]*pyjson.Object, len(batch))
	for i, item := range batch {
		ieds[i] = item.ied
		everyone = append(everyone, item.ied)
	}
	if err := c.assignEndpoints(everyone); err != nil {
		return err
	}
	var allowed []string
	if pyval.Str(network, "mode") == "lab" {
		allowed = c.opt.LabAddresses(pyval.Str(network, "adapter_guid"))
	}
	if err := preflight(ieds, network, allowed); err != nil {
		return err
	}
	folder := filepath.Join(c.opt.DataDir, "runtime")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return apperr.RuntimeError(err.Error())
	}
	buses, err := c.opt.Live.BusFor(ieds)
	if err != nil {
		return err
	}
	failures := make([]error, len(batch))
	var group sync.WaitGroup
	for i, item := range batch {
		group.Add(1)
		go func(i int, item *running) {
			defer group.Done()
			failures[i] = c.launch(item, executable, folder, network, buses[item.model.id])
		}(i, item)
	}
	group.Wait()
	for _, failure := range failures {
		if failure != nil {
			c.stopWorkers(batch)
			return failure
		}
	}
	return nil
}

func (c *Coordinator) launch(item *running, executable, folder string, network *pyjson.Object, bus string) error {
	id := item.model.id
	config := filepath.Join(folder, id+".cfg")
	if err := os.WriteFile(config, []byte(pyval.Str(item.ied, "cfg")), 0o644); err != nil {
		return apperr.RuntimeError(err.Error())
	}
	absolute, _ := filepath.Abs(config)
	worker, err := native.StartIed(executable, func(event *pyjson.Object) { c.onWorkerEvent(id, event) })
	if err != nil {
		return apperr.RuntimeError(err.Error())
	}
	item.worker = worker
	item.startActor()
	point := effectiveEndpoint(item.ied, network)
	request := pyval.Obj("cfg", absolute, "ip", point.host, "port", point.port, "controls", item.model.controls(),
		"edition", pyval.Get(item.ied, "edition"), "goose", gooseRequest(item.ied, bus))
	started, err := worker.StartModel(request)
	if err != nil {
		return err
	}
	item.host, item.port = point.host, portNumber(point.port)
	if host, ok := started.Get("ip"); ok {
		item.host, _ = host.(string)
	}
	if port, ok := started.Get("port"); ok {
		item.port = portNumber(port)
	}
	goose, ok := started.Get("goose")
	if !ok {
		goose = pyjson.NewObject()
	}
	c.mu.Lock()
	c.gooseStates[id] = goose
	c.endpoints[id] = endpoint{item.host, item.port}
	c.hub.notify(Change{IEDID: id})
	c.mu.Unlock()
	for i, entry := range item.model.signals {
		if !startKinds[entry.Kind] {
			continue
		}
		value, _ := item.table.get(item.model.signals[i].Ref)
		if pyval.Truthy(value) {
			if _, err := worker.Update(entry.Ref, value); err != nil {
				return err
			}
		}
	}
	c.mu.Lock()
	c.setIedStateLocked(id, "running")
	c.mu.Unlock()
	return nil
}

// stopWorkers is Supervisor.stop_some: close these workers in parallel, mark the IEDs stopped.
func (c *Coordinator) stopWorkers(batch []*running) {
	var group sync.WaitGroup
	for _, item := range batch {
		if item.worker == nil && item.simulated() {
			c.markStopped(item.model.id)
			continue
		}
		if !item.simulated() {
			continue
		}
		group.Add(1)
		go func(item *running) {
			defer group.Done()
			item.stopActor()
			item.worker.Close()
		}(item)
	}
	group.Wait()
	for _, item := range batch {
		if item.simulated() {
			c.markStopped(item.model.id)
		}
	}
}

func (c *Coordinator) markStopped(id string) {
	c.mu.Lock()
	c.setIedStateLocked(id, "stopped")
	delete(c.endpoints, id)
	delete(c.gooseStates, id)
	c.mu.Unlock()
}
