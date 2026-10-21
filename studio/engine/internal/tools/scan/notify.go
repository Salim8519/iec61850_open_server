package scan

import "sync/atomic"

// Change is the name this service reports through OnChange: the push stream's trigger of
// the topic that shows its status (internal/push).
const Change = "scan"

// changes holds the OnChange hook.
type changes struct {
	hook atomic.Pointer[func(name string)]
}

// OnChange sets the function called with Change after a change the page shows; nil removes
// it. It runs on the goroutine that made the change, never under a lock of this package,
// and must not block. Over-reporting is harmless: the push stream coalesces and sends only
// changed bytes.
func (c *changes) OnChange(fn func(name string)) {
	if fn == nil {
		c.hook.Store(nil)
		return
	}
	c.hook.Store(&fn)
}

func (c *changes) changed() {
	if fn := c.hook.Load(); fn != nil {
		(*fn)(Change)
	}
}
