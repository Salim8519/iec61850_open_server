package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// MeasurementKind is the measurement list panel (ADR-0090): named numeric rows with their
// own scale, offset, unit and decimals. Python does not know this kind.
const MeasurementKind = "MEASUREMENT_LIST"

// checkMeasurements rejects measurement rows on other kinds and checks the rows of a list:
// unique identifiers and numeric signals of known IEDs.
func checkMeasurements(data *pyjson.Object, kind string, ieds *Lookup) error {
	rows := items(pyval.Lookup(data, "measurements", nil))
	if kind != MeasurementKind {
		if len(rows) > 0 {
			return apperr.New("Only a measurement list accepts measurements")
		}
		return nil
	}
	seen := map[string]bool{}
	for _, item := range rows {
		entry := object(item)
		id := text(entry, "id")
		if seen[id] {
			return apperr.New("Measurement identifiers must be unique")
		}
		seen[id] = true
		signal, err := bindingSignal(object(pyval.Lookup(entry, "source", nil)), ieds)
		if err != nil {
			return err
		}
		if signalKind := text(signal, "kind"); !pyval.InTuple(signalKind, numericKind...) {
			return apperr.New("Measurement \"" + text(entry, "label") + "\" needs a number signal " +
				"(integer or float), but " + text(signal, "ref") + " is " + signalKind +
				". Choose a measured value such as mag.f")
		}
		// The point panel refuses scale 0 the same way (change 0176).
		if scale, ok := number(pyval.Lookup(entry, "scale", nil)); ok && scale == 0 {
			return apperr.New("Measurement \"" + text(entry, "label") + "\" has scale 0, so its shown value could " +
				"never change. Use a scale such as 1 or 0.001")
		}
	}
	return nil
}

// settleMeasurements gives a fresh measurement list its empty row list and drops an empty
// "measurements" key from other kinds, so their saved bytes never change.
func settleMeasurements(data *pyjson.Object, kind string) {
	if kind == MeasurementKind {
		if _, present := data.Get("measurements"); !present {
			data.Set("measurements", []any{})
		}
		return
	}
	data.Delete("measurements")
}

// measurementUsesIED says whether a row of the list reads from one of the IEDs.
func measurementUsesIED(data *pyjson.Object, ids map[string]bool) bool {
	for _, entry := range items(pyval.Lookup(data, "measurements", nil)) {
		if ids[text(object(pyval.Lookup(object(entry), "source", nil)), "ied_id")] {
			return true
		}
	}
	return false
}

// dropMeasurementsOf removes the rows that read from the IEDs; the list itself stays.
func dropMeasurementsOf(data *pyjson.Object, ids map[string]bool) {
	if data == nil {
		return
	}
	rows, present := data.Get("measurements")
	if !present {
		return
	}
	kept := []any{}
	for _, entry := range items(rows) {
		if !ids[text(object(pyval.Lookup(object(entry), "source", nil)), "ied_id")] {
			kept = append(kept, entry)
		}
	}
	data.Set("measurements", kept)
}
