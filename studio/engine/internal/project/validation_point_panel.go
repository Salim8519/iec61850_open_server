package project

import (
	"math"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// PointPanelKind is the point panel (ADR-0102): one IED and up to 300 points to set or operate
// from one popup. Python does not know this kind.
const PointPanelKind = "POINT_PANEL"

// panelRefusal says a node of another kind carries panel fields.
const panelRefusal = "Only a point panel accepts an IED and points"

// checkPointPanel rejects panel fields on other kinds and checks a panel: one known IED, unique
// point ids, and for every point the signal or control it needs, of the right type and IED.
func checkPointPanel(data *pyjson.Object, kind string, ieds *Lookup) error {
	rows := items(pyval.Lookup(data, "points", nil))
	iedID := text(data, "ied_id")
	if kind != PointPanelKind {
		if len(rows) > 0 || iedID != "" {
			return apperr.New(panelRefusal)
		}
		return nil
	}
	if pyval.Truthy(pyval.Lookup(data, "state_mapping", nil)) {
		return apperr.New("A point panel uses its own points instead of a single state mapping")
	}
	if iedID == "" {
		if len(rows) > 0 {
			return apperr.New("Choose the IED of this point panel before adding points")
		}
		return nil
	}
	ied := ieds.IED(iedID)
	if ied == nil {
		return apperr.New("A point panel references an unknown IED. Choose another IED or import it again")
	}
	seen := map[string]bool{}
	for _, item := range rows {
		entry := object(item)
		id := text(entry, "id")
		if seen[id] {
			return apperr.New("Point identifiers must be unique inside a point panel")
		}
		seen[id] = true
		if err := checkPoint(entry, ied, iedID, ieds); err != nil {
			return err
		}
	}
	return nil
}

// number reads a validated JSON number (the contract keeps floats and whole numbers apart).
func number(value any) (float64, bool) {
	switch x := value.(type) {
	case float64:
		return x, true
	case int64:
		return float64(x), true
	}
	return 0, false
}

// checkPoint checks one row against the panel's IED.
func checkPoint(entry, ied *pyjson.Object, iedID string, ieds *Lookup) error {
	name, kind := "Point \""+text(entry, "label")+"\"", text(entry, "type")
	if err := checkPointSettings(entry, name, kind); err != nil {
		return err
	}
	source := object(pyval.Lookup(entry, "source", nil))
	control := object(pyval.Lookup(entry, "control", nil))
	if kind == "command" {
		if source != nil {
			return apperr.New(name + " is a command, so it uses a control and no signal")
		}
		return checkPointControl(control, name, text(entry, "unit"), ied, iedID, ieds)
	}
	if control != nil {
		return apperr.New(name + " is not a command, so it uses a signal and no control")
	}
	if source == nil {
		return apperr.New(name + " needs a signal. Add the point again from the IED's signal list")
	}
	if text(source, "ied_id") != iedID {
		return apperr.New(name + " reads from another IED. A point panel holds the points of one IED; " +
			"make another panel for the other IED")
	}
	ref := text(source, "ref")
	signal := ieds.Entry(pyval.Lookup(ied, "signals", nil), ref)
	if signal == nil {
		return apperr.New(name + " uses a signal this IED does not have: " + ref + ". Remove the point or import the IED again")
	}
	if _, err := bindingSignal(source, ieds); err != nil {
		return err
	}
	return checkPointSignal(name, kind, signal)
}

// checkPointSignal checks the signal type: status = BOOLEAN, double = Dbpos, value = numeric MX or a
// whole-number status (ST INT…: a tap position or a counter, change 0173).
func checkPointSignal(name, kind string, signal *pyjson.Object) error {
	have, ref := text(signal, "kind"), text(signal, "ref")
	switch kind {
	case "status":
		if have != "BOOLEAN" {
			return apperr.New(name + " is a status point and needs an on/off (Boolean) signal, but " + ref +
				" is " + have + ". Choose a single-point status such as stVal of an SPS, or use another point type")
		}
	case "double":
		if have != "Dbpos" {
			return apperr.New(name + " is a double point and needs a position (Dbpos) signal, but " + ref +
				" is " + have + ". Choose a double-point status such as stVal of a DPS, or use another point type")
		}
	default:
		fc := text(signal, "fc")
		wholeStatus := fc == "ST" && strings.HasPrefix(have, "INT")
		if !pyval.InTuple(have, numericKind...) || (fc != "MX" && !wholeStatus) {
			return apperr.New(name + " is a value point and needs a measured number (MX) or a whole-number status " +
				"(ST), but " + ref + " is " + have + " (" + fc + "). Choose a measured value such as mag.f or a position " +
				"such as valWTr.posVal")
		}
	}
	return nil
}

// checkPointControl checks a command's control: it exists on the panel's IED and is SPC or DPC.
func checkPointControl(control *pyjson.Object, name, unit string, ied *pyjson.Object, iedID string, ieds *Lookup) error {
	if control == nil {
		return apperr.New(name + " is a command and needs a control. Add it again from the IED's control list")
	}
	if text(control, "ied_id") != iedID {
		return apperr.New(name + " sends to another IED. A point panel holds the points of one IED; " +
			"make another panel for the other IED")
	}
	ref := text(control, "ref")
	entry := ieds.Entry(pyval.Lookup(ied, "controls", nil), ref)
	classes := []string{"SPC", "DPC"}
	if entry == nil {
		// Typed controls (ADR-0104, ADR-0108): integer, list, set-point and raise/lower commands.
		entry, classes = findByRef(extendedControlsOf(ied), ref), typedClasses
	}
	if entry == nil {
		return apperr.New(name + " uses a control this IED does not have: " + ref + ". Remove the point or import the IED again")
	}
	class := text(entry, "cdc")
	if !pyval.InTuple(class, classes...) {
		return apperr.New(name + " needs a single (SPC), double (DPC), integer (INC), list (ENC), set-point (APC) " +
			"or raise/lower (BAC, BSC, ISC) command control, but " + ref + " is " + class)
	}
	if unit != "" && class != "APC" {
		return apperr.New(name + " is a " + class + " command, so it cannot have a unit. Only a set point (APC) shows one")
	}
	return nil
}

// typedClasses are the control classes a command row may use from the typed list.
var typedClasses = []string{"SPC", "DPC", "INC", "ENC", "APC", "BAC", "BSC", "ISC"}

// extendedControlsOf is the IED's typed control list (ADR-0104 `extended_controls`), read from
// its stored SCL; it is never part of the IED payload. A variable only so tests can give an IED
// a list without writing SCL.
var extendedControlsOf = func(ied *pyjson.Object) any {
	list, err := scl.CachedExtendedControls(text(ied, "id"), text(ied, "name"), text(ied, "source"))
	if err != nil {
		return nil
	}
	return list
}

// checkPointSettings checks what only some types may carry: value settings and pins on value
// points, a healthy value of the type's own kind, none on commands.
func checkPointSettings(entry *pyjson.Object, name, kind string) error {
	scale, _ := number(pyval.Lookup(entry, "scale", nil))
	offset, _ := number(pyval.Lookup(entry, "offset", nil))
	decimals, _ := number(pyval.Lookup(entry, "decimals", nil))
	if kind != "value" {
		// A command's unit is checked against its control (only a set point, APC, shows one).
		if (kind != "command" && text(entry, "unit") != "") || scale != 1 || offset != 0 || decimals != 2 {
			return apperr.New(name + " is not a value point, so it cannot have a unit, scale, offset or decimals")
		}
		if pyval.Lookup(entry, "pinned", nil) == true {
			return apperr.New(name + " is not a value point, so it cannot be pinned to the card")
		}
	} else if scale == 0 {
		return apperr.New(name + " has scale 0, so its shown value could never change. Use a scale such as 1 or 0.001")
	}
	healthy := pyval.Lookup(entry, "healthy", nil)
	if healthy == nil {
		return nil
	}
	return checkHealthy(healthy, name, kind)
}

func checkHealthy(healthy any, name, kind string) error {
	_, isBool := healthy.(bool)
	value, isNumber := number(healthy)
	switch kind {
	case "status":
		if !isBool {
			return apperr.New(name + " is a status point, so its healthy value is on (true) or off (false)")
		}
	case "double":
		if !isNumber || value != math.Trunc(value) || value < 0 || value > 3 {
			return apperr.New(name + " is a double point, so its healthy value is 0 (intermediate), 1 (open), " +
				"2 (closed) or 3 (bad)")
		}
	case "value":
		if !isNumber {
			return apperr.New(name + " is a value point, so its healthy value is a number")
		}
	default:
		return apperr.New(name + " is a command, so it cannot have a healthy value")
	}
	return nil
}

// settlePointPanel gives a fresh point panel its empty list and no IED, and drops the panel
// fields of other kinds, so their saved bytes never change.
func settlePointPanel(data *pyjson.Object, kind string) {
	if kind == PointPanelKind {
		if _, present := data.Get("points"); !present {
			data.Set("points", []any{})
		}
		if _, present := data.Get("ied_id"); !present {
			data.Set("ied_id", nil)
		}
		return
	}
	data.Delete("points")
	data.Delete("ied_id")
}

// pointPanelUsesIED says whether the panel belongs to one of the IEDs.
func pointPanelUsesIED(data *pyjson.Object, ids map[string]bool) bool {
	return kindOf(data) == PointPanelKind && ids[text(data, "ied_id")]
}

// dropPointPanelOf empties a panel of a removed IED; the panel itself stays on the diagram.
func dropPointPanelOf(data *pyjson.Object, ids map[string]bool) {
	if data == nil || !pointPanelUsesIED(data, ids) {
		return
	}
	data.Set("ied_id", nil)
	data.Set("points", []any{})
}

func kindOf(data *pyjson.Object) string { return text(data, "kind") }
