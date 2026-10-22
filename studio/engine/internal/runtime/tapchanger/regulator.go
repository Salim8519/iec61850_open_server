package tapchanger

import "math"

// Actions and the regulator's states and events. A state is what the transformer shows now; an
// event is a decision worth one runtime log line (it is reported only when it happens).
const (
	Raise = "raise"
	Lower = "lower"

	StateManual     = "manual"      // Manual mode: the regulator does nothing
	StateInBand     = "in_band"     // the LV voltage is inside the band
	StateTiming     = "timing"      // out of band; waits for the delay before one step
	StateAtLimit    = "at_limit"    // out of band, but the tap is already at its end
	StateNoPosition = "no_position" // the tap position is not known yet

	EventManual     = "manual"       // switched to Manual: a running delay is cancelled
	EventInBand     = "back_in_band" // came back inside the band: the delay is cancelled
	EventOutOfBand  = "out_of_band"  // left the band (or turned the other way): the delay starts
	EventStep       = "step"         // the delay passed: one raise or lower is sent
	EventAtMaxTap   = "at_max_tap"   // would raise, but the tap is at the highest tap
	EventAtMinTap   = "at_min_tap"   // would lower, but the tap is at the lowest tap
	EventNoPosition = "no_position"  // the tap position is unknown: nothing is decided
)

// Decision is one regulator step: the action to send ("" for none), the event to log ("" for
// none), the state to show, how far the voltage is from the target and the seconds left.
type Decision struct {
	Action, Event, State string
	DeviationPct         float64
	RemainingS           float64
}

// Regulator is the simulated automatic voltage regulator (90) of one transformer. It keeps the
// running delay between steps; Step is called every cycle with the time in seconds.
type Regulator struct {
	timing    bool
	direction string
	since     float64
	last      string // last state, so a repeated state logs nothing
}

// Step decides for one cycle. tap is the present tap (known says whether it was read), lv the
// LV voltage in kV and auto whether the regulator is in Auto.
func (r *Regulator) Step(s Settings, now float64, tap int64, known bool, lv float64, auto bool) Decision {
	if !auto {
		return r.settle(Decision{State: StateManual}, EventManual)
	}
	if !known {
		return r.settle(Decision{State: StateNoPosition}, EventNoPosition)
	}
	deviation := 0.0
	if s.TargetKV > 0 {
		deviation = (lv - s.TargetKV) / s.TargetKV * 100
	}
	if math.Abs(deviation) <= s.BandwidthPct+1e-9 {
		return r.settle(Decision{State: StateInBand, DeviationPct: deviation}, EventInBand)
	}
	direction := Raise
	if deviation > 0 {
		direction = Lower
	}
	if _, allowed := s.Next(tap, direction); !allowed {
		event := EventAtMinTap
		if direction == Raise {
			event = EventAtMaxTap
		}
		r.timing = false
		return r.settle(Decision{State: StateAtLimit, DeviationPct: deviation}, event)
	}
	decision := Decision{State: StateTiming, DeviationPct: deviation}
	if !r.timing || r.direction != direction {
		r.timing, r.direction, r.since = true, direction, now
		decision.Event = EventOutOfBand
	}
	decision.RemainingS = math.Max(0, s.DelayS-(now-r.since))
	if now-r.since >= s.DelayS-1e-9 {
		decision.Action, decision.Event = direction, EventStep
		r.since = now // the next step waits the whole delay again
		decision.RemainingS = s.DelayS
	}
	r.last = decision.State
	return decision
}

// settle ends any running delay and reports event only when the state changes.
func (r *Regulator) settle(decision Decision, event string) Decision {
	r.timing = false
	if r.last != decision.State {
		if r.last != "" || event != EventInBand && event != EventManual {
			decision.Event = event
		}
		r.last = decision.State
	}
	return decision
}

// Direction is the step the running delay leads to ("" when none).
func (r *Regulator) Direction() string {
	if !r.timing {
		return ""
	}
	return r.direction
}
