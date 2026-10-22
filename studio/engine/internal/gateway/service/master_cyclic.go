package service

import (
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// cyclicSender repeats one command on a timer (Go-only, change 0169). Each send may step the
// address by a fixed amount and toggle the value, so an operator can exercise a point or sweep
// a range without clicking for every shot. Only one cycle runs at a time; Stop ends it, and a
// dropped link ends it too. Every shot still goes through Master.Command, so it is logged and
// warned about exactly like a hand-sent command.
type cyclicSender struct {
	master *Master
	mu     sync.Mutex
	cancel chan struct{}
	run    bool
	sends  int64
	every  int64
}

func newCyclic(master *Master) *cyclicSender { return &cyclicSender{master: master} }

// Start begins the cycle. interval is milliseconds (100-600000); step is added to the address
// after each shot; toggle flips an on/off value each shot. Starting replaces any running cycle.
func (c *cyclicSender) Start(command *pyjson.Object, interval, step int64, toggle bool) (*pyjson.Object, error) {
	if interval < 100 || interval > 600000 {
		return nil, apperr.ValueError("The cycle time is milliseconds from 100 to 600000")
	}
	if step < 0 || step > 65535 {
		return nil, apperr.ValueError("The address step is a whole number from 0 to 65535")
	}
	if _, err := c.master.activeProcess(); err != nil {
		return nil, err
	}
	c.Stop()
	cancel := make(chan struct{})
	c.mu.Lock()
	c.cancel, c.run, c.sends, c.every = cancel, true, 0, interval
	c.mu.Unlock()
	go c.loop(command, interval, step, toggle, cancel)
	c.master.changed(ChangeMasterStatus)
	return c.Snapshot(), nil
}

func (c *cyclicSender) loop(command *pyjson.Object, interval, step int64, toggle bool, cancel chan struct{}) {
	address, _ := pytext.Int64(get(command, "address"))
	base, _ := pytext.Float(get(command, "value"))
	kind := text(command, "kind")
	for shot := int64(0); ; shot++ {
		shotCommand := with(with(command, "address", address+step*shot), "value",
			toggledValue(kind, base, toggle && shot%2 == 1))
		if _, err := c.master.Command(shotCommand, "Cyclic \xc2\xb7 this page"); err != nil {
			break // the link dropped or the worker went away: end the cycle quietly
		}
		c.mu.Lock()
		c.sends++
		c.mu.Unlock()
		c.master.changed(ChangeMasterStatus)
		select {
		case <-cancel:
			return
		case <-time.After(time.Duration(interval) * time.Millisecond):
		}
	}
	c.stopOwn(cancel)
}

// stopOwn ends the cycle only if it is still the one this loop started, so a loop that fails
// after a restart never stops the newer cycle.
func (c *cyclicSender) stopOwn(cancel chan struct{}) {
	c.mu.Lock()
	own := c.run && c.cancel == cancel
	if own {
		c.run = false
		close(c.cancel)
	}
	c.mu.Unlock()
	if own {
		c.master.changed(ChangeMasterStatus)
	}
}

// Stop ends the running cycle; it is safe to call when nothing runs.
func (c *cyclicSender) Stop() {
	c.mu.Lock()
	if c.run {
		c.run = false
		close(c.cancel)
	}
	c.mu.Unlock()
	c.master.changed(ChangeMasterStatus)
}

// Snapshot is the cyclic state for the page: running, shots sent, and the cycle time.
func (c *cyclicSender) Snapshot() *pyjson.Object {
	c.mu.Lock()
	defer c.mu.Unlock()
	return pyjson.NewObject().Set("running", c.run).Set("sends", c.sends).Set("interval_ms", c.every)
}

// toggledValue flips an on/off command's value on alternate shots; other kinds keep their value.
func toggledValue(kind string, base float64, flip bool) any {
	if !flip {
		return numberValue(base)
	}
	switch kind {
	case "single", "coil", "binary_output":
		return numberValue(1 - base)
	case "double", "regulating":
		return numberValue(3 - base)
	}
	return numberValue(base)
}

// numberValue keeps a whole number an int (so 1, not 1.0, reaches the worker like the page sends).
func numberValue(v float64) any {
	if v == float64(int64(v)) {
		return int64(v)
	}
	return v
}
