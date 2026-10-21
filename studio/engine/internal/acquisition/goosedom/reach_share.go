package goosedom

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

const (
	riskLab = "Your simulated IEDs use real network addresses on this adapter, so real devices " +
		"may treat their GOOSE as coming from the real IEDs."
	riskHeard = "Real GOOSE is already being received on this adapter, so real devices are connected."
	riskReal  = "This project also has real IEDs; a real IED may already send the same GOOSE."
	riskWifi  = "Wi-Fi usually drops GOOSE from simulated IEDs."
	needNpcap = "GOOSE sharing needs the free Npcap driver. Install it from npcap.com, then press Run again."
)

// ShareRisk says how dangerous sharing is on this adapter: {level, reasons}, level
// "high" when real devices are likely to act on the shared GOOSE, else "warning".
func ShareRisk(network, uplink *pyjson.Object, ieds []*pyjson.Object, heard bool) *pyjson.Object {
	reasons := []any{}
	if pyv.Equal(pyv.Get(network, "mode"), "lab") && pyv.Truthy(pyv.Get(uplink, "guid")) &&
		pyv.Equal(pyv.Get(uplink, "guid"), pyv.Get(network, "adapter_guid")) {
		reasons = append(reasons, riskLab)
	}
	if heard {
		reasons = append(reasons, riskHeard)
	}
	for _, ied := range ieds {
		if pyv.Equal(pyv.Get(ied, "role"), "real") {
			reasons = append(reasons, riskReal)
			break
		}
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

// ShareHold returns why sharing that the user switched on cannot run right now, or ""
// when it can. GOOSE-19.
func ShareHold(network, uplink *pyjson.Object, npcap bool) string {
	if !pyv.Equal(pyv.Get(network, "goose_reach"), "outside") {
		return ""
	}
	if !npcap {
		return needNpcap
	}
	if !pyv.Truthy(pyv.Get(uplink, "guid")) {
		if reason := pyv.Get(uplink, "reason"); pyv.Truthy(reason) {
			return pyv.PyStr(reason)
		}
		return NoAdapter
	}
	if !pyv.Equal(pyv.Get(network, "goose_share_guid"), pyv.Get(uplink, "guid")) {
		return "GOOSE sharing is paused because the adapter is now " + pyv.PyStr(pyv.Get(uplink, "label")) +
			". Switch sharing on again to confirm this adapter."
	}
	return ""
}

// ReachStatus returns the single status line shown for GOOSE while running, as
// (state, message). link is the native link report (nil when there is none).
func ReachStatus(network, uplink *pyjson.Object, hold string, link *pyjson.Object) (string, string) {
	label := "the network adapter"
	if value := pyv.Get(uplink, "label"); pyv.Truthy(value) {
		label = pyv.PyStr(value)
	}
	if hold != "" {
		return "share_held", hold
	}
	share := pyv.GetOr(link, "share", "off")
	if pyv.Equal(share, "sharing") {
		return "sharing", "Sharing GOOSE with the network on " + label + ". The smart guard is on."
	}
	if pyv.Equal(share, "checking") {
		return "checking", "Checking " + label + " for real devices before sharing GOOSE…"
	}
	if pyv.Equal(pyv.Get(link, "uplink"), "open") {
		return "listening", "GOOSE runs inside ScadaBench and listens to real devices on " + label +
			". Nothing is sent to the network."
	}
	return "inside", "GOOSE runs inside ScadaBench. Nothing is sent to the network."
}
