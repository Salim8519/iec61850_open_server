package scl

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/xmldoc"
)

// extendedShape is where one command class (IEC 61850-7-3) keeps its value and feedback.
type extendedShape struct {
	feedback      string          // feedback leaf below the data object
	feedbackKinds map[string]bool // accepted SCL basic types of the feedback leaf
	ctlVal        string          // ctlVal leaf below the data object
	ctlKinds      map[string]bool // accepted SCL basic types of ctlVal
	valueKind     string          // what a client sends: int, enum, float or step
	minRef        string          // optional lower limit (CF), "" when the class has none
	maxRef        string
	stepRef       string
	posLimits     bool // tap positions: -64..63 unless the SCL sets narrower limits
	transInd      string
}

func kinds(names ...string) map[string]bool {
	set := map[string]bool{}
	for _, name := range names {
		set[name] = true
	}
	return set
}

// extendedShapes are the classes beyond SPC/DPC that ScadaBench can command (ADR-0104).
// APC/BAC use the float member "f" of AnalogueValue only.
var extendedShapes = map[string]extendedShape{
	"INC": {feedback: ".stVal", feedbackKinds: kinds("INT32"), ctlVal: ".Oper.ctlVal", ctlKinds: kinds("INT32"),
		valueKind: "int", minRef: ".minVal", maxRef: ".maxVal", stepRef: ".stepSize"},
	"ENC": {feedback: ".stVal", feedbackKinds: kinds("Enum"), ctlVal: ".Oper.ctlVal", ctlKinds: kinds("Enum"),
		valueKind: "enum"},
	"APC": {feedback: ".mxVal.f", feedbackKinds: kinds("FLOAT32"), ctlVal: ".Oper.ctlVal.f", ctlKinds: kinds("FLOAT32"),
		valueKind: "float", minRef: ".minVal.f", maxRef: ".maxVal.f", stepRef: ".stepSize.f"},
	"BAC": {feedback: ".mxVal.f", feedbackKinds: kinds("FLOAT32"), ctlVal: ".Oper.ctlVal", ctlKinds: kinds("Tcmd"),
		valueKind: "step", minRef: ".minVal.f", maxRef: ".maxVal.f", stepRef: ".stepSize.f"},
	"BSC": {feedback: ".valWTr.posVal", feedbackKinds: kinds("INT8"), ctlVal: ".Oper.ctlVal", ctlKinds: kinds("Tcmd"),
		valueKind: "step", minRef: ".minVal", maxRef: ".maxVal", posLimits: true, transInd: ".valWTr.transInd"},
	"ISC": {feedback: ".valWTr.posVal", feedbackKinds: kinds("INT8"), ctlVal: ".Oper.ctlVal", ctlKinds: kinds("INT8"),
		valueKind: "int", minRef: ".minVal", maxRef: ".maxVal", posLimits: true, transInd: ".valWTr.transInd"},
}

// extendedControl records a commandable INC/ENC/APC/BAC/BSC/ISC object. Objects that do not
// match their class shape are left out silently: the IED's warnings stay exactly Python's.
func (b *builder) extendedControl(definition *xmldoc.Element, ref string) {
	cdc, _ := definition.Get("cdc")
	shape, known := extendedShapes[cdc]
	if !known {
		return
	}
	model := b.byReference[ref+".ctlModel"]
	feedback := b.byReference[ref+shape.feedback]
	ctlVal := b.byReference[ref+shape.ctlVal]
	if model == nil || feedback == nil || ctlVal == nil || !truthy(get(model, "value")) {
		return
	}
	feedbackKind, _ := get(feedback, "kind").(string)
	ctlKind, _ := get(ctlVal, "kind").(string)
	if !shape.feedbackKinds[feedbackKind] || !shape.ctlKinds[ctlKind] {
		return
	}
	timeout := any(int64(0))
	if found := b.byReference[ref+".sboTimeout"]; found != nil {
		timeout = get(found, "value")
	}
	low, high := b.limits(ref, shape)
	labels := pyjson.NewObject()
	if shape.valueKind == "enum" {
		labels, _ = get(ctlVal, "enums").(*pyjson.Object)
		if labels == nil || labels.Len() == 0 {
			labels, _ = get(feedback, "enums").(*pyjson.Object)
		}
		if labels == nil {
			labels = pyjson.NewObject()
		}
	}
	var transInd any
	if shape.transInd != "" && b.byReference[ref+shape.transInd] != nil {
		transInd = ref + shape.transInd
	}
	control := pyjson.NewObject().Set("ref", ref).Set("status", get(feedback, "ref")).Set("cdc", cdc).
		Set("ctl_val_kind", ctlKind).Set("ctl_val_ref", get(ctlVal, "ref")).Set("value_kind", shape.valueKind).
		Set("model", get(model, "value")).Set("sbo_timeout_ms", timeout).Set("min", low).Set("max", high).
		Set("step", b.step(ref, shape)).Set("labels", labels).Set("trans_ind", transInd)
	b.extended = append(b.extended, control)
}

// limits are the SCL's minVal/maxVal when both are set and ordered (an attribute without
// Val reads 0, so 0/0 means "not set"); tap positions fall back to -64..63.
func (b *builder) limits(ref string, shape extendedShape) (any, any) {
	var low, high any
	if shape.posLimits {
		low, high = int64(-64), int64(63)
	}
	if shape.minRef == "" {
		return low, high
	}
	minimum, okMin := signalNumber(b.byReference[ref+shape.minRef])
	maximum, okMax := signalNumber(b.byReference[ref+shape.maxRef])
	if okMin && okMax && maximum > minimum {
		return numeric(b.byReference[ref+shape.minRef]), numeric(b.byReference[ref+shape.maxRef])
	}
	return low, high
}

// step is stepSize when set and positive; steps of tap changers are always one position.
func (b *builder) step(ref string, shape extendedShape) any {
	if shape.posLimits {
		return int64(1)
	}
	if shape.stepRef == "" {
		return nil
	}
	if size, ok := signalNumber(b.byReference[ref+shape.stepRef]); ok && size > 0 {
		return numeric(b.byReference[ref+shape.stepRef])
	}
	return nil
}

func numeric(signal *pyjson.Object) any { return get(signal, "value") }

// signalNumber reads an int64 or float64 signal value.
func signalNumber(signal *pyjson.Object) (float64, bool) {
	if signal == nil {
		return 0, false
	}
	switch value := get(signal, "value").(type) {
	case int64:
		return float64(value), true
	case float64:
		return value, true
	}
	return 0, false
}
