package acquisition

import (
	"sync"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// GooseInputs drives simulated subscribers' GOOSE inputs from received frames (ADR-0066):
// an accepted frame becomes ordinary samples for the subscriber's worker.
type GooseInputs struct {
	deps   Deps
	mu     sync.Mutex
	routes map[string][]inputRoute
}

type inputRoute struct {
	iedID       string
	bindings    []goosedom.Binding
	memberCount int64
}

// NewGooseInputs returns an empty router.
func NewGooseInputs(deps Deps) *GooseInputs {
	return &GooseInputs{deps: deps, routes: map[string][]inputRoute{}}
}

func (g *GooseInputs) withMembers(ied, subscription *pyjson.Object) (*pyjson.Object, error) {
	complete := pyv.Has(subscription, "member_count")
	for _, item := range pyv.Objects(pyv.GetList(subscription, "inputs")) {
		complete = complete && pyv.Has(item, "member")
	}
	if complete || g.deps.Members == nil {
		return subscription, nil
	}
	return g.deps.Members(ied, subscription)
}

// Prepare binds inputs once per Run; each watch's subscribers record how many inputs apply.
// It runs under the monitor's lock (the watches are not published yet).
func (g *GooseInputs) Prepare(ieds []*pyjson.Object, watches *pyjson.Object) {
	byID := map[string]*pyjson.Object{}
	for _, ied := range ieds {
		byID[pyv.GetStr(ied, "id")] = ied
	}
	routes := map[string][]inputRoute{}
	binder := goosedom.NewBinder()
	for _, key := range watches.Keys() {
		watch := pyv.GetObj(watches, key)
		gocbRef := pyv.Get(watch, "gocb_ref")
		for _, subscriber := range pyv.Objects(pyv.GetList(watch, "subscribers")) {
			ied := byID[pyv.GetStr(subscriber, "ied_id")]
			var subscription *pyjson.Object
			for _, item := range pyv.Objects(pyv.GetList(pyv.OrObj(ied, "goose"), "subscribes")) {
				if pyv.Equal(pyv.Get(item, "gocb_ref"), gocbRef) {
					subscription = item
					break
				}
			}
			if subscription == nil {
				continue
			}
			subscription, err := g.withMembers(ied, subscription)
			if err != nil { // a stored source that no longer parses is reported, not fatal
				subscriber.Set("applied_inputs", int64(0)).Set("input_problems", []any{"Inputs not applied: " + err.Error()})
				continue
			}
			bindings, problems := binder.Bind(ied, subscription)
			if len(problems) > 20 {
				problems = problems[:20]
			}
			subscriber.Set("applied_inputs", len(bindings)).Set("input_problems", pyv.Strings(problems))
			if len(bindings) > 0 {
				count, _ := pyv.Int64(pyv.GetOr(subscription, "member_count", int64(0)))
				ref := pyv.PyStr(gocbRef)
				routes[ref] = append(routes[ref], inputRoute{iedID: pyv.GetStr(ied, "id"), bindings: bindings, memberCount: count})
			}
		}
	}
	g.mu.Lock()
	g.routes = routes
	g.mu.Unlock()
}

// Apply turns one received frame into samples for every subscriber bound to the block.
func (g *GooseInputs) Apply(watch, event *pyjson.Object) {
	gocbRef := pyv.GetStr(watch, "gocb_ref")
	g.mu.Lock()
	routes := g.routes[gocbRef]
	g.mu.Unlock()
	// A timeout keeps the last applied values, as a subscriber holds its inputs until told otherwise.
	if len(routes) == 0 || pyv.Equal(pyv.Get(event, "state"), "timeout") {
		return
	}
	publisherValues := g.publisherValues(watch, event)
	for _, route := range routes {
		if reason := goosedom.AcceptFrame(event, route.memberCount); reason != "" {
			g.deps.Events.Emit("goose_input_refused", pyv.O("ied_id", route.iedID, "gocb_ref", gocbRef, "reason", reason))
			continue
		}
		updates, problems := goosedom.InputUpdates(route.bindings, pyv.GetList(event, "values"), publisherValues)
		applied := []any{}
		if len(updates) > 0 && g.deps.Samples.Running(route.iedID) {
			if err := g.deps.Samples.Apply(route.iedID, toRefValues(updates), "goose_input"); err != nil {
				if !isValueError(err) {
					continue // Python lets other errors escape the event consumer
				}
				problems = append(problems, err.Error())
			} else {
				for _, update := range updates {
					applied = append(applied, update.Ref)
				}
			}
		}
		if len(problems) > 10 {
			problems = problems[:10]
		}
		g.deps.Events.Emit("goose_input", pyv.O("ied_id", route.iedID, "gocb_ref", gocbRef,
			"st_num", pyv.Get(event, "st_num"), "applied", applied, "problems", pyv.Strings(problems)))
	}
}

func (g *GooseInputs) publisherValues(watch, event *pyjson.Object) *pyjson.Object {
	publisher := pyv.OrObj(watch, "publisher")
	values, isList := pyv.Get(event, "values").([]any)
	if !pyv.Truthy(pyv.Get(publisher, "ied_id")) || !isList {
		return nil
	}
	ied := findIED(g.deps.Projects.Current(), pyv.GetStr(publisher, "ied_id"))
	control := publishedControl(ied, pyv.Get(watch, "gocb_ref"))
	if control == nil {
		return nil
	}
	return goosedom.DatasetValues(ied, control, values)
}
