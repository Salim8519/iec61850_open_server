package runtime

import (
	"regexp"
	"sort"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Mode and behaviour writes of simulated IEDs (domain/test_mode.py mode_points, mode_writes;
// simulation/test_mode_apply.py). Mod/Beh: 1 on, 2 on-blocked, 3 test, 4 test/blocked, 5 off.
var (
	simPattern = regexp.MustCompile(`^([^/]+)/LPHD\d*\.Sim\.stVal$`)
	behPattern = regexp.MustCompile(`^([^/]+)/([^.]+)\.Beh\.stVal$`)
)

func isMode(value any) bool {
	number, ok := value.(int64)
	return ok && number >= 1 && number <= 5
}

func deriveBeh(ldMod, lnMod int64) int64 {
	if ldMod == 5 || lnMod == 5 {
		return 5
	}
	test := ldMod == 3 || ldMod == 4 || lnMod == 3 || lnMod == 4
	blocked := ldMod == 2 || ldMod == 4 || lnMod == 2 || lnMod == 4
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

type modeDevice struct {
	mod string
	beh [][2]string // (Beh reference, own Mod reference)
}

func modePoints(model *iedModel) (map[string]*modeDevice, []string) {
	refs := make([]string, 0, len(model.index))
	for ref := range model.index {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	devices := map[string]*modeDevice{}
	var simulation []string
	for _, ref := range refs {
		if match := behPattern.FindStringSubmatch(ref); match != nil {
			device := devices[match[1]]
			if device == nil {
				device = &modeDevice{}
				devices[match[1]] = device
			}
			device.beh = append(device.beh, [2]string{ref, match[1] + "/" + match[2] + ".Mod.stVal"})
		}
		if simPattern.MatchString(ref) {
			simulation = append(simulation, ref)
		}
	}
	for domain, device := range devices {
		if _, ok := model.index[domain+"/LLN0.Mod.stVal"]; ok {
			device.mod = domain + "/LLN0.Mod.stVal"
		}
	}
	return devices, simulation
}

// modeWrites is mode_writes for stored settings {devices: {domain: mode}, simulation: bool}.
func modeWrites(model *iedModel, settings *pyjson.Object, current func(string) any) []item {
	devices, simulation := modePoints(model)
	var writes []item
	modes := pyval.Map(settings, "devices")
	for _, domain := range keysOf(modes) {
		device := devices[domain]
		if device == nil {
			continue
		}
		mode, _ := pyval.Get(modes, domain).(int64)
		if device.mod != "" {
			writes = append(writes, item{device.mod, mode, nil})
		}
		for _, pair := range device.beh {
			lnMod := any(mode)
			if pair[1] != device.mod {
				lnMod = current(pair[1])
				if lnMod == nil {
					lnMod = int64(1)
				}
			}
			if !isMode(lnMod) {
				lnMod = int64(1)
			}
			writes = append(writes, item{pair[0], deriveBeh(mode, lnMod.(int64)), nil})
		}
	}
	flag := pyval.Get(settings, "simulation")
	for _, ref := range simulation {
		writes = append(writes, item{ref, flag, nil})
	}
	return writes
}

// writeTestMode is test_mode_apply._write: the IED's saved modes, when it is a running simulated IED.
func (c *Coordinator) writeTestMode(ied *pyjson.Object) (int, error) {
	id := pyval.Str(ied, "id")
	target := c.simulatedRun(id)
	if roleOf(ied) != "simulated" || target == nil {
		return 0, nil
	}
	settings := pyval.Map(ied, "test_mode")
	if settings == nil || settings.Len() == 0 {
		settings = pyval.Obj("devices", pyjson.NewObject(), "simulation", false)
	}
	current := func(ref string) any {
		if !strings.HasSuffix(ref, ".Mod.stVal") {
			return nil
		}
		value, _ := target.table.get(ref)
		return value
	}
	writes := modeWrites(target.model, settings, current)
	if len(writes) > 0 {
		if _, err := c.apply(id, writes, "test_mode"); err != nil {
			return 0, err
		}
	}
	return len(writes), nil
}

// applyTestModes writes every simulated IED's saved modes at Run start (apply_test_modes),
// then derives each Beh from the Mods the IED now holds (ADR-0115).
func (c *Coordinator) applyTestModes(project *pyjson.Object, only map[string]bool) error {
	for _, ied := range projectIeds(project) {
		id := pyval.Str(ied, "id")
		if only != nil && !only[id] {
			continue
		}
		importedModes(c.simulatedRun(id)) // what the restarted worker holds
		settings := pyval.Map(ied, "test_mode")
		if pyval.Truthy(pyval.Get(settings, "devices")) || pyval.Truthy(pyval.Get(settings, "simulation")) {
			if _, err := c.writeTestMode(ied); err != nil {
				return err
			}
		}
		if err := c.syncBehaviour(id); err != nil {
			return err
		}
	}
	return nil
}

// ApplyTestMode writes one IED's (already saved) modes while running; it returns how many
// points were written (set_test_mode after the project actor stored the settings).
func (c *Coordinator) ApplyTestMode(iedID string) (int, error) {
	if !c.isLive() {
		return 0, nil
	}
	ied := findIed(c.opt.Project.Current(), iedID)
	if ied == nil {
		return 0, nil
	}
	return c.writeTestMode(ied)
}
