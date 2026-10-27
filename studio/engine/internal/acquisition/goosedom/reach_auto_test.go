package goosedom

import (
	"strings"
	"testing"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

const labGUID = "{0123ABCD-0123-4567-89AB-0123456789AB}"

// labNetwork is a stored lab network with the given goose_reach.
func labNetwork(reach string) *pyjson.Object {
	return pyv.O("mode", "lab", "adapter_guid", labGUID, "goose_reach", reach, "goose_uplink_guid", "",
		"goose_share_guid", "")
}

func labUplink() *pyjson.Object {
	return pyv.O("guid", labGUID, "label", "VMware VMnet8", "kind", "virtual", "why", "lab")
}

// ADR-0122: GOOSE stays inside until the user confirmed sharing; no network sends it by itself.
func TestGooseStaysInsideUntilShared(t *testing.T) {
	if Shares(labNetwork("inside"), labUplink()) {
		t.Fatal("lab IEDs must not send GOOSE without the user's confirmation")
	}
	chosen := pyv.O("guid", labGUID, "label", "Ethernet", "kind", "ethernet", "why", "chosen")
	if Shares(pyv.O("mode", "loopback", "goose_reach", "inside"), chosen) {
		t.Fatal("an adapter chosen under Advanced must not send GOOSE by itself")
	}
	if !Shares(labNetwork("outside"), labUplink()) || !Shares(pyv.O("goose_reach", "outside"), chosen) {
		t.Fatal("a confirmed share sends on its adapter")
	}
	if Shares(labNetwork("outside"), pyv.O("guid", "")) {
		t.Fatal("no adapter, nothing sent")
	}
	if Reach(pyv.O()) != "inside" || Reach(labNetwork("outside")) != "outside" {
		t.Fatal("reach is the stored goose_reach")
	}
	state, message := Status(labNetwork("inside"), labUplink(), "", "", pyv.O("uplink", "open", "share", "off"))
	if state != "listening" || !strings.Contains(message, "Nothing is sent") {
		t.Fatalf("inside status %s %q", state, message)
	}
}

func TestNpcapMissingIsOneClearLine(t *testing.T) {
	state, message := Hold(labNetwork("outside"), labUplink(), false, nil)
	if state != "share_held" || message != NpcapMissing {
		t.Fatalf("hold %s %q", state, message)
	}
	if state, _ := Hold(labNetwork("inside"), labUplink(), false, nil); state != "" {
		t.Fatal("GOOSE inside never needs Npcap")
	}
}

func TestSharingNextToRealDevicesIsConfirmedPerAdapter(t *testing.T) {
	real := []*pyjson.Object{pyv.O("id", "R1", "role", "real")}
	network := labNetwork("outside")
	if state, _ := Hold(network, labUplink(), true, real); state != "confirm" {
		t.Fatalf("real IEDs without confirmation on this adapter: %s", state)
	}
	if pyv.GetStr(Risk(network, labUplink(), real, false), "level") != "high" {
		t.Fatal("real IEDs make the warning high")
	}
	network.Set("goose_share_guid", labGUID)
	if state, _ := Hold(network, labUplink(), true, real); state != "" || !Confirmed(network, labUplink()) {
		t.Fatal("the confirmation for this adapter should let GOOSE go")
	}
	state, message := Status(network, labUplink(), "", "", pyv.O("uplink", "open", "share", "real_hold"))
	if state != "confirm" || !strings.Contains(message, "VMware VMnet8") {
		t.Fatalf("engine real hold %s %q", state, message)
	}
}

func TestTwinKeepsADR0073(t *testing.T) {
	t.Cleanup(AsPythonTwin())
	if Shares(labNetwork("inside"), labUplink()) || !Shares(labNetwork("outside"), labUplink()) {
		t.Fatal("twin shares only when outside")
	}
	if !Confirmed(labNetwork("outside"), labUplink()) || len(EngineCapabilities()) != 1 {
		t.Fatal("twin confirmation and capabilities")
	}
}
