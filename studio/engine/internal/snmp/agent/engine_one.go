package agent

import "studio-engine/internal/snmp/model"

// StartOne starts (or restarts) one device and leaves every other running device as it is
// (change 0234: a Run button per device).
func (e *Engine) StartOne(cfg model.Device) Started {
	e.mu.Lock()
	defer e.mu.Unlock()
	if old, found := e.devices[cfg.ID]; found {
		old.Close()
		delete(e.devices, cfg.ID)
	}
	e.boots++
	device, err := Start(cfg, e.boots, e.now, e.emit)
	if err != nil {
		return Started{ID: cfg.ID, Error: err.Error()}
	}
	e.devices[cfg.ID] = device
	return Started{ID: cfg.ID, OK: true}
}

// StopOne closes one device; the others keep running.
func (e *Engine) StopOne(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if device, found := e.devices[id]; found {
		device.Close()
		delete(e.devices, id)
	}
}
