package runtime

// SetDeadbands turns the "Use SCD deadbands" switch of a running IED on or off at once
// (ADR-0118); the project keeps the setting for the next Run.
func (c *Coordinator) SetDeadbands(iedID string, on bool) {
	c.mu.Lock()
	item := c.runs[iedID]
	c.mu.Unlock()
	if item != nil {
		item.bands.Store(on)
	}
}
