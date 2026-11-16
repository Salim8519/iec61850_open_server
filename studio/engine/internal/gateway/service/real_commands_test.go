package service

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// realBay is the bay as a real device; allow is its Allow commands switch, nil when unset.
func realBay(allow any) *pyjson.Object {
	live := pyjson.NewObject().Set("ip", "192.168.10.5").Set("port", int64(102))
	if allow != nil {
		live.Set("allow_commands", allow)
	}
	return copyObject(bay()).Set("role", "real").Set("live", live)
}

// A SCADA command reaches a real IED only when the gateway switch and the IED's own Allow
// commands switch are both on; a missing switch is off (ADR-0138).
func TestRealIEDCommandsNeedBothSwitches(t *testing.T) {
	command := point("command_double", 1, "BayCTRL/XCBR1.Pos")
	cases := []struct {
		name    string
		gateway bool
		allow   any
		want    string
	}{
		{"both on", true, true, ""},
		{"never set", true, nil, "Bay is a real device with commands off; turn on Allow commands in Devices"},
		{"IED off", true, false, "Bay is a real device with commands off; turn on Allow commands in Devices"},
		{"gateway off", false, true, "commands to real IEDs are switched off in the gateway settings"},
		{"both off", false, false, "commands to real IEDs are switched off in the gateway settings"},
	}
	for _, item := range cases {
		if got := Refusal(command, realBay(item.allow), item.gateway); got != item.want {
			t.Errorf("%s: refusal %q, want %q", item.name, got, item.want)
		}
	}
}

// The native worker gets monitor_only for a real IED's points until both switches are on, so
// it answers the SCADA negatively itself (104 negative ACT_CON, DNP3 NOT_AUTHORIZED).
func TestRealIEDPointsAreMonitorOnlyUntilAllowed(t *testing.T) {
	manager := &Manager{}
	command := point("command_double", 1, "BayCTRL/XCBR1.Pos")
	on := pyjson.NewObject().Set("real_commands", true)
	off := pyjson.NewObject().Set("real_commands", false)
	if !manager.MonitorOnly(command, on, realBay(nil)) || !manager.MonitorOnly(command, on, realBay(false)) {
		t.Fatal("a real IED without Allow commands is monitor-only")
	}
	if !manager.MonitorOnly(command, off, realBay(true)) {
		t.Fatal("the gateway switch still makes a real IED monitor-only")
	}
	if manager.MonitorOnly(command, on, realBay(true)) {
		t.Fatal("with both switches on, the real IED takes commands")
	}
	if manager.MonitorOnly(command, on, bay()) {
		t.Fatal("a simulated IED never needs the switch")
	}
	if reason := Refusal(command, bay(), true); strings.Contains(reason, "Allow commands") {
		t.Fatalf("a simulated IED is not refused for the switch: %q", reason)
	}
}
