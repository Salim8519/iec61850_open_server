package netset

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// DefaultLive returns a fresh DEFAULT_LIVE (the real-IED connection settings).
func DefaultLive() *pyjson.Object {
	return pyjson.NewObject().Set("ip", "").Set("port", int64(102)).Set("access", "auto_reports").
		Set("poll_ms", int64(1000))
}

// ValidateLive is validate_live: how ScadaBench reaches a real IED.
func ValidateLive(raw any) (*pyjson.Object, error) {
	value, err := pyval.Merge(DefaultLive(), pyval.Or(raw, pyjson.NewObject()))
	if err != nil {
		return nil, err
	}
	access, _ := value.Get("access")
	if member, err := pyval.InStrings(access, "auto_reports", "polling"); err != nil {
		return nil, err
	} else if !member {
		return nil, apperr.ValueError("Access must be 'auto_reports' (use a free report block) or 'polling'")
	}
	poll, _ := value.Get("poll_ms")
	if !pyval.IsInt(poll) || !pyval.Between(poll, 250, 60000) {
		return nil, apperr.ValueError("Polling interval must be between 250 ms and 60 s")
	}
	// A real device may also be another simulator on this PC, so loopback is allowed here.
	ip, err := ipv4(pyval.Lookup(value, "ip", nil), "Real IED address", true, true)
	if err != nil {
		return nil, err
	}
	number, err := port(pyval.Lookup(value, "port", nil), "Real IED port")
	if err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("ip", ip).Set("port", number).Set("access", access).Set("poll_ms", poll), nil
}

// ValidateIEDSettings is validate_ied_settings: the stored settings of one IED with the
// format-11 defaults, in the order role, live, lab_ip, goose_publish, test_mode.
func ValidateIEDSettings(value *pyjson.Object) (*pyjson.Object, error) {
	role := pyval.Lookup(value, "role", "simulated")
	if member, err := pyval.InStrings(role, "simulated", "real", "off"); err != nil {
		return nil, err
	} else if !member {
		return nil, apperr.ValueError("IED role must be simulated, real or off")
	}
	publish := pyval.Lookup(value, "goose_publish", true)
	if _, ok := publish.(bool); !ok {
		return nil, apperr.ValueError("GOOSE publishing must be on or off")
	}
	live, err := ValidateLive(pyval.Lookup(value, "live", nil))
	if err != nil {
		return nil, err
	}
	lab, err := ipv4(pyval.Lookup(value, "lab_ip", ""), "Lab network address", true, false)
	if err != nil {
		return nil, err
	}
	testMode, err := ValidateTestMode(pyval.Lookup(value, "test_mode", nil))
	if err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("role", role).Set("live", live).Set("lab_ip", lab).
		Set("goose_publish", publish).Set("test_mode", testMode), nil
}

// EffectiveEndpoint is effective_endpoint: where a simulated IED listens in this mode.
func EffectiveEndpoint(ied, network *pyjson.Object) (any, any) {
	portValue := pyval.Lookup(ied, "port", int64(102))
	mode, _ := network.Get("mode")
	if lab := pyval.Lookup(ied, "lab_ip", nil); mode == "lab" && pyval.Truthy(lab) {
		return lab, portValue
	}
	return pyval.Lookup(ied, "ip", nil), portValue
}

// RealEndpoint is real_endpoint: the user override, else the SCD address, and its origin.
func RealEndpoint(ied *pyjson.Object) (any, any, string) {
	live, _ := pyval.Or(pyval.Lookup(ied, "live", nil), DefaultLive()).(*pyjson.Object)
	address, _ := pyval.Or(pyval.Lookup(ied, "scl_address", nil), pyjson.NewObject()).(*pyjson.Object)
	scd := pyval.Lookup(address, "ip", "")
	ip, _ := live.Get("ip")
	portValue, _ := live.Get("port")
	origin := "missing"
	if pyval.Truthy(ip) {
		origin = "override"
	} else if pyval.Truthy(scd) {
		origin = "scd"
	}
	return pyval.Or(ip, scd), portValue, origin
}
