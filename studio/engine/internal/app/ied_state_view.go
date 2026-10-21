package app

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime"
)

// Python kept an IED's runtime-only "state" ("running", "real_live", "failed", ...) and
// "goose_state" on the project's IED dict, so every reader of project["ieds"] saw them.
// The Go runtime keeps them in its own overlay (Coordinator.IedStates); the readers that
// need them get the IEDs through the views below.

// iedStates is the runtime's overlay source (*runtime.Coordinator).
type iedStates interface {
	IedStates() map[string]runtime.IEDOverlay
}

// applyOverlay sets the overlay of ied (looked up by its id) on ied itself.
func applyOverlay(ied *pyjson.Object, overlay map[string]runtime.IEDOverlay) {
	id, _ := ied.Get("id")
	text, _ := id.(string)
	state, known := overlay[text]
	if !known {
		return
	}
	if state.State != "" {
		ied.Set("state", state.State)
	}
	if state.HasGoose {
		ied.Set("goose_state", state.GooseState)
	} else {
		ied.Delete("goose_state")
	}
}

// gatewayProjects is the project service as gateway/service.Projects whose IEDs carry the
// runtime state: the gateway's point values (ied_online) and its command check ("is not
// running; press Run first") read ied["state"].
type gatewayProjects struct {
	projectIEDs
	states iedStates
}

// projectIEDs is the part of the project service the gateway uses.
type projectIEDs interface {
	IEDs() []*pyjson.Object
	Gateway() *pyjson.Object
	SaveGateway(config *pyjson.Object) error
}

// IEDs returns the project's IEDs; an IED the runtime knows is a shallow copy with its
// overlay applied (the published project is immutable and never changed here).
func (p gatewayProjects) IEDs() []*pyjson.Object {
	ieds := p.projectIEDs.IEDs()
	overlay := p.states.IedStates()
	if len(overlay) == 0 {
		return ieds
	}
	result := make([]*pyjson.Object, len(ieds))
	for i, ied := range ieds {
		result[i] = ied
		if ied == nil {
			continue
		}
		id, _ := ied.Get("id")
		if text, _ := id.(string); text == "" {
			continue
		} else if _, known := overlay[text]; !known {
			continue
		}
		copied := pyjson.NewObject()
		for _, key := range ied.Keys() {
			value, _ := ied.Get(key)
			copied.Set(key, value)
		}
		applyOverlay(copied, overlay)
		result[i] = copied
	}
	return result
}
