package goosedom

import (
	"strings"
	"testing"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

const labGUID = "{0123ABCD-0123-4567-89AB-0123456789AB}"

// labNetwork is a stored lab network; local adds the Go-only goose_local key.
func labNetwork(local bool) *pyjson.Object {
	network := pyv.O("mode", "lab", "adapter_guid", labGUID, "goose_reach", "inside", "goose_uplink_guid", "",
		"goose_share_guid", "")
	if local {
		network.Set("goose_local", true)
	}
	return network
}

func labUplink() *pyjson.Object {
	return pyv.O("guid", labGUID, "label", "VMware VMnet8", "kind", "virtual", "why", "lab")
}

// ADR-0119: GOOSE goes out with the IEDs' network; this-PC-only IEDs keep it inside.
func TestGooseGoesOutWithTheIEDsNetwork(t *testing.T) {
	if !Shares(labNetwork(false), labUplink()) {
		t.Fatal("lab IEDs: GOOSE should go to the lab adapter")
	}
	if Shares(labNetwork(true), labUplink()) {
		t.Fatal("local keeps GOOSE inside")
	}
	loopback := pyv.O("mode", "loopback", "goose_reach", "inside")
	wired := pyv.O("guid", labGUID, "label", "Ethernet", "kind", "ethernet", "why", "wired")
	if Shares(loopback, wired) {
		t.Fatal("this-PC-only IEDs must not send GOOSE to an automatically found cable")
	}
	if !Shares(loopback, pyv.O("guid", labGUID, "label", "Ethernet", "why", "chosen")) {
		t.Fatal("an adapter chosen under Advanced receives the GOOSE")
	}
	if !Shares(pyv.O("mode", "loopback", "goose_reach", "outside"), wired) {
		t.Fatal("a project shared before keeps sending on its adapter")
	}
	if Reach(labNetwork(true)) != "local" || Reach(labNetwork(false)) != "auto" {
		t.Fatal("reach is read from goose_local")
	}
	state, message := Status(loopback, wired, "", "", pyv.O("uplink", "open", "share", "off"))
	if state != "listening" || !strings.Contains(message, "this PC only") {
		t.Fatalf("loopback status %s %q", state, message)
	}
}

func TestNpcapMissingIsOneClearLine(t *testing.T) {
	state, message := Hold(labNetwork(false), labUplink(), false, nil)
	if state != "share_held" || message != NpcapMissing {
		t.Fatalf("hold %s %q", state, message)
	}
	if state, _ := Hold(labNetwork(true), labUplink(), false, nil); state != "" {
		t.Fatal("local GOOSE never needs Npcap")
	}
}

func TestRealDevicesAskOncePerAdapter(t *testing.T) {
	real := []*pyjson.Object{pyv.O("id", "R1", "role", "real")}
	network := labNetwork(false)
	if state, _ := Hold(network, labUplink(), true, real); state != "confirm" {
		t.Fatalf("real IEDs without confirmation: %s", state)
	}
	if pyv.GetStr(Risk(network, labUplink(), real, false), "level") != "high" {
		t.Fatal("real IEDs make the warning high")
	}
	network.Set("goose_share_guid", labGUID)
	if state, _ := Hold(network, labUplink(), true, real); state != "" || !Confirmed(network, labUplink()) {
		t.Fatal("the confirmation for this adapter should let GOOSE go")
	}
	if pyv.GetStr(Risk(network, labUplink(), nil, false), "level") != "warning" {
		t.Fatal("the IED network alone is not a high risk any more")
	}
	state, message := Status(network, labUplink(), "", "", pyv.O("uplink", "open", "share", "real_hold"))
	if state != "confirm" || !strings.Contains(message, "VMware VMnet8") {
		t.Fatalf("engine real hold %s %q", state, message)
	}
}

func TestTwinKeepsADR0073(t *testing.T) {
	t.Cleanup(AsPythonTwin())
	outside := labNetwork(false).Set("goose_reach", "outside")
	if Shares(labNetwork(false), labUplink()) || !Shares(outside, labUplink()) {
		t.Fatal("twin shares only when outside")
	}
	if !Confirmed(outside, labUplink()) || len(EngineCapabilities()) != 1 {
		t.Fatal("twin confirmation and capabilities")
	}
}
