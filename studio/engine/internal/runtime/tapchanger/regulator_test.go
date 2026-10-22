package tapchanger

import (
	"math"
	"testing"

	"studio-engine/internal/pyjson"
)

func settings() Settings {
	return Settings{MinTap: 1, MaxTap: 17, NominalTap: 9, Enabled: true, TargetKV: 11, BandwidthPct: 1.5,
		DelayS: 30, StepPct: 1.25, HVRatedKV: 132, LVRatedKV: 11, SourceKV: 132}
}

func TestLVVoltageFollowsTheTap(t *testing.T) {
	s := settings()
	cases := map[int64]float64{9: 11, 10: 11.1375, 8: 10.8625, 17: 12.1, 1: 9.9}
	for tap, want := range cases {
		if got := s.LVVoltage(tap); math.Abs(got-want) > 1e-9 {
			t.Errorf("tap %d: got %v kV, want %v kV", tap, got, want)
		}
	}
	s.SourceKV = 140 // a high source voltage raises the LV voltage with the same ratio
	if got := s.LVVoltage(9); math.Abs(got-11.6667) > 1e-9 {
		t.Errorf("source 140 kV at nominal tap: got %v", got)
	}
	if (Settings{}).Ratio(5) != 0 {
		t.Error("a zero HV rating must give ratio 0, not a division by zero")
	}
}

func TestNextRespectsLimits(t *testing.T) {
	s := settings()
	if next, ok := s.Next(17, Raise); ok || next != 18 {
		t.Errorf("raise at max: %d %v", next, ok)
	}
	if _, ok := s.Next(1, Lower); ok {
		t.Error("lower at min must not be allowed")
	}
	if next, ok := s.Next(9, Lower); !ok || next != 8 {
		t.Errorf("lower from 9: %d %v", next, ok)
	}
	if s.Clamp(40) != 17 || s.Clamp(-3) != 1 {
		t.Error("clamp")
	}
}

func TestInBandDoesNothing(t *testing.T) {
	var r Regulator
	s := settings()
	for now := 0.0; now < 100; now += 10 {
		d := r.Step(s, now, 9, true, 11.1, true) // +0.9 %, inside 1.5 %
		if d.Action != "" || d.State != StateInBand || d.Event != "" {
			t.Fatalf("at %v: %+v", now, d)
		}
	}
}

func TestAboveBandLowersAfterDelay(t *testing.T) {
	var r Regulator
	s := settings()
	d := r.Step(s, 0, 9, true, 11.5, true) // +4.5 %
	if d.Event != EventOutOfBand || d.Action != "" || d.State != StateTiming || d.RemainingS != 30 {
		t.Fatalf("start: %+v", d)
	}
	if d = r.Step(s, 29.9, 9, true, 11.5, true); d.Action != "" || d.Event != "" {
		t.Fatalf("before the delay: %+v", d)
	}
	if d = r.Step(s, 30, 9, true, 11.5, true); d.Action != Lower || d.Event != EventStep {
		t.Fatalf("after the delay: %+v", d)
	}
	if d = r.Step(s, 31, 8, true, 11.4, true); d.Action != "" {
		t.Fatalf("the next step must wait the whole delay again: %+v", d)
	}
	if d = r.Step(s, 60, 8, true, 11.4, true); d.Action != Lower {
		t.Fatalf("second step: %+v", d)
	}
}

func TestBelowBandRaisesAndBackInBandCancels(t *testing.T) {
	var r Regulator
	s := settings()
	r.Step(s, 0, 9, true, 10.5, true)
	if d := r.Step(s, 10, 9, true, 11, true); d.Event != EventInBand || d.Action != "" {
		t.Fatalf("back in band: %+v", d)
	}
	if d := r.Step(s, 35, 9, true, 10.5, true); d.Event != EventOutOfBand || d.Action != "" {
		t.Fatalf("the delay must restart after coming back in band: %+v", d)
	}
	if d := r.Step(s, 65, 9, true, 10.5, true); d.Action != Raise {
		t.Fatalf("raise after the delay: %+v", d)
	}
}

func TestLimitsStopTheRegulator(t *testing.T) {
	var r Regulator
	s := settings()
	d := r.Step(s, 0, 17, true, 10, true)
	if d.State != StateAtLimit || d.Event != EventAtMaxTap || d.Action != "" {
		t.Fatalf("at max: %+v", d)
	}
	if d = r.Step(s, 100, 17, true, 10, true); d.Event != "" || d.Action != "" {
		t.Fatalf("a limit is logged once and never commands: %+v", d)
	}
	var low Regulator
	if d = low.Step(s, 0, 1, true, 12, true); d.Event != EventAtMinTap || d.Action != "" {
		t.Fatalf("at min: %+v", d)
	}
}

func TestManualAndUnknownTapDoNothing(t *testing.T) {
	var r Regulator
	s := settings()
	r.Step(s, 0, 9, true, 12, true)
	if d := r.Step(s, 10, 9, true, 12, false); d.State != StateManual || d.Event != EventManual {
		t.Fatalf("manual: %+v", d)
	}
	if d := r.Step(s, 100, 9, true, 12, false); d.Action != "" || d.Event != "" {
		t.Fatalf("manual never commands: %+v", d)
	}
	if d := r.Step(s, 110, 0, false, 12, true); d.State != StateNoPosition || d.Event != EventNoPosition || d.Action != "" {
		t.Fatalf("unknown tap: %+v", d)
	}
	var fresh Regulator
	if d := fresh.Step(s, 0, 9, true, 11, false); d.Event != "" {
		t.Fatalf("a start in Manual logs nothing: %+v", d)
	}
}

func TestZeroDelayStepsAtOnce(t *testing.T) {
	var r Regulator
	s := settings()
	s.DelayS = 0
	if d := r.Step(s, 0, 9, true, 12, true); d.Action != Lower {
		t.Fatalf("zero delay: %+v", d)
	}
}

func TestReadDefaultsAndValues(t *testing.T) {
	if got := Read(nil); got.MaxTap != 17 || got.TargetKV != 11 || got.Enabled {
		t.Fatalf("defaults: %+v", got)
	}
	avr := pyjson.NewObject().Set("enabled", true).Set("target_kv", 33.0).Set("delay_s", int64(5))
	tap := pyjson.NewObject().Set("min_tap", int64(-8)).Set("max_tap", int64(8)).Set("nominal_tap", int64(0)).Set("avr", avr)
	got := Read(tap)
	if !got.Enabled || got.TargetKV != 33 || got.DelayS != 5 || got.MinTap != -8 || got.NominalTap != 0 {
		t.Fatalf("values: %+v", got)
	}
}
