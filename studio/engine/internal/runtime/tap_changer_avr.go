package runtime

import (
	"math"
	"sync"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/runtime/tapchanger"
)

// tapChangers runs the simulated voltage regulators (90) of the transformers (ADR-0109). Every
// cycle (0.1 s) it computes each regulated transformer's LV voltage = source x ratio(tap),
// writes it to a simulated LV voltage point, and in Auto sends one raise or lower after the
// delay while the voltage stays out of band. It never commands a real IED: a regulator whose
// points are not all on running simulated IEDs is blocked (logged once).
type tapChangers struct {
	c     *Coordinator
	cycle sync.Mutex // one advance at a time (the cadence tick and an Auto/Manual request)
	mu    sync.Mutex
	nodes []tapNode
	runs  map[string]*tapRun
}

// tapRun is the run state of one transformer.
type tapRun struct {
	regulator tapchanger.Regulator
	auto      bool // the regulator's own mode (used when no Auto/Manual status is bound)
	busy      bool // a regulator command is in flight
	written   any  // last LV voltage written to the IED (nil: none)
	blocked   string
	last      *pyjson.Object // last published "<node>:avr" value
}

func newTapChangers(c *Coordinator) *tapChangers {
	return &tapChangers{c: c, runs: map[string]*tapRun{}}
}

// reset reads the transformers of the run's diagram; every regulator starts in Manual.
func (t *tapChangers) reset(diagram *pyjson.Object) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes, t.runs = tapNodes(diagram), map[string]*tapRun{}
	for _, node := range t.nodes {
		t.runs[node.id] = &tapRun{}
	}
}

// autoMode is the transformer's mode: its bound status, or the regulator's own mode.
func (t *tapChangers) autoMode(node tapNode) (auto, known bool) {
	if node.mode.Set() {
		return modeOf(node, t.c.plant.read(bindingOf(node.mode)))
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if run := t.runs[node.id]; run != nil {
		return run.auto, true
	}
	return false, true
}

// setAuto changes the regulator's own mode and logs the decision.
func (t *tapChangers) setAuto(node tapNode, auto bool, correlation string) {
	t.mu.Lock()
	if run := t.runs[node.id]; run != nil {
		run.auto = auto
	}
	t.mu.Unlock()
	event := "mode_manual"
	if auto {
		event = "mode_auto"
	}
	t.c.emit("avr_decision", pyval.Obj("node_id", node.id, "ied_id", node.commandIED(), "event", event,
		"correlation_id", correlation, "origin", "studio_api", "message", node.label))
	t.advance()
}

// blockedReason is why a regulator cannot act now ("" when it can): every point it uses must be
// on a running simulated IED.
func (t *tapChangers) blockedReason(node tapNode) string {
	points := []tapchanger.Binding{node.tap, node.commands.Step, node.commands.Raise, node.commands.Lower, node.voltage}
	for _, point := range points {
		if !point.Set() {
			continue
		}
		if ied := findIed(t.c.opt.Project.Current(), point.IedID); ied != nil && roleOf(ied) == "real" {
			return "real_ied"
		}
		if t.c.simulatedRun(point.IedID) == nil {
			return "not_running"
		}
	}
	return ""
}

// advance is one cycle of every regulator.
func (t *tapChangers) advance() {
	t.cycle.Lock()
	defer t.cycle.Unlock()
	t.mu.Lock()
	nodes := t.nodes
	t.mu.Unlock()
	for _, node := range nodes {
		t.regulate(node)
	}
}

func (t *tapChangers) regulate(node tapNode) {
	t.mu.Lock()
	run := t.runs[node.id]
	t.mu.Unlock()
	if run == nil {
		return
	}
	auto, _ := t.autoMode(node)
	if !node.settings.Enabled {
		t.publish(node, run, pyval.Obj("auto", auto, "regulator", false))
		return
	}
	if reason := t.blockedReason(node); reason != "" {
		if run.blocked != reason {
			run.blocked = reason
			t.c.emit("avr_decision", pyval.Obj("node_id", node.id, "ied_id", node.commandIED(), "event", "blocked",
				"reason", reason, "correlation_id", newHexID(), "origin", "avr", "message", node.label))
		}
		t.publish(node, run, pyval.Obj("auto", auto, "regulator", true, "state", "blocked", "reason", reason))
		return
	}
	run.blocked = ""
	tap, known := t.c.readTap(node)
	lv := node.settings.LVVoltage(tap)
	if known && node.voltage.Set() && !pyval.Equal(run.written, lv) {
		run.written = lv // one try per new value, so a refused point is not written every cycle
		if _, err := t.c.apply(node.voltage.IedID, []item{{node.voltage.Ref, lv, nil}}, "tap_changer"); err != nil {
			t.c.emit("avr_decision", pyval.Obj("node_id", node.id, "ied_id", node.voltage.IedID, "event",
				"voltage_write_failed", "ref", node.voltage.Ref, "lv_kv", lv, "correlation_id", newHexID(),
				"origin", "avr", "message", node.label+": "+errorText(err)))
		}
	}
	t.mu.Lock()
	busy := run.busy
	t.mu.Unlock()
	decision := tapchanger.Decision{State: "sending"} // a regulator step is in flight
	if !busy {
		decision = run.regulator.Step(node.settings, t.c.now(), tap, known, lv, auto)
	}
	if decision.Event != "" {
		t.decide(node, run, decision, tap, known, lv)
	}
	state := pyval.Obj("auto", auto, "regulator", true, "state", decision.State,
		"remaining_s", math.Ceil(decision.RemainingS), "deviation_pct", math.Round(decision.DeviationPct*10)/10)
	if known {
		state.Set("lv_kv", lv)
	}
	t.publish(node, run, state)
}

// decide logs one regulator decision and, for a step, sends it without holding up the cycle.
func (t *tapChangers) decide(node tapNode, run *tapRun, decision tapchanger.Decision, tap int64, known bool, lv float64) {
	correlation := newHexID()
	event := pyval.Obj("node_id", node.id, "ied_id", node.commandIED(), "event", decision.Event,
		"state", decision.State, "action", decision.Action, "tap", knownTap(tap, known),
		"lv_kv", lv, "target_kv", node.settings.TargetKV, "deviation_pct", decision.DeviationPct,
		"correlation_id", correlation, "origin", "avr", "message", node.label)
	t.c.emit("avr_decision", event)
	if decision.Action == "" {
		return
	}
	t.mu.Lock()
	run.busy = true
	t.mu.Unlock()
	go func() {
		defer func() {
			t.mu.Lock()
			run.busy = false
			t.mu.Unlock()
		}()
		_, _ = t.c.tapStep(node, decision.Action, "avr", correlation) // logged as applied or failed
	}()
}

// publish emits "<node>:avr" when its value changed.
func (t *tapChangers) publish(node tapNode, run *tapRun, value *pyjson.Object) {
	t.mu.Lock()
	same := run.last != nil && pyval.Equal(run.last, value)
	if !same {
		run.last = value
	}
	t.mu.Unlock()
	if !same {
		t.c.emit("equipment_state", pyval.Obj("node_id", node.id+":avr", "value", value))
	}
}

// states are the last "<node>:avr" values, for GET /equipment/states.
func (t *tapChangers) states() []any {
	t.mu.Lock()
	defer t.mu.Unlock()
	var states []any
	for _, node := range t.nodes {
		if run := t.runs[node.id]; run != nil && run.last != nil {
			states = append(states, pyval.Obj("node_id", node.id+":avr", "value", run.last))
		}
	}
	return states
}
