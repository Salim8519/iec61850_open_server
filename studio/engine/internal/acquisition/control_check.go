package acquisition

import (
	"strings"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// Device control check (ADR-0138): after connecting, the live session reads, read-only, how
// the real device itself is set up to take each command of the CID (ctlModel, sboTimeout) and
// whether it is in Local (Loc.stVal). The device wins: commands use its own model at send
// time, so a difference is shown, never hidden, and an unreadable value is never replaced by
// the CID value.

// controlCheckLimit bounds one check; a bigger model checks its first controls and says so.
const controlCheckLimit = 500

// modelWords are the IEC 61850-7-2 ctlModel values 0 to 4 in plain words.
var modelWords = []string{"status only (no commands)", "direct, normal security", "select before operate, normal security",
	"direct, enhanced security", "select before operate, enhanced security"}

// typedControlsOf is the IED's typed control list (BSC, ISC, APC, INC, ...) from its stored
// SCL; a variable so tests can give an IED a list without writing SCL.
var typedControlsOf = func(ied *pyjson.Object) []any {
	list, err := scl.CachedExtendedControls(pyv.GetStr(ied, "id"), pyv.GetStr(ied, "name"), pyv.GetStr(ied, "source"))
	if err != nil {
		return nil
	}
	return list
}

// checkedControls is every control object of the CID once: SPC/DPC first, then typed ones.
func checkedControls(ied *pyjson.Object) []*pyjson.Object {
	seen, result := map[string]bool{}, []*pyjson.Object{}
	for _, control := range pyv.Objects(append(pyv.GetList(ied, "controls"), typedControlsOf(ied)...)) {
		ref := pyv.GetStr(control, "ref")
		if ref != "" && !seen[ref] {
			seen[ref] = true
			result = append(result, control)
		}
	}
	return result
}

func isSBO(model int64) bool { return model == 2 || model == 4 }

// modelOf is a ctlModel as a number 0..4; ok is false for anything else.
func modelOf(value any) (int64, bool) {
	number, ok := pyv.Int64(value)
	return number, ok && number >= 0 && number <= 4
}

// locRefs are the Loc.stVal points of the CID that decide whether these controls are in Local:
// LLN0 of each control's logical device and the control's own logical node.
func locRefs(ied *pyjson.Object, controls []*pyjson.Object) []string {
	wanted := map[string]bool{}
	for _, control := range controls {
		node, _, _ := strings.Cut(pyv.GetStr(control, "ref"), ".")
		device, _, _ := strings.Cut(node, "/")
		wanted[node+".Loc.stVal"], wanted[device+"/LLN0.Loc.stVal"] = true, true
	}
	refs := []string{}
	for _, signal := range pyv.Objects(pyv.GetList(ied, "signals")) {
		ref := pyv.GetStr(signal, "ref")
		if wanted[ref] && pyv.Equal(pyv.Get(signal, "fc"), "ST") {
			refs, wanted[ref] = append(refs, ref), false
		}
	}
	return refs
}

// controlCheckItems are the read requests: ctlModel of every control, sboTimeout of the SBO
// ones, and the Loc points.
func controlCheckItems(controls []*pyjson.Object, locs []string) []any {
	items := []any{}
	for _, control := range controls {
		ref := pyv.GetStr(control, "ref")
		items = append(items, pyv.O("ref", ref+".ctlModel", "fc", "CF"))
		if model, _ := modelOf(pyv.Get(control, "model")); isSBO(model) {
			items = append(items, pyv.O("ref", ref+".sboTimeout", "fc", "CF"))
		}
	}
	for _, ref := range locs {
		items = append(items, pyv.O("ref", ref, "fc", "ST"))
	}
	return items
}

// checkControls reads the check on this association. It never ends the session: a failed
// read is kept as the check's error and the values keep flowing.
func (s *RealSession) checkControls(process Worker) {
	check, err := readControlCheck(process, s.ied, s.deps.Clock.Now)
	if err != nil {
		check = pyv.O("error", "ScadaBench could not read the device's control settings: "+err.Error(),
			"checked_at", pyv.ISOSeconds(s.deps.Clock.Now()), "source", "mms_read")
	}
	s.mu.Lock()
	s.controls = check
	s.mu.Unlock()
}

// Controls is the latest device control check, nil before the first read.
func (s *RealSession) Controls() *pyjson.Object {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.controls
}

// readControlCheck reads the device and returns the check (empty when the CID has no control).
func readControlCheck(process Worker, ied *pyjson.Object, now func() time.Time) (*pyjson.Object, error) {
	controls := checkedControls(ied)
	limited := len(controls) > controlCheckLimit
	if limited {
		controls = controls[:controlCheckLimit]
	}
	locs := locRefs(ied, controls)
	values, errors, err := readItems(process, controlCheckItems(controls, locs))
	if err != nil {
		return nil, err
	}
	check := compareControls(controls, locs, values, errors)
	return check.Set("limited", limited).Set("checked_at", pyv.ISOSeconds(now())).Set("source", "mms_read"), nil
}
