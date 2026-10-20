// Package runtime is the Go engine's simulation runtime (stream C, P1): Run/Stop of simulated
// IEDs, per-IED actors owning the native workers, sources, pulses, scenarios, equipment
// control and feedback, energisation and the run log, with the events of the Python
// Coordinator (simulation/coordinator.py and friends): same kinds, fields and per-IED order.
package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"studio-engine/internal/pyjson"
)

// Options wires a Coordinator to its surroundings.
type Options struct {
	NativeDir    string // folder with studio_ied.exe, studio_control.exe, ...
	DataDir      string // runtime/<ied>.cfg and logs/ live here
	Bus          Bus
	Project      Project
	Live         Live
	// LabAddresses lists the IPv4 addresses on the lab adapter (ip_helper.addresses_on); its
	// error fails the Run like the OSError Python raised.
	LabAddresses func(adapterGUID string) ([]string, error)
	// CommandKeep is how long a canvas command's MMS association stays open for the next
	// command to the same IED; 0 closes it after each command like studio_probe did.
	CommandKeep time.Duration
}

type sourceKey struct{ ied, ref string }

// source is one automatic source of the run; spec is never mutated once stored.
type source struct {
	spec       *pyjson.Object
	overridden bool
}

// Coordinator owns one simulation run.
type Coordinator struct {
	opt   Options
	epoch time.Time

	lifeMu sync.Mutex // Run, Stop, Pause and the cadence tick never overlap (cycle_lock)
	mu     sync.Mutex // the fields below; never held across I/O
	state  string
	active map[string]bool
	runs   map[string]*running
	srcs   map[sourceKey]*source
	// srcOrder is the insertion order of srcs (Python dict order).
	srcOrder []sourceKey
	// elapsed and lastAutomation are the automation clock in seconds (coordinator.py).
	elapsed, lastAutomation float64
	elapsedFloat            bool
	cadence                 *cadence
	iedStates               map[string]string
	gooseStates             map[string]any
	// endpoints is supervisor.endpoints: where each started worker confirmed it listens.
	endpoints map[string]endpoint
	// index is coordinator.ied_indexes (and index): the value tables of the IEDs of the last
	// Run, kept after Stop as Python kept them, replaced at the next Run.
	index map[string]*valueTable
	hub   hub

	emitMu sync.Mutex
	log    *RunLog
	// commandMu orders a canvas command before the native feedback of that command.
	commandMu sync.Mutex

	tables    tables
	pulses    *pulses
	scenarios *scenarioRunner
	plant     *plantStates
	feedback  *feedback
	access    *mmsAccess
	commands  *commandClient
	shutdown  chan struct{}
	ticker    sync.WaitGroup
}

// New creates a stopped coordinator and starts its cadence goroutine.
func New(opt Options) *Coordinator {
	if opt.Live == nil {
		opt.Live = NoLive{}
	}
	if opt.LabAddresses == nil {
		opt.LabAddresses = func(string) ([]string, error) { return nil, nil }
	}
	c := &Coordinator{opt: opt, epoch: time.Now(), state: "stopped", active: map[string]bool{},
		runs: map[string]*running{}, srcs: map[sourceKey]*source{}, iedStates: map[string]string{},
		gooseStates: map[string]any{}, endpoints: map[string]endpoint{}, index: map[string]*valueTable{},
		shutdown: make(chan struct{})}
	c.tables.hub = &c.hub
	c.cadence = newCadence(c.now())
	c.lastAutomation = c.now()
	c.pulses = newPulses(c)
	c.scenarios = newScenarioRunner(c)
	c.plant = newPlantStates(c)
	c.feedback = newFeedback(c)
	c.access = newMmsAccess(c)
	c.commands = newCommandClient(filepath.Join(opt.NativeDir, "studio_control.exe"), opt.CommandKeep)
	c.ticker.Add(1)
	go c.tick()
	return c
}

// now is time.monotonic() in seconds.
func (c *Coordinator) now() float64 { return time.Since(c.epoch).Seconds() }

// emit publishes an event and appends it to the run log while a run is open (Coordinator.emit).
func (c *Coordinator) emit(kind string, payload *pyjson.Object) *pyjson.Object {
	c.emitMu.Lock()
	defer c.emitMu.Unlock()
	event := c.opt.Bus.Publish(kind, payload)
	if c.log != nil && event != nil {
		c.log.Write(event)
	}
	return event
}

// publish sends an event to the bus only (never the run log), as mms_access and project_changed.
func (c *Coordinator) publish(kind string, payload *pyjson.Object) {
	c.opt.Bus.Publish(kind, payload)
}

func (c *Coordinator) setLog(log *RunLog) *RunLog {
	c.emitMu.Lock()
	defer c.emitMu.Unlock()
	previous := c.log
	c.log = log
	return previous
}

// State is the run state: stopped, starting, running, paused or stopping.
func (c *Coordinator) State() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// Running returns the sorted ids of the IEDs in the run (sorted(coordinator.active)).
func (c *Coordinator) Running() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sortedActive()
}

func (c *Coordinator) sortedActive() []string {
	ids := make([]string, 0, len(c.active))
	for id := range c.active {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (c *Coordinator) live() bool { return c.state == "running" || c.state == "paused" }

// elapsedValue is self.elapsed as Python holds it: the int 0 until time is first added.
func (c *Coordinator) elapsedValue() any {
	if !c.elapsedFloat {
		return int64(0)
	}
	return c.elapsed
}

// addElapsed is self.elapsed += delta.
func (c *Coordinator) addElapsed(delta float64) {
	c.elapsed += delta
	c.elapsedFloat = true
}

// IedState is the runtime-only "state" Python kept on the IED dict ("" when never set, so the
// project's stored value applies) and its "goose_state" (nil when absent).
func (c *Coordinator) IedState(iedID string) (string, any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.iedStates[iedID], c.gooseStates[iedID]
}

// ResetValues forgets the runtime values of an IED that was imported again or removed.
func (c *Coordinator) ResetValues(iedID string) { c.tables.reset(iedID) }

// Close stops the run and every background goroutine (coordinator.close).
func (c *Coordinator) Close() {
	select {
	case <-c.shutdown:
		return
	default:
	}
	close(c.shutdown)
	c.ticker.Wait()
	_ = c.Stop(nil)
	c.feedback.close()
	c.pulses.close()
	c.commands.close()
}

func newHexID() string {
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	raw[6] = raw[6]&0x0f | 0x40
	raw[8] = raw[8]&0x3f | 0x80
	return hex.EncodeToString(raw[:])
}

func sortedKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
