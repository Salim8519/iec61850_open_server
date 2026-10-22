package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// pointPanelHealthy sets every status, double and value point that has a healthy value in one
// request and reports each point: written (with the value read back) or why it was not.
// Points that cannot be written do not stop the others.
func (c *Coordinator) pointPanelHealthy(nodeID string, ied *pyjson.Object, points []panelPoint) (*pyjson.Object, error) {
	var chosen []panelPoint
	for _, point := range points {
		if point.healthy != nil && point.kind != "command" {
			chosen = append(chosen, point)
		}
	}
	if len(chosen) == 0 {
		return nil, apperr.New("No point of this panel has a healthy value yet. Set the healthy values " +
			"in the panel's editor while the IEDs are stopped.")
	}
	target, err := c.panelTarget(ied)
	if err != nil {
		return nil, err
	}
	iedID, correlation := pyval.Str(ied, "id"), newHexID()
	c.emit("point_panel_requested", pyval.Obj("node_id", nodeID, "ied_id", iedID, "point_type", "healthy",
		"requested", int64(len(chosen)), "correlation_id", correlation, "origin", "studio_api"))
	results := make([]*pyjson.Object, len(chosen))
	var writes []item
	var owners []int
	for index, point := range chosen {
		write, err := pointWrite(target, point, point.healthy)
		if err != nil {
			results[index] = failedPoint(point, err)
			continue
		}
		writes, owners = append(writes, write), append(owners, index)
	}
	for start := 0; start < len(writes); start += panelBatch {
		end := min(start+panelBatch, len(writes))
		written, err := c.apply(iedID, writes[start:end], "equipment_button")
		for offset, write := range writes[start:end] {
			index := owners[start+offset]
			var actual any
			found := false
			if written != nil {
				actual, found = written.Get(write.ref)
			}
			if err != nil || !found {
				results[index] = failedPoint(chosen[index], err)
				continue
			}
			results[index] = pyval.Obj("row", chosen[index].id, "ok", true, "value", actual)
		}
	}
	applied := int64(0)
	list := make([]any, len(results))
	for index, result := range results {
		list[index] = result
		if ok, _ := result.Get("ok"); ok == true {
			applied++
		}
	}
	failed := int64(len(results)) - applied
	summary := pyval.Obj("node_id", nodeID, "ied_id", iedID, "point_type", "healthy", "requested", int64(len(chosen)),
		"applied", applied, "failed", failed, "correlation_id", correlation, "origin", "studio_api")
	c.emit("point_panel_applied", copyWith(summary, "message", "All healthy"))
	return copyWith(summary, "action", "healthy", "results", list), nil
}

// failedPoint is the report of a point that was not written.
func failedPoint(point panelPoint, err error) *pyjson.Object {
	message := "The IED did not confirm the new value."
	if err != nil {
		message = errorText(err)
	}
	return pyval.Obj("row", point.id, "ok", false, "error", message)
}
