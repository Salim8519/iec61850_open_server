package runtime

import (
	"sort"
	"strings"

	"studio-engine/internal/runtime/pyval"
)

// Beh of a simulated IED follows IEC 61850-7-4 Annex A (TMODE-05, TMODE-13, ADR-0115): each
// LN's Beh combines its own Mod with its LD's LLN0.Mod (deriveBeh). Studio derives it at Run,
// after every Mod write it makes (in the same native batch as the Mod) and after an MMS client's
// Mod command. An LD whose LLN0.Mod holds no mode 1-5 is left as imported: nothing is invented.

const behaviourSource = "behaviour"

// isModRef reports whether ref is a Mod status (LLN0 or any LN).
func isModRef(ref string) bool { return strings.HasSuffix(ref, ".Mod.stVal") }

// behaviourWrites returns the Beh writes the LDs of model need. pending holds Mod values about
// to be written (they win over the table); skip holds Beh references already being written.
// With onlyChanged, a Beh already holding its derived value is left alone.
func behaviourWrites(model *iedModel, current func(string) any, pending map[string]any, skip map[string]bool,
	onlyChanged bool) []item {
	mode := func(ref string) (int64, bool) {
		value, found := pending[ref]
		if !found {
			value = current(ref)
		}
		number, ok := value.(int64)
		return number, ok && isMode(number)
	}
	devices, _ := modePoints(model)
	var writes []item
	for _, domain := range sortedDeviceKeys(devices) {
		device := devices[domain]
		ldMod, valid := int64(1), true
		if device.mod != "" {
			ldMod, valid = mode(device.mod)
		}
		if !valid {
			continue
		}
		for _, pair := range device.beh {
			if skip[pair[0]] {
				continue
			}
			lnMod, ok := mode(pair[1])
			if !ok || pair[1] == device.mod {
				lnMod = ldMod
			}
			target := deriveBeh(ldMod, lnMod)
			if onlyChanged && pyval.Equal(current(pair[0]), target) {
				continue
			}
			writes = append(writes, item{pair[0], target, nil})
		}
	}
	return writes
}

// importedModes puts every Mod and Beh back to its imported (SCL) value in the table: a worker
// restarts from those values while the table keeps the last run's, and Beh must follow what the
// wire holds. Mod and Beh are no start kinds, so nothing restores them into the worker.
func importedModes(target *running) {
	if target == nil {
		return
	}
	for i, entry := range target.model.signals {
		if isModRef(entry.Ref) || strings.HasSuffix(entry.Ref, ".Beh.stVal") {
			target.table.set(entry.Ref, pyval.Get(target.model.rawSignals[i], "value"))
		}
	}
}

func sortedDeviceKeys(devices map[string]*modeDevice) []string {
	keys := make([]string, 0, len(devices))
	for key := range devices {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// withBehaviour adds the Beh writes that Mod items imply, for a batch about to be sent.
func withBehaviour(target *running, items []item) []item {
	pending, skip := map[string]any{}, map[string]bool{}
	for _, entry := range items {
		if isModRef(entry.ref) {
			pending[entry.ref] = entry.value
		}
		if strings.HasSuffix(entry.ref, ".Beh.stVal") {
			skip[entry.ref] = true
		}
	}
	if len(pending) == 0 {
		return items
	}
	current := func(ref string) any {
		value, _ := target.table.get(ref)
		return value
	}
	return append(items, behaviourWrites(target.model, current, pending, skip, true)...)
}

// syncBehaviour writes each Beh of a running simulated IED that differs from what its Mods
// imply, comparing with the table (at Run importedModes first made it match the worker).
func (c *Coordinator) syncBehaviour(iedID string) error {
	target := c.simulatedRun(iedID)
	if target == nil {
		return nil
	}
	current := func(ref string) any {
		value, _ := target.table.get(ref)
		return value
	}
	writes := behaviourWrites(target.model, current, nil, nil, true)
	if len(writes) == 0 {
		return nil
	}
	_, err := c.apply(iedID, writes, behaviourSource)
	return err
}
