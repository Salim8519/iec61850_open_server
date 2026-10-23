package goosedom

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// liveFields is LIVE_FIELDS: the listener fields a declared link carries.
var liveFields = [...]string{"state", "problem", "st_num", "sq_num", "observed_at", "frames"}

// WithLive returns a copy of the link map where each declared link carries the
// listener's view of its block. A link is never marked received from the declaration
// alone: without a watch it has `live: None`, and while stopped every declared link
// reads "stopped". The map's other values are shared, not copied (like {**map}).
func WithLive(linkMap *pyjson.Object, watches []*pyjson.Object, running bool) *pyjson.Object {
	byRef := map[string]*pyjson.Object{}
	for _, watch := range watches {
		byRef[dictKey(pyv.Get(watch, "gocb_ref"))] = watch
	}
	links := []any{}
	for _, link := range pyv.Objects(pyv.GetList(linkMap, "links")) {
		var live any
		if pyv.Equal(pyv.Get(link, "status"), "declared") {
			watch := byRef[dictKey(pyv.Get(link, "gocb_ref"))]
			if !running {
				live = pyv.O("state", "stopped")
			} else if watch != nil && !equalsAny(pyv.Get(watch, "state"), nil, "stopped") {
				// While running, "stopped" means the listener is off (no lab mode or no Npcap): not watched.
				view := pyjson.NewObject()
				for _, key := range liveFields {
					if value := pyv.Get(watch, key); value != nil {
						view.Set(key, value)
					}
				}
				live = view
			}
		}
		links = append(links, pyv.Merge(link).Set("live", live))
	}
	outside := map[string]bool{}
	for _, ied := range pyv.Objects(pyv.GetList(linkMap, "ieds")) {
		if pyv.Equal(pyv.Get(ied, "in_project"), false) {
			outside[pyv.PyStr(pyv.Get(ied, "name"))] = true
		}
	}
	result := pyv.Merge(linkMap)
	return result.Set("links", links).Set("running", running).Set("summary", Summary(pyv.Objects(links), outside))
}

// Summary returns, per IED name in first-seen order, how many links it sends, whom it
// gets GOOSE from, and what is not sent, remote, not heard or in problem. A link whose
// publisher is outside the project (in the file but not imported) that is not heard counts
// as remote, not as "not heard": nothing in the project sends it (ADR-0114).
func Summary(links []*pyjson.Object, outside map[string]bool) *pyjson.Object {
	result := pyjson.NewObject()
	entry := func(name any) *pyjson.Object {
		key := pyv.PyStr(name)
		if existing, found := result.Get(key); found {
			return existing.(*pyjson.Object)
		}
		created := pyv.O("sends_to", int64(0), "gets_from", []any{}, "not_sent", int64(0), "remote", int64(0),
			"not_heard", int64(0), "problems", int64(0))
		result.Set(key, created)
		return created
	}
	for _, link := range links {
		publisherName := pyv.Get(link, "publisher")
		publisher, subscriber := entry(publisherName), entry(pyv.Get(link, "subscriber"))
		status := pyv.Get(link, "status")
		if pyv.Equal(status, "declared") {
			increment(publisher, "sends_to")
		}
		if !contains(pyv.GetList(subscriber, "gets_from"), publisherName) {
			subscriber.Set("gets_from", append(pyv.GetList(subscriber, "gets_from"), publisherName))
		}
		if pyv.Equal(status, "remote") {
			increment(subscriber, "remote")
		} else if !pyv.Equal(status, "declared") {
			increment(subscriber, "not_sent")
		}
		state := pyv.Get(pyv.Obj(pyv.Get(link, "live")), "state")
		if equalsAny(state, "waiting", "timeout") && outside[pyv.PyStr(publisherName)] && pyv.Equal(status, "declared") {
			increment(subscriber, "remote")
		} else if equalsAny(state, "waiting", "timeout") {
			increment(subscriber, "not_heard")
		} else if pyv.Equal(state, "problem") {
			increment(subscriber, "problems")
		}
	}
	return result
}

func increment(object *pyjson.Object, key string) {
	count, _ := pyv.Int64(pyv.Get(object, key))
	object.Set(key, count+1)
}

// contains is Python's `value in items`.
func contains(items []any, value any) bool {
	for _, item := range items {
		if pyv.Equal(item, value) {
			return true
		}
	}
	return false
}
