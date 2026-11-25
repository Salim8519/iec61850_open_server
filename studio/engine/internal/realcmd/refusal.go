// Package realcmd sends one confirmed command to a real IED from Devices (ADR-0138): only while
// that IED's own Allow commands is on and its live session is connected, through the shared
// studio_control client with the device's own control model, and it observes the feedback.
package realcmd

import (
	"studio-engine/internal/project/netset"
	"studio-engine/internal/pyjson"
)

// Request is what Devices asks: the IED, the control object and on/close (true) or off/open.
type Request struct {
	IED       *pyjson.Object
	Ref       string
	Value     any
	Confirmed bool
	State     string // the live session state, "" without a session
}

// Refusal is why this command is not sent, in words, or "" when it may be sent. Pure.
func Refusal(request Request) string {
	name := text(request.IED, "name")
	switch {
	case !request.Confirmed:
		return "Confirm the command first"
	case text(request.IED, "role") != "real":
		return name + " is not set to Monitor real device, so nothing was sent from here"
	case !netset.AllowsCommands(request.IED):
		return name + " has Allow commands off; turn it on in Devices first"
	}
	if _, isBool := request.Value.(bool); !isBool {
		return "A command here is on/off or close/open"
	}
	if controlOf(request.IED, request.Ref) == nil {
		return name + " has no on/off or open/close command " + request.Ref + " in its CID"
	}
	if request.State != "live" {
		return name + " is not connected; press Run and wait until it shows Live"
	}
	return ""
}

// StepRequest is one step Devices asks for (ADR-0149); Known says the CID has the command.
type StepRequest struct {
	IED   *pyjson.Object
	Step  Step
	Known bool
	State string // the live session state, "" without a session
}

// StepRefusal is why this step is not sent, in words, or "" when it may be sent. Pure. Cancel
// needs no confirmation: it only gives a selection back.
func StepRefusal(request StepRequest) string {
	name := text(request.IED, "name")
	step := request.Step
	switch {
	case stepOps[step.Action] == "":
		return "A command step is select, operate or cancel"
	case !step.Confirmed && step.Action != "cancel":
		return "Confirm the command first"
	case text(request.IED, "role") != "real":
		return name + " is not set to Monitor real device, so nothing was sent from here"
	case !netset.AllowsCommands(request.IED):
		return name + " has Allow commands off; turn it on in Devices first"
	case !request.Known:
		return name + " has no command " + step.Ref + " in its CID"
	case step.Action == "operate" && step.Value == nil:
		return "Choose the value to send first"
	case request.State != "live":
		return name + " is not connected; press Run and wait until it shows Live"
	}
	return ""
}

// controlOf is the SPC/DPC control with this reference, or nil.
func controlOf(ied *pyjson.Object, ref string) *pyjson.Object {
	list, _ := get(ied, "controls").([]any)
	for _, raw := range list {
		if control, ok := raw.(*pyjson.Object); ok && text(control, "ref") == ref {
			if cdc := text(control, "cdc"); cdc == "SPC" || cdc == "DPC" {
				return control
			}
		}
	}
	return nil
}

// expected is the feedback a successful command leaves: DPC closed 2 / open 1, SPC the value.
func expected(control *pyjson.Object, value bool) any {
	if text(control, "cdc") == "DPC" {
		return map[bool]int64{true: 2, false: 1}[value]
	}
	return value
}

// positionText says a feedback value in words.
func positionText(value any) string {
	switch v := value.(type) {
	case bool:
		return map[bool]string{true: "on", false: "off"}[v]
	case int64:
		if word, found := map[int64]string{0: "intermediate", 1: "open", 2: "closed", 3: "bad state"}[v]; found {
			return word
		}
	}
	return "unknown"
}

func get(o *pyjson.Object, key string) any {
	if o == nil {
		return nil
	}
	value, _ := o.Get(key)
	return value
}

func text(o *pyjson.Object, key string) string {
	value, _ := get(o, key).(string)
	return value
}
