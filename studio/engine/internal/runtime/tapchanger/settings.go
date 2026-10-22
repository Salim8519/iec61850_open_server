// Package tapchanger is the domain of the tap-changer transformer (ADR-0109): the tap range,
// the LV voltage for a tap and the simulated voltage regulator's decisions. It has no I/O.
package tapchanger

import (
	"math"

	"studio-engine/internal/pyjson"
)

// Settings are the validated tap range and regulator settings of one transformer.
type Settings struct {
	MinTap, MaxTap, NominalTap int64
	Enabled                    bool
	TargetKV                   float64 // LV voltage the regulator holds
	BandwidthPct               float64 // no action while |LV - target| <= this % of target
	DelayS                     float64 // out of band this long before one step
	StepPct                    float64 // LV change per tap step, % of the rated ratio
	HVRatedKV, LVRatedKV       float64 // rated ratio at the nominal tap
	SourceKV                   float64 // HV voltage feeding the transformer
}

func number(object *pyjson.Object, key string, fallback float64) float64 {
	if object == nil {
		return fallback
	}
	value, _ := object.Get(key)
	switch typed := value.(type) {
	case float64:
		return typed
	case int64:
		return float64(typed)
	}
	return fallback
}

// Read takes the settings from a validated `tap_changer` object (defaults when absent).
func Read(tap *pyjson.Object) Settings {
	var avr *pyjson.Object
	if tap != nil {
		value, _ := tap.Get("avr")
		avr, _ = value.(*pyjson.Object)
	}
	enabled := false
	if avr != nil {
		value, _ := avr.Get("enabled")
		enabled = value == true
	}
	return Settings{
		MinTap: int64(number(tap, "min_tap", 1)), MaxTap: int64(number(tap, "max_tap", 17)),
		NominalTap: int64(number(tap, "nominal_tap", 9)), Enabled: enabled,
		TargetKV: number(avr, "target_kv", 11), BandwidthPct: number(avr, "bandwidth_pct", 1.5),
		DelayS: number(avr, "delay_s", 30), StepPct: number(avr, "step_pct", 1.25),
		HVRatedKV: number(avr, "hv_rated_kv", 132), LVRatedKV: number(avr, "lv_rated_kv", 11),
		SourceKV: number(avr, "source_kv", 132),
	}
}

// Ratio is the LV/HV voltage ratio at a tap: the rated ratio, one step % higher per tap above
// the nominal tap (Raise means more turns on the LV side, so a higher LV voltage).
func (s Settings) Ratio(tap int64) float64 {
	if s.HVRatedKV <= 0 {
		return 0
	}
	return s.LVRatedKV / s.HVRatedKV * (1 + float64(tap-s.NominalTap)*s.StepPct/100)
}

// LVVoltage is the simulated LV voltage in kV: source x ratio(tap), rounded to 0.1 V so that
// float noise never makes a new value.
func (s Settings) LVVoltage(tap int64) float64 {
	return math.Round(s.SourceKV*s.Ratio(tap)*10000) / 10000
}

// Clamp keeps a tap inside the range.
func (s Settings) Clamp(tap int64) int64 {
	return max(s.MinTap, min(s.MaxTap, tap))
}

// Next is the tap after one raise or lower and whether that move is allowed by the limits.
func (s Settings) Next(tap int64, action string) (int64, bool) {
	switch action {
	case Raise:
		return tap + 1, tap < s.MaxTap
	case Lower:
		return tap - 1, tap > s.MinTap
	}
	return tap, false
}
