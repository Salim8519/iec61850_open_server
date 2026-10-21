package runtime

// ReadNative is Supervisor.read for the packaged self-test (launcher/self_test.py reads
// runtime.supervisor.read): one read from the native worker of a simulated IED, on its actor.
func (c *Coordinator) ReadNative(iedID, ref string) (any, error) { return c.readNative(iedID, ref) }

// PendingPulses is len(runtime.pulses.pending): the pulses still waiting to restore their
// previous value.
func (c *Coordinator) PendingPulses() int {
	c.pulses.mu.Lock()
	defer c.pulses.mu.Unlock()
	return len(c.pulses.pending)
}
