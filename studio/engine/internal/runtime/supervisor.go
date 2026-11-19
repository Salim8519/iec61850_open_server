package runtime

import (
	"errors"
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

// startBatch is the most attributes the worker takes in one update_batch.
const startBatch = 4096

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
// in project order is returned, as the serial Python loop would. A fresh Run opens the GOOSE
// bus while the lab addresses are prepared (ADR-0133): a failed fresh Run stops the GOOSE engine
// anyway, and the errors keep their order. The IED processes start after both, so a slow PC
// never starts the GOOSE engine and every IED process at the same moment.
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
	bus := c.openBus(ieds, len(current) == 0)
	if err := c.prepareNetwork(batch, ieds, network); err != nil {
		bus.wait()
		return err
	}
	folder := filepath.Join(c.opt.DataDir, "runtime")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		bus.wait()
		return apperr.RuntimeError(err.Error())
	}
	buses, err := bus.result(ieds)
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

// prepareNetwork puts the lab card's addresses on (lab mode) and runs the preflight checks.
func (c *Coordinator) prepareNetwork(batch []*running, ieds []*pyjson.Object, network *pyjson.Object) error {
	var allowed []string
	if pyval.Str(network, "mode") == "lab" {
		starting := make([]string, len(batch))
		for i, item := range batch {
			starting[i] = item.model.id
		}
		addresses, err := c.opt.LabAddresses(pyval.Str(network, "adapter_guid"), starting)
		if err != nil {
			return err // not an apperr: Flask's generic 500, its text in the runtime event
		}
		allowed = addresses
	}
	err := preflight(ieds, network, allowed)
	var refused *listenFailure
	if errors.As(err, &refused) {
		c.emit("listen_check_failed", refused.event) // the run log keeps Windows' answer (ADR-0134)
	}
	return err
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
	point := effectiveEndpoint(item.ied, network)
	request := startRequest(item, absolute, point, gooseRequest(item.ied, bus))
	started, err := worker.StartModel(request)
	if err != nil {
		worker.Close() // never published, so stopWorkers would not find it
		return err
	}
	host, port := point.host, portNumber(point.port)
	if value, ok := started.Get("ip"); ok {
		host, _ = value.(string)
	}
	if value, ok := started.Get("port"); ok {
		port = portNumber(value)
	}
	goose, ok := started.Get("goose")
	if !ok {
		goose = pyjson.NewObject()
	}
	// Publish the worker under the lock only once its model started: until then commands
	// and sample writes find "IED is not running".
	c.mu.Lock()
	item.worker, item.host, item.port = worker, host, port
	c.gooseStates[id] = goose
	c.endpoints[id] = endpoint{item.host, item.port}
	c.hub.notify(Change{IEDID: id})
	c.mu.Unlock()
	item.startActor()
	if err := writeStartValues(item, worker); err != nil {
		return err
	}
	stampStartTimes(item, pyval.Get(started, "started_at"))
	c.mu.Lock()
	c.setIedStateLocked(id, "running")
	c.mu.Unlock()
	return nil
}

// writeStartValues writes the last known positions, commands, qualities and changed times in
// signal order, in batches of one data-model lock each (ADR-0133; the worker validates a whole
// batch before it changes anything, so a refused value fails the Run as the single write did).
func writeStartValues(item *running, worker *native.Ied) error {
	previousStart := item.table.startTime()
	var updates []any
	for i, entry := range item.model.signals {
		if !startKinds[entry.Kind] {
			continue
		}
		value, _ := item.table.get(item.model.signals[i].Ref)
		if entry.Kind == "Timestamp" && previousStart != nil && pyval.Equal(value, previousStart) {
			continue // never changed in the last run: stampStartTimes gives it this run's start
		}
		if pyval.Truthy(value) {
			updates = append(updates, pyjson.NewObject().Set("ref", entry.Ref).Set("value", value))
		}
	}
	for len(updates) > 0 {
		size := min(len(updates), startBatch)
		if _, err := worker.UpdateBatch(updates[:size]); err != nil {
			return err
		}
		updates = updates[size:]
	}
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
