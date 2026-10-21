package modes

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// Write is one (reference, value) pair mode_writes returns.
type Write struct {
	Ref   string
	Value any
}

// ModeWrites is mode_writes: the writes that put a simulated IED in the saved test mode.
// settings is the validated test_mode object; current maps refs to present values (a
// missing node Mod counts as 1, as do values that are not a mode).
func ModeWrites(ied, settings, current *pyjson.Object) []Write {
	devices, simulation := ModePoints(ied)
	index := deviceIndex(devices)
	writes := []Write{}
	wanted := pyv.GetObj(settings, "devices")
	if wanted != nil {
		for _, domain := range wanted.Keys() {
			mode, _ := wanted.Get(domain)
			at, found := index[domain]
			if !found {
				continue
			}
			device := devices[at]
			if device.Mod != "" {
				writes = append(writes, Write{Ref: device.Mod, Value: mode})
			}
			for _, beh := range device.Beh {
				lnMod := pyv.GetOr(current, beh.OwnMod, int64(1))
				if device.Mod != "" && beh.OwnMod == device.Mod {
					lnMod = mode
				}
				if _, known := modeKey(lnMod); !known {
					lnMod = int64(1)
				}
				writes = append(writes, Write{Ref: beh.Ref, Value: DeriveBeh(mode, lnMod)})
			}
		}
	}
	for _, ref := range simulation {
		writes = append(writes, Write{Ref: ref, Value: pyv.Get(settings, "simulation")})
	}
	return writes
}

// ModeSummary is mode_summary: the mode of each logical device from read values (Beh,
// else Mod), the worst mode for badges and whether any LPHD.Sim reads true (nil when no
// Sim value is a boolean).
func ModeSummary(ied, values *pyjson.Object) *pyjson.Object {
	devices, simulation := ModePoints(ied)
	names := pyjson.NewObject()
	var worst any
	for _, device := range devices {
		value := pyv.GetOr(values, device.Domain+"/LLN0.Beh.stVal", pyv.Get(values, device.Domain+"/LLN0.Mod.stVal"))
		name, ok := ModeName(value)
		if !ok {
			names.Set(device.Domain, "not available")
			continue
		}
		names.Set(device.Domain, name)
		if worst == nil || modeRank(name) > modeRank(worst.(string)) {
			worst = name
		}
	}
	var sims []bool
	for _, ref := range simulation {
		if flag, isBool := pyv.Get(values, ref).(bool); isBool {
			sims = append(sims, flag)
		}
	}
	var anySim any
	if len(sims) > 0 {
		anySim = false
		for _, flag := range sims {
			if flag {
				anySim = true
			}
		}
	}
	return pyv.O("devices", names, "simulation", anySim, "worst", worst)
}
