package service

import "sync/atomic"

// Change names the gateway and the master report through OnChange. They are the push
// stream's topic triggers (internal/push): a page showing that state re-reads it.
const (
	ChangeGatewayStatus   = "gateway.status"
	ChangeGatewayPoints   = "gateway.points"
	ChangeGatewayCommands = "gateway.commands"
	ChangeGatewayTraffic  = "gateway.traffic"
	ChangeMasterStatus    = "master.status"
	ChangeMasterPoints    = "master.points"
	ChangeMasterCommands  = "master.commands"
	ChangeMasterTraffic   = "master.traffic"
)

// changes holds the OnChange hook of a Manager or Master.
type changes struct {
	hook atomic.Pointer[func(name string)]
}

// OnChange sets the function called after a change a page shows (status, points, logs);
// nil removes it. It is called on the goroutine that made the change, never under a lock
// of this package, and must not block. Over-reporting is harmless: the push stream
// coalesces and sends only changed bytes.
func (c *changes) OnChange(fn func(name string)) {
	if fn == nil {
		c.hook.Store(nil)
		return
	}
	c.hook.Store(&fn)
}

func (c *changes) changed(name string) {
	if fn := c.hook.Load(); fn != nil {
		(*fn)(name)
	}
}
