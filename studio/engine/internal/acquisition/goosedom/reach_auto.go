package goosedom

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// pythonTwin switches ADR-0119 off, so golden tests still compare every other byte with
// the frozen Python engine. Only tests set it; the application always runs with it false.
var pythonTwin bool

// AsPythonTwin turns ADR-0119 off for a golden test and returns the function that turns
// it back on. The application never calls it.
func AsPythonTwin() (restore func()) {
	pythonTwin = true
	return func() { pythonTwin = false }
}

// PythonTwin reports whether a golden test switched ADR-0119 off.
func PythonTwin() bool { return pythonTwin }

// EngineCapabilities are what studio_goose must offer; v2 holds sharing next to an
// unconfirmed real device (ADR-0119).
func EngineCapabilities() []string {
	if pythonTwin {
		return []string{"goose_engine_v1"}
	}
	return []string{"goose_engine_v1", "goose_engine_v2"}
}

const (
	// NpcapMissing is the one line shown when GOOSE should leave the PC but cannot.
	NpcapMissing = "Npcap isn't installed, so GOOSE can't leave this PC."
	realIEDs     = "This project has real IEDs. Confirm before ScadaBench sends GOOSE next to them on "
	realHeard    = "A real device is sending GOOSE on "
	riskRealIEDs = "This project has real IEDs; a real IED may act on the GOOSE or already send the same GOOSE."
	riskHeard2   = "GOOSE from real devices is being received on this adapter."
)

// Reach is where the user wants GOOSE (ADR-0122, as ADR-0073): "inside" by default, "outside"
// only after the user confirmed sharing. The stored goose_reach is read as it is, so the
// Python engine and this one agree on every project.
func Reach(network *pyjson.Object) string {
	return pyv.PyStr(pyv.GetOr(network, "goose_reach", "inside"))
}

// Shares reports whether GOOSE goes out: only when the user confirmed sharing (stored
// "outside") and an adapter is there. Neither the IED network nor an adapter chosen under
// Advanced sends GOOSE by itself (ADR-0122).
func Shares(network, uplink *pyjson.Object) bool {
	if pythonTwin {
		return pyv.Equal(pyv.Get(network, "goose_reach"), "outside")
	}
	return Reach(network) == "outside" && pyv.Truthy(pyv.Get(uplink, "guid"))
}

// Confirmed reports whether the user confirmed sharing next to real devices on this adapter.
// The Python twin had one confirmation for all sharing, so it is always true there.
func Confirmed(network, uplink *pyjson.Object) bool {
	if pythonTwin {
		return true
	}
	guid := pyv.Get(uplink, "guid")
	return pyv.Truthy(guid) && pyv.Equal(pyv.Get(network, "goose_share_guid"), guid)
}

// Hold returns (state, message) when GOOSE that should go out cannot right now:
// "share_held" (Npcap, adapter) or "confirm" (real IEDs not confirmed); ("", "") when it can.
func Hold(network, uplink *pyjson.Object, npcap bool, ieds []*pyjson.Object) (string, string) {
	if pythonTwin {
		if message := ShareHold(network, uplink, npcap); message != "" {
			return "share_held", message
		}
		return "", ""
	}
	if !Shares(network, uplink) {
		return "", ""
	}
	if !npcap {
		return "share_held", NpcapMissing
	}
	if hasRealIED(ieds) && !Confirmed(network, uplink) {
		return "confirm", realIEDs + uplinkLabel(uplink) + "."
	}
	return "", ""
}

// Risk is ShareRisk for the confirm warning: only real devices make it high (ADR-0119).
func Risk(network, uplink *pyjson.Object, ieds []*pyjson.Object, heard bool) *pyjson.Object {
	if pythonTwin {
		return ShareRisk(network, uplink, ieds, heard)
	}
	reasons := []any{}
	if hasRealIED(ieds) {
		reasons = append(reasons, riskRealIEDs)
	}
	if heard {
		reasons = append(reasons, riskHeard2)
	}
	level := "warning"
	if len(reasons) > 0 {
		level = "high"
	}
	if pyv.Equal(pyv.Get(uplink, "kind"), "wifi") {
		reasons = append(reasons, riskWifi)
	}
	return pyv.O("level", level, "reasons", reasons)
}

// Status is the single (state, message) shown for GOOSE while running.
func Status(network, uplink *pyjson.Object, holdState, hold string, link *pyjson.Object) (string, string) {
	if pythonTwin {
		return ReachStatus(network, uplink, hold, link)
	}
	label := uplinkLabel(uplink)
	switch {
	case holdState != "":
		return holdState, hold
	case pyv.Equal(pyv.GetOr(link, "share", "off"), "real_hold"):
		return "confirm", realHeard + label + ". Confirm before ScadaBench sends GOOSE next to it."
	case pyv.Equal(pyv.GetOr(link, "share", "off"), "sharing"):
		return "sharing", "GOOSE is shared on " + label + ". Protection is on."
	case pyv.Equal(pyv.GetOr(link, "share", "off"), "checking"):
		return "checking", "Checking " + label + " for real devices before sending GOOSE…"
	case pyv.Equal(pyv.Get(link, "uplink"), "open"):
		return "listening", "GOOSE stays inside ScadaBench. Real GOOSE on " + label + " is heard. Nothing is sent."
	}
	return "inside", "GOOSE stays inside ScadaBench. Nothing is sent to the network."
}

func hasRealIED(ieds []*pyjson.Object) bool {
	for _, ied := range ieds {
		if pyv.Equal(pyv.Get(ied, "role"), "real") {
			return true
		}
	}
	return false
}

func uplinkLabel(uplink *pyjson.Object) string {
	if value := pyv.Get(uplink, "label"); pyv.Truthy(value) {
		return pyv.PyStr(value)
	}
	return "the network adapter"
}
