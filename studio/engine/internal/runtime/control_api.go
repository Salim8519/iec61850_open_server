package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// KnownIed is ProjectService.find_ied's check: StudioError "Unknown IED" (not_found).
func (c *Coordinator) KnownIed(iedID string) error {
	if findIed(c.opt.Project.Current(), iedID) == nil {
		return apperr.Coded("Unknown IED", "not_found")
	}
	return nil
}

// OperateControl is POST /controls/operate (api/signals.py operate_control): an external-style
// MMS command marked as ScadaBench's own, through the same canvas command path.
func (c *Coordinator) OperateControl(iedID, ref string, value any) (*pyjson.Object, error) {
	ied := findIed(c.opt.Project.Current(), iedID)
	if ied == nil {
		return nil, apperr.Coded("Unknown IED", "not_found")
	}
	c.access.markOwn(iedID, ref)
	c.mu.Lock()
	item, state := c.runs[iedID], c.iedStates[iedID]
	c.mu.Unlock()
	if item == nil || item.worker == nil {
		return nil, apperr.ValueError("IED is not running")
	}
	return c.commands.operate(ied, state, ref, value, item.host, item.port)
}
