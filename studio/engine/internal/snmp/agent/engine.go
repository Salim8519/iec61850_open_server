package agent

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"studio-engine/internal/snmp/model"
)

// bootsEpoch makes engine boots grow across restarts without storing them: a start's boots
// are the seconds since 2020 (valid until 2088), plus one per start within this process.
var bootsEpoch = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// Engine runs the simulated devices of one studio_snmp process.
type Engine struct {
	mu      sync.Mutex
	devices map[string]*Device
	emit    Emit
	now     func() time.Time
	boots   int32
}

// NewEngine returns an engine with no devices running.
func NewEngine(emit Emit, now func() time.Time) *Engine {
	if now == nil {
		now = time.Now
	}
	return &Engine{devices: map[string]*Device{}, emit: emit, now: now,
		boots: int32(now().Sub(bootsEpoch) / time.Second)}
}

// Started is the outcome of starting one device.
type Started struct {
	ID    string `json:"id"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// Start replaces every running device with devices; one that cannot start does not stop the others.
func (e *Engine) Start(devices []model.Device) []Started {
	e.Stop()
	e.mu.Lock()
	defer e.mu.Unlock()
	results := make([]Started, 0, len(devices))
	for _, cfg := range devices {
		e.boots++
		device, err := Start(cfg, e.boots, e.now, e.emit)
		if err != nil {
			results = append(results, Started{ID: cfg.ID, Error: err.Error()})
			continue
		}
		e.devices[cfg.ID] = device
		results = append(results, Started{ID: cfg.ID, OK: true})
	}
	return results
}

// Stop closes every device.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for id, device := range e.devices {
		device.Close()
		delete(e.devices, id)
	}
}

// Running lists the ids of the running devices.
func (e *Engine) Running() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	ids := make([]string, 0, len(e.devices))
	for id := range e.devices {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (e *Engine) device(id string) (*Device, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if device := e.devices[id]; device != nil {
		return device, nil
	}
	return nil, fmt.Errorf("the device is not running; press Start first")
}

// Points is a running device's points with their current values.
func (e *Engine) Points(id string) ([]model.Point, error) {
	device, err := e.device(id)
	if err != nil {
		return nil, err
	}
	return device.Points(), nil
}

// SetValue changes one point of a running device.
func (e *Engine) SetValue(id, oid, value string, isHex bool) (model.Point, error) {
	device, err := e.device(id)
	if err != nil {
		return model.Point{}, err
	}
	return device.SetValue(oid, value, isHex)
}

// SendTrap sends a notification from a running device.
func (e *Engine) SendTrap(id string, trap Trap) ([]Result, error) {
	device, err := e.device(id)
	if err != nil {
		return nil, err
	}
	return device.SendTrap(trap)
}
