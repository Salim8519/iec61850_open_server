package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/goosemsg"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// GOOSE send from the SLD (ADR-0112), POST /equipment/<node>/goose. Equipment with a goose_link
// sends {action: open|closed|on|off|value, value?}; an IED component sends {gocb_ref, ref, value}
// for one value of its own messages. All values of one action are written in ONE sample batch
// on the simulated IED's actor: the native worker applies a batch inside one data model lock and
// libiec61850 publishes pending GOOSE when the lock is released, so the block's stNum rises
// exactly once. Real devices are watched only; nothing is ever sent for them.

// gooseSource marks the writes in signal events and pauses a generator on those points.
const gooseSource = "goose_send"

// gooseWrite is the one batch write of a send (a variable only so tests can count the batches).
var gooseWrite = (*Coordinator).apply

// gooseSend is one parsed send: the IED, its block and the writes of one action.
type gooseSend struct {
	ied            *pyjson.Object
	gocbRef        string
	action         string
	writes         []goosemsg.Write
	requestedValue any
}

// SendGoose writes the values of one GOOSE send in one batch and logs what was asked and what
// the IED then held (goose_send_requested, goose_send_applied or goose_send_failed, with the
// node, IED, block and one correlation id; the run log adds the run id).
func (c *Coordinator) SendGoose(nodeID string, body *pyjson.Object) (*pyjson.Object, error) {
	project := c.opt.Project.Current()
	var node *pyjson.Object
	for _, candidate := range diagramNodes(project) {
		if pyval.Str(candidate, "id") == nodeID {
			node = candidate
			break
		}
	}
	if node == nil {
		return nil, apperr.New("Unknown equipment")
	}
	if !c.isLive() {
		return nil, apperr.New(goosemsg.NotRunning)
	}
	request, err := c.planGooseSend(project, pyval.Map(node, "data"), body)
	if err != nil {
		return nil, err
	}
	iedID := pyval.Str(request.ied, "id")
	refs, values, items := []any{}, []any{}, make([]item, 0, len(request.writes))
	for _, write := range request.writes {
		refs, values = append(refs, write.Ref), append(values, write.Value)
		items = append(items, item{ref: write.Ref, value: write.Value})
	}
	correlation := newHexID()
	event := pyval.Obj("node_id", nodeID, "ied_id", iedID, "gocb_ref", request.gocbRef, "action", request.action,
		"refs", refs, "values", values, "correlation_id", correlation, "origin", "studio_api")
	c.emit("goose_send_requested", event)
	written, err := gooseWrite(c, iedID, items, gooseSource)
	if err != nil {
		c.emit("goose_send_failed", copyWith(event, "message", errorText(err)))
		return nil, err
	}
	applied := pyjson.NewObject()
	for _, write := range request.writes {
		value, _ := written.Get(write.Ref)
		applied.Set(write.Ref, value)
	}
	c.emit("goose_send_applied", copyWith(event, "applied", applied))
	return pyval.Obj("node_id", nodeID, "ied_id", iedID, "gocb_ref", request.gocbRef, "action", request.action,
		"applied", applied, "correlation_id", correlation), nil
}

// planGooseSend checks the node, the IED and the body, and plans the writes.
func (c *Coordinator) planGooseSend(project, data, body *pyjson.Object) (gooseSend, error) {
	kind := pyval.Str(data, "kind")
	var link *goosemsg.Link
	iedID := pyval.Str(data, "ied_id")
	if kind == goosemsg.IEDKind {
		if iedID == "" {
			return gooseSend{}, apperr.New("This IED component has no IED yet. Choose one in its sheet while the IEDs are stopped.")
		}
	} else if link = goosemsg.ParseLink(pyval.Get(data, "goose_link")); link == nil {
		return gooseSend{}, apperr.New(pyval.Str(data, "label") + " has no GOOSE link. Add one in its sheet while the IEDs are stopped.")
	} else {
		iedID = link.IEDID
	}
	ied := findIed(project, iedID)
	if ied == nil {
		return gooseSend{}, apperr.Coded("Unknown IED", "not_found")
	}
	if err := c.gooseTarget(ied); err != nil {
		return gooseSend{}, err
	}
	catalog := goosemsg.For(ied)
	request := gooseSend{ied: ied, requestedValue: pyval.Get(body, "value")}
	if link == nil {
		request.gocbRef, request.action = pyval.Str(body, "gocb_ref"), "value"
		block := catalog.Block(request.gocbRef)
		if block == nil {
			return gooseSend{}, apperr.New(pyval.Str(ied, "name") + " does not send this GOOSE message. Choose one of its messages.")
		}
		writes, err := goosemsg.PlanValue(block, pyval.Str(body, "ref"), request.requestedValue)
		request.writes = writes
		return request, err
	}
	if err := link.Check(kind, catalog); err != nil {
		return gooseSend{}, err
	}
	request.gocbRef, request.action = link.GocbRef, pyval.Str(body, "action")
	writes, err := link.Plan(kind, catalog.Block(link.GocbRef), request.action, request.requestedValue)
	request.writes = writes
	return request, err
}

// gooseTarget refuses what must not send (goosemsg.Refusal): a real device (watched only), a
// skipped IED, an IED whose GOOSE is switched off in Devices, and an IED that is not running.
func (c *Coordinator) gooseTarget(ied *pyjson.Object) error {
	if refusal := goosemsg.Refusal(ied, c.simulatedRun(pyval.Str(ied, "id")) != nil); refusal != "" {
		return apperr.New(refusal)
	}
	return nil
}
