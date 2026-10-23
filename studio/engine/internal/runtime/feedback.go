package runtime

import (
	"sync"

	"studio-engine/internal/native"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/plant"
	"studio-engine/internal/runtime/pyval"
)

// feedback is NativeControlFeedback: an MMS control applied by a worker (from any client)
// drives the explicitly mapped equipment state, on its own goroutine, in arrival order. The
// queue is unbounded like Python's queue.Queue: enqueue runs on a worker's dispatcher, and a
// bounded queue could stall that pipe while a canvas command holding commandMu waits for a
// reply on it.
type feedback struct {
	c          *Coordinator
	mu         sync.Mutex
	generation int
	byControl  map[bindingKey][]*pyjson.Object
	queue      *native.Fifo[feedbackItem]
	done       chan struct{}
}

type feedbackItem struct {
	generation int
	iedID      string
	event      *pyjson.Object
	typed      bool // an INC, ENC, APC, BSC or ISC control (ADR-0104): no switch symbol to drive
}

func newFeedback(c *Coordinator) *feedback {
	f := &feedback{c: c, byControl: map[bindingKey][]*pyjson.Object{}, queue: native.NewFifo[feedbackItem](),
		done: make(chan struct{})}
	go f.run()
	return f
}

func (f *feedback) reset(diagram *pyjson.Object) {
	grouped := map[bindingKey][]*pyjson.Object{}
	for _, item := range pyval.List(diagram, "nodes") {
		node := item.(*pyjson.Object)
		if binding := pyval.Map(pyval.Map(node, "data"), "control_binding"); binding != nil && binding.Len() > 0 {
			key := bindingKey{pyval.Str(binding, "ied_id"), pyval.Str(binding, "ref")}
			grouped[key] = append(grouped[key], node)
		}
	}
	f.mu.Lock()
	f.generation++
	f.byControl = grouped
	f.mu.Unlock()
}

func (f *feedback) clear() {
	f.mu.Lock()
	f.generation++
	f.byControl = map[bindingKey][]*pyjson.Object{}
	f.mu.Unlock()
}

func (f *feedback) enqueue(iedID string, event *pyjson.Object) {
	value := pyval.Get(event, "ctl_value")
	_, isBool := value.(bool)
	if !isBool && value == nil {
		f.c.emit("equipment_feedback_failed", pyval.Obj("ied_id", iedID, "control_ref", pyval.Get(event, "ref"),
			"message", "Native control event has no Boolean ctlVal"))
		return
	}
	f.mu.Lock()
	generation := f.generation
	f.mu.Unlock()
	f.queue.Push(feedbackItem{generation, iedID, pyval.Copy(event).(*pyjson.Object), !isBool})
}

func (f *feedback) run() {
	defer close(f.done)
	for {
		entry, ok := f.queue.Pop()
		if !ok {
			return
		}
		f.process(entry)
	}
}

func (f *feedback) process(entry feedbackItem) {
	f.mu.Lock()
	if entry.generation != f.generation {
		f.mu.Unlock()
		return
	}
	nodes := append([]*pyjson.Object(nil), f.byControl[bindingKey{entry.iedID, pyval.Str(entry.event, "ref")}]...)
	f.mu.Unlock()
	// A canvas command in progress finishes first, as under Python's cycle_lock, so its
	// feedback events keep their order; nothing else waits on this narrow lock.
	f.c.commandMu.Lock()
	defer f.c.commandMu.Unlock()
	f.mu.Lock()
	stale := entry.generation != f.generation
	f.mu.Unlock()
	if stale {
		return
	}
	if !f.c.isLive() {
		return
	}
	for _, node := range nodes {
		if !entry.typed { // a typed value only moves its status, already in the table
			f.applyNode(node, entry.iedID, entry.event)
		}
	}
	statusRef, _ := pyval.Get(entry.event, "status_ref").(string)
	f.c.plant.publishChanged([]bindingKey{{entry.iedID, statusRef}})
	if isModRef(statusRef) { // a client changed a Mod: its Beh follow (ADR-0115)
		if err := f.c.syncBehaviour(entry.iedID); err != nil {
			f.c.emit("behaviour_failed", pyval.Obj("ied_id", entry.iedID, "ref", statusRef, "message", errorText(err)))
		}
	}
}

func (f *feedback) applyNode(node *pyjson.Object, iedID string, event *pyjson.Object) {
	c := f.c
	data := pyval.Map(node, "data")
	defer c.plant.publishNode(pyval.Str(node, "id"))
	failed := func(err error) {
		c.emit("equipment_feedback_failed", pyval.Obj("node_id", pyval.Get(node, "id"), "control_ref",
			pyval.Get(event, "ref"), "origin", "native_control", "message", errorText(err)))
	}
	action, wire, err := plant.SemanticCommand(data, "", pyval.Get(event, "ctl_value"))
	if err != nil {
		failed(err)
		return
	}
	origin := "An external client"
	if text, _ := pyval.Get(event, "client_origin").(string); text != "" {
		origin = text
	}
	c.plant.observe(node, action, origin)
	targets, err := plant.FeedbackTargets(data, pyval.Map(data, "state_mapping"), action)
	if err != nil {
		failed(err)
		return
	}
	statusRef := pyval.Get(event, "status_ref")
	type group struct {
		ied   string
		items []item
	}
	var groups []*group
	byIed := map[string]*group{}
	for _, target := range targets {
		current := c.plant.read(target.Binding)
		native := target.Binding.IedID == iedID && pyval.Equal(target.Binding.Ref, statusRef)
		if (!native || !pyval.Equal(target.Value, pyval.Get(event, "value"))) &&
			(current == nil && !c.indexed(target.Binding) || !pyval.Equal(current, target.Value)) {
			entry := byIed[target.Binding.IedID]
			if entry == nil {
				entry = &group{ied: target.Binding.IedID}
				byIed[entry.ied] = entry
				groups = append(groups, entry)
			}
			entry.items = append(entry.items, item{target.Binding.Ref, target.Value, nil})
		}
	}
	changed := int64(0)
	for _, entry := range groups {
		result, err := c.apply(entry.ied, entry.items, "control_feedback")
		if err != nil {
			failed(err)
			return
		}
		for _, written := range entry.items {
			if _, ok := result.Get(written.ref); ok {
				changed++
			}
		}
	}
	c.emit("equipment_feedback_observed", pyval.Obj("node_id", pyval.Get(node, "id"), "control_ref",
		pyval.Get(event, "ref"), "target_count", int64(len(targets)), "changed_count", changed, "action", action,
		"wire_value", wire, "origin", "native_control"))
}

// indexed reports whether a binding is in the run's signal index.
func (c *Coordinator) indexed(key plant.Binding) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	item := c.runs[key.IedID]
	if item == nil {
		return false
	}
	_, ok := item.model.index[key.Ref]
	return ok
}

func (f *feedback) close() {
	f.clear()
	f.queue.Close()
	<-f.done
}
