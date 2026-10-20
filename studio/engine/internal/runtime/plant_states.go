package runtime

import (
	"sync"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/plant"
	"studio-engine/internal/runtime/pyval"
)

type bindingKey struct{ ied, ref string }

// ownCommandSeconds is energization_states.OWN_COMMAND_S.
const ownCommandSeconds = 20

// plantStates is EquipmentStates plus EnergizationStates: mapped equipment display states,
// published only when they change, and the energisation recomputed after a state publish.
type plantStates struct {
	c        *Coordinator
	mu       sync.Mutex
	order    []string
	mappings map[string]*pyjson.Object
	bySource map[bindingKey][]string
	last     map[string]any
	diagram  *pyjson.Object
	lastPlan *pyjson.Object // last published energisation {nodes, edges, hints}
	own      map[string]float64
}

func newPlantStates(c *Coordinator) *plantStates {
	return &plantStates{c: c, own: map[string]float64{}}
}

// reset is EquipmentStates.reset and EnergizationStates.reset for a run.
func (p *plantStates) reset(diagram *pyjson.Object) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.order, p.mappings, p.bySource = nil, map[string]*pyjson.Object{}, map[bindingKey][]string{}
	p.last, p.diagram, p.lastPlan = map[string]any{}, diagram, nil
	add := func(id string, mapping *pyjson.Object) {
		if _, seen := p.mappings[id]; !seen {
			p.order = append(p.order, id)
		}
		p.mappings[id] = mapping
	}
	nodes := pyval.List(diagram, "nodes")
	for _, item := range nodes {
		node := item.(*pyjson.Object)
		if mapping := pyval.Map(pyval.Map(node, "data"), "state_mapping"); mapping != nil && mapping.Len() > 0 {
			add(pyval.Str(node, "id"), mapping)
		}
	}
	for _, item := range nodes {
		node := item.(*pyjson.Object)
		data := pyval.Map(node, "data")
		rows := "relay_statuses"
		switch pyval.Str(data, "kind") {
		case "RELAY":
		case "MEASUREMENT_LIST": // raw values; the panel scales them (ADR-0090)
			rows = "measurements"
		default:
			continue
		}
		for _, entry := range pyval.List(data, rows) {
			status := entry.(*pyjson.Object)
			add(pyval.Str(node, "id")+":"+pyval.Str(status, "id"), pyval.Obj("mode", "direct", "source", pyval.Get(status, "source")))
		}
	}
	for _, id := range p.order {
		for _, key := range plant.BindingKeys(p.mappings[id]) {
			p.bySource[bindingKey{key.IedID, key.Ref}] = appendUnique(p.bySource[bindingKey{key.IedID, key.Ref}], id)
		}
	}
}

func appendUnique(items []string, item string) []string {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

// read is EquipmentStates._read: the indexed value of a binding, None when not in the run.
func (p *plantStates) read(key plant.Binding) any {
	p.c.mu.Lock()
	item := p.c.runs[key.IedID]
	p.c.mu.Unlock()
	if item == nil {
		return nil
	}
	value, _ := item.table.get(key.Ref)
	return value
}

func (p *plantStates) evaluate(id string) any { return plant.EvaluateState(p.mappings[id], p.read) }

// publishLocked is EquipmentStates._publish; needs p.mu.
func (p *plantStates) publishLocked(id string, force bool) {
	value := p.evaluate(id)
	if force || !pyval.Equal(p.last[id], value) {
		p.last[id] = value
		p.c.emit("equipment_state", pyval.Obj("node_id", id, "value", value))
	}
}

func (p *plantStates) publishAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, id := range p.order {
		p.publishLocked(id, true)
	}
	p.energize(true)
}

func (p *plantStates) publishNode(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.mappings[id]; ok {
		p.publishLocked(id, true)
		p.energize(false)
	}
}

// publishChanged publishes the nodes mapped to these points; energisation is recomputed only
// when a mapped state point was written (0111 rule 7).
func (p *plantStates) publishChanged(keys []bindingKey) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var affected []string
	for _, key := range keys {
		for _, id := range p.bySource[key] {
			affected = appendUnique(affected, id)
		}
	}
	for _, id := range affected {
		p.publishLocked(id, false)
	}
	if len(affected) > 0 {
		p.energize(false)
	}
}

// plan is EnergizationStates._current; nil when not running.
func (p *plantStates) plan() *pyjson.Object {
	if !p.c.isLive() || p.diagram == nil || p.diagram.Len() == 0 {
		return nil
	}
	positions := map[string]any{}
	for _, id := range p.order {
		positions[id] = p.evaluate(id)
	}
	sections := plant.NewSections(p.diagram, positions)
	nodes, edges := sections.Describe()
	return pyval.Obj("nodes", nodes, "edges", edges, "hints", sections.SwitchingHints())
}

// energize is EnergizationStates.publish; needs p.mu.
func (p *plantStates) energize(force bool) {
	result := p.plan()
	if result == nil || (!force && p.lastPlan != nil && pyval.Equal(result, p.lastPlan)) {
		return
	}
	p.lastPlan = result
	p.c.emit("energization", result)
}

// markOwn is EnergizationStates.mark_own.
func (p *plantStates) markOwn(id string) {
	p.mu.Lock()
	p.own[id] = p.c.now() + ownCommandSeconds
	p.mu.Unlock()
}

// observe is EnergizationStates.observe: an external client switched against a hint.
func (p *plantStates) observe(node *pyjson.Object, action, origin string) {
	id := pyval.Str(node, "id")
	p.mu.Lock()
	until, own := p.own[id]
	delete(p.own, id)
	if own && until > p.c.now() {
		p.mu.Unlock()
		return
	}
	var hints []any
	if p.lastPlan != nil {
		hints = pyval.List(pyval.Map(pyval.Map(p.lastPlan, "hints"), id), action)
	}
	p.mu.Unlock()
	verb := "opened"
	if action == "close" {
		verb = "closed"
	}
	for _, entry := range hints {
		hint := entry.(*pyjson.Object)
		p.c.emit("interlock_hint_observed", pyval.Obj("node_id", id, "action", action, "origin", origin,
			"code", pyval.Get(hint, "code"), "message", origin+" "+verb+" "+pyval.Str(pyval.Map(node, "data"), "label")+
				". "+pyval.Str(hint, "text")+" A real station's interlocking would usually stop this."))
	}
}

// EquipmentStates is GET /equipment/states: {"states": [...], "energization": {...}}.
func (c *Coordinator) EquipmentStates() (states []any, energization *pyjson.Object) {
	p := c.plant
	p.mu.Lock()
	defer p.mu.Unlock()
	states = []any{}
	if c.isLive() {
		for _, id := range p.order {
			states = append(states, pyval.Obj("node_id", id, "value", p.evaluate(id)))
		}
	}
	energization = p.plan()
	if energization == nil {
		energization = pyval.Obj("nodes", pyjson.NewObject(), "edges", pyjson.NewObject(), "hints", pyjson.NewObject())
	}
	return states, energization
}
