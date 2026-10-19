package modes

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// Behaviour is one logical node of a logical device: its Beh.stVal reference and the
// Mod.stVal reference of the same node.
type Behaviour struct {
	Ref    string
	OwnMod string
}

// Device is one logical device as mode_points sees it: its LLN0.Mod.stVal reference
// ("" when the SCD model has none, Python None) and the Beh points of its nodes in
// sorted reference order.
type Device struct {
	Domain string
	Mod    string
	Beh    []Behaviour
}

// ModeSignals is mode_signals: reference and FC of the only points ModePoints and
// ModeItems read (Mod, Beh, Sim), so health can keep a small list.
func ModeSignals(signals []any) []any {
	result := []any{}
	for _, signal := range pyv.Objects(signals) {
		if isModeSignal(pyv.GetStr(signal, "ref")) {
			result = append(result, pyv.O("ref", pyv.Get(signal, "ref"), "fc", pyv.Get(signal, "fc")))
		}
	}
	return result
}

// DeriveBeh is derive_beh: the behaviour of a logical node from its logical device's Mod
// and its own Mod (values compare with Python ==, so 3.0 counts as 3).
func DeriveBeh(ldMod, lnMod any) int64 {
	modes := []any{ldMod, lnMod}
	if containsMode(modes, 5) {
		return 5
	}
	test := containsMode(modes, 3) || containsMode(modes, 4)
	blocked := containsMode(modes, 2) || containsMode(modes, 4)
	switch {
	case test && blocked:
		return 4
	case test:
		return 3
	case blocked:
		return 2
	}
	return 1
}

func containsMode(modes []any, mode int64) bool {
	for _, value := range modes {
		if pyv.Equal(value, mode) {
			return true
		}
	}
	return false
}

// ModePoints is mode_points: per logical device (in order of first sorted Beh reference)
// its Mod reference and the Beh references of its nodes, plus the sorted LPHD.Sim refs.
func ModePoints(ied *pyjson.Object) ([]Device, []string) {
	refs := map[string]bool{}
	for _, signal := range pyv.Objects(pyv.GetList(ied, "signals")) {
		refs[pyv.GetStr(signal, "ref")] = true
	}
	sorted := sortedKeys(refs)
	devices := []Device{}
	index := map[string]int{}
	for _, ref := range sorted {
		domain, node, ok := behaviourRef(ref)
		if !ok {
			continue
		}
		at, found := index[domain]
		if !found {
			at = len(devices)
			index[domain] = at
			devices = append(devices, Device{Domain: domain})
		}
		devices[at].Beh = append(devices[at].Beh, Behaviour{Ref: ref, OwnMod: domain + "/" + node + ".Mod.stVal"})
	}
	for at := range devices {
		if mod := devices[at].Domain + "/LLN0.Mod.stVal"; refs[mod] {
			devices[at].Mod = mod
		}
	}
	simulation := []string{}
	for _, ref := range sorted {
		if isSimulationRef(ref) {
			simulation = append(simulation, ref)
		}
	}
	return devices, simulation
}

// deviceIndex maps each domain to its position in devices (index once, look up often).
func deviceIndex(devices []Device) map[string]int {
	index := make(map[string]int, len(devices))
	for at, device := range devices {
		index[device.Domain] = at
	}
	return index
}
