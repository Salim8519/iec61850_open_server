package goosemsg

import "studio-engine/internal/pyjson"

// NotRunning is the words while nothing runs.
const NotRunning = "Run the IEDs to send GOOSE. Press Run, then try again."

// Refusal says in plain words why ScadaBench must not send GOOSE for this IED now, or "": a real
// device is watched only, a skipped IED and an IED whose GOOSE is switched off in Devices send
// nothing, and an IED must be part of the run (inRun).
func Refusal(ied *pyjson.Object, inRun bool) string {
	name := text(ied, "name")
	role, set := field(ied, "role").(string)
	if !set {
		role = "simulated"
	}
	switch role {
	case "real":
		return name + " is a real device. ScadaBench only watches its GOOSE and never sends for it."
	case "off":
		return name + " is skipped in this project's run. Set it to 'Simulate' to send its GOOSE."
	}
	if publish, present := ied.Get("goose_publish"); present && publish == false {
		return name + " does not send GOOSE: 'Send its GOOSE messages' is off in Devices. Switch it on, then Run again."
	}
	if !inRun {
		return name + " is not running. Run this IED first."
	}
	return ""
}
