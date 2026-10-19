package goosedom

import (
	"sort"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// Reaches lists REACHES: GOOSE stays inside the app or is also shared outside (ADR-0073).
func Reaches() []string { return []string{"inside", "outside"} }

// ReachKeys lists REACH_KEYS, the network settings that decide where GOOSE goes.
func ReachKeys() []string { return []string{"goose_reach", "goose_uplink_guid", "goose_share_guid"} }

// NoAdapter is NO_ADAPTER, the reason shown when no wired adapter can be used.
const NoAdapter = "No connected wired network adapter was found, so GOOSE stays inside ScadaBench. " +
	"Plug in a network cable, or choose an adapter under Advanced."

const chosenMissing = "The adapter chosen under Advanced is not on this PC any more, so GOOSE stays inside."

const wifiWarning = "Wi-Fi usually drops GOOSE from simulated IEDs; use a cable when you can."

// IsNpcapLoopback reports whether the adapter is the Npcap loopback adapter.
func IsNpcapLoopback(adapter *pyjson.Object) bool {
	text := pyLower(strOr(adapter, "name", "") + " " + strOr(adapter, "description", ""))
	return strings.Contains(text, "npcap") && strings.Contains(text, "loopback")
}

func choice(adapter *pyjson.Object, auto bool, why string) *pyjson.Object {
	label := pyv.Get(adapter, "name")
	if !pyv.Truthy(label) {
		label = pyv.Get(adapter, "description")
	}
	if !pyv.Truthy(label) {
		label = pyv.Get(adapter, "guid")
	}
	return pyv.O("guid", pyv.Get(adapter, "guid"), "label", label, "kind", pyv.GetOr(adapter, "kind", ""),
		"connected", pyv.Equal(pyv.Get(adapter, "status"), "Up"), "auto", auto, "why", why, "reason", "")
}

func noUplink(auto bool, why, reason string) *pyjson.Object {
	return pyv.O("guid", "", "label", "", "kind", "", "connected", false, "auto", auto, "why", why,
		"reason", reason)
}

// PickUplink returns the adapter GOOSE listens on (and sends on, when shared). GOOSE-24.
// Order: the adapter chosen under Advanced; the lab adapter in lab mode; then a connected
// wired adapter, preferring the one with the default route. Wi-Fi, VPN, virtual and
// loopback adapters are never picked automatically.
func PickUplink(adapters []*pyjson.Object, network *pyjson.Object) *pyjson.Object {
	byGUID := map[string]*pyjson.Object{}
	for _, adapter := range adapters {
		byGUID[dictKey(pyv.Get(adapter, "guid"))] = adapter
	}
	chosen := pyv.Get(network, "goose_uplink_guid")
	if pyv.Truthy(chosen) {
		if adapter, found := byGUID[dictKey(chosen)]; found {
			return choice(adapter, false, "chosen")
		}
		return noUplink(false, "chosen", chosenMissing)
	}
	lab := pyv.Get(network, "adapter_guid")
	if !pyv.Truthy(lab) {
		lab = ""
	}
	if adapter, found := byGUID[dictKey(lab)]; found && pyv.Equal(pyv.Get(network, "mode"), "lab") {
		return choice(adapter, true, "lab")
	}
	var wired []*pyjson.Object
	for _, adapter := range adapters {
		if pyv.Equal(pyv.Get(adapter, "kind"), "ethernet") && pyv.Equal(pyv.Get(adapter, "status"), "Up") &&
			!IsNpcapLoopback(adapter) {
			wired = append(wired, adapter)
		}
	}
	sort.SliceStable(wired, func(i, j int) bool {
		left, right := !pyv.Truthy(pyv.Get(wired[i], "default_route")), !pyv.Truthy(pyv.Get(wired[j], "default_route"))
		if left != right {
			return !left
		}
		return pyLower(strOr(wired[i], "name", "")) < pyLower(strOr(wired[j], "name", ""))
	})
	if len(wired) > 0 {
		return choice(wired[0], true, "wired")
	}
	return noUplink(true, "", NoAdapter)
}

// AdvancedChoices lists the adapters the user may pick under Advanced, with a warning
// where GOOSE is unreliable. Npcap loopback, Bluetooth and VPN adapters are left out.
func AdvancedChoices(adapters []*pyjson.Object) []any {
	choices := []any{}
	for _, adapter := range adapters {
		kind := pyv.Get(adapter, "kind")
		if IsNpcapLoopback(adapter) || pyv.Equal(kind, "bluetooth") || pyv.Equal(kind, "vpn") {
			continue
		}
		warning := ""
		if pyv.Equal(kind, "wifi") {
			warning = wifiWarning
		}
		choices = append(choices, choice(adapter, false, "chosen").Set("warning", warning))
	}
	return choices
}
