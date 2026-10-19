package acquisition

import (
	"errors"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

var (
	rxFields       = []string{"state", "problem", "st_num", "sq_num", "frames", "origin", "src_mac"}
	receivedFields = []string{"conf_rev", "go_id", "dataset", "test"}
	unknownFields  = []string{"gocb_ref", "app_id", "mac", "src_mac", "vlan_id", "dataset", "go_id", "conf_rev", "st_num", "test"}
)

// onEvent handles studio_live GOOSE events (goose_monitor.py:43-78).
func (g *GooseMonitor) onEvent(event *pyjson.Object) {
	switch pyv.GetStr(event, "event") {
	case "goose_rx":
		g.received(event)
	case "goose_stats":
		g.mu.Lock()
		for _, item := range pyv.Objects(pyv.GetList(event, "watches")) {
			watch := pyv.GetObj(g.watches, pyv.GetStr(item, "key"))
			if watch == nil {
				continue
			}
			// Retransmissions raise sqNum without a state change; the stats carry it (change 0091).
			watch.Set("frames", pyv.Get(item, "frames")).Set("age_ms", pyv.Get(item, "age_ms"))
			if pyv.Has(item, "sq_num") {
				watch.Set("sq_num", pyv.Get(item, "sq_num"))
			}
		}
		g.mu.Unlock()
	case "goose_unknown":
		entry := pyjson.NewObject()
		for _, key := range unknownFields {
			entry.Set(key, pyv.Get(event, key))
		}
		g.mu.Lock()
		g.status = pyv.Shallow(g.status)
		g.status.Set("unknown", append(append([]any{}, pyv.GetList(g.status, "unknown")...), entry))
		g.mu.Unlock()
		g.deps.Events.Emit("goose_unknown", pyv.O("gocb_ref", pyv.Get(event, "gocb_ref"), "app_id", pyv.Get(event, "app_id")))
	case "goose_unknown_cap":
		g.mu.Lock()
		g.status = pyv.Shallow(g.status)
		g.status.Set("unknown_limit", pyv.Get(event, "limit"))
		g.mu.Unlock()
	case "worker_failed":
		g.mu.Lock()
		g.status = pyv.Shallow(g.status)
		g.status.Set("state", "failed").Set("message", "The GOOSE listener stopped unexpectedly")
		g.mu.Unlock()
		message := pyv.Get(event, "error")
		if !pyv.Truthy(message) {
			message = ""
		}
		g.deps.Events.Emit("goose_listener_failed", pyv.O("message", message))
	}
}

func (g *GooseMonitor) received(event *pyjson.Object) {
	key, _ := pyv.Get(event, "key").(string)
	g.mu.Lock()
	watch := pyv.GetObj(g.watches, key)
	if watch == nil {
		g.mu.Unlock()
		return
	}
	for _, name := range rxFields {
		if value, found := event.Get(name); found {
			watch.Set(name, value)
		}
	}
	// Received header fields are kept apart from the expected SCD values of the watch.
	for _, name := range receivedFields {
		if value, found := event.Get(name); found {
			watch.Set("received_"+name, value)
		}
	}
	watch.Set("observed_at", pyv.ISOSeconds(g.deps.Clock.Now()))
	view := pyv.Shallow(watch)
	g.mu.Unlock()
	payload := pyjson.NewObject()
	for _, name := range event.Keys() {
		if name != "event" && name != "values" {
			value, _ := event.Get(name)
			payload.Set(name, value)
		}
	}
	g.deps.Events.Emit("goose_status", payload)
	g.route(view, event)
	g.inputs.Apply(view, event)
}

// route writes LGOS supervision points of simulated subscribers and live values of a real
// publisher.
func (g *GooseMonitor) route(watch, event *pyjson.Object) {
	project := g.deps.Projects.Current()
	for _, subscriber := range pyv.Objects(pyv.GetList(watch, "subscribers")) {
		iedID := pyv.GetStr(subscriber, "ied_id")
		var index map[string]*pyjson.Object
		if g.deps.Values.Has(iedID) {
			if ied := findIED(project, iedID); ied != nil {
				index = g.refs.For(ied)
			}
		}
		updates := goosedom.LgosUpdates(pyv.Get(subscriber, "lgos"), event, index)
		if len(updates) == 0 || !g.deps.Samples.Running(iedID) {
			continue
		}
		if err := g.deps.Samples.Apply(iedID, toRefValues(updates), "goose_supervision"); err != nil && isValueError(err) {
			g.deps.Events.Emit("goose_supervision_failed", pyv.O("ied_id", iedID, "message", err.Error()))
		}
	}
	publisher := pyv.OrObj(watch, "publisher")
	if !pyv.Equal(pyv.Get(publisher, "role"), "real") || !pyv.Has(event, "values") {
		return
	}
	ied := findIED(project, pyv.GetStr(publisher, "ied_id"))
	control := publishedControl(ied, pyv.Get(watch, "gocb_ref"))
	if control == nil {
		return
	}
	if mapped := goosedom.DatasetValues(ied, control, pyv.Get(event, "values")); pyv.Truthy(mapped) {
		_, _ = g.values.Apply(pyv.GetStr(ied, "id"), mapped, "live_goose")
	}
}

func publishedControl(ied *pyjson.Object, gocbRef any) *pyjson.Object {
	for _, control := range pyv.Objects(pyv.GetList(pyv.OrObj(ied, "goose"), "publishes")) {
		if pyv.Equal(pyv.Get(control, "gocb_ref"), gocbRef) {
			return control
		}
	}
	return nil
}

func toRefValues(updates []goosedom.Update) []RefValue {
	result := make([]RefValue, len(updates))
	for index, update := range updates {
		result[index] = RefValue{Ref: update.Ref, Value: update.Value}
	}
	return result
}

// isValueError reports a Python ValueError (the only error the Python callers catch).
func isValueError(err error) bool {
	var studio *apperr.Error
	return errors.As(err, &studio) && studio.Kind == apperr.Value
}
