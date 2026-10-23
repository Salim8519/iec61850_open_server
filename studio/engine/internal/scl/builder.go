package scl

import (
	"strconv"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

type typeKey struct {
	tag, id string
	hasID   bool
}

// builder is model_builder.ModelBuilder: cfg lines, signals and controls of one IED.
type builder struct {
	types           map[typeKey]*xmldoc.Element
	signals         []any
	byReference     map[string]*pyjson.Object
	controls        []any
	controlWarnings []string
	lines           []string
	extended        []any       // Go-only typed controls (ADR-0104); never part of the IED model
	notes           importNotes // Go-only import notes (ADR-0114)
}

func newBuilder(types map[typeKey]*xmldoc.Element) *builder {
	return &builder{types: types, byReference: map[string]*pyjson.Object{}, signals: []any{}, controls: []any{}}
}

func (b *builder) declaration(tag string, id string, present bool) (*xmldoc.Element, error) {
	if found := b.types[typeKey{tag, id, present}]; found != nil {
		return found, nil
	}
	if !adr0114() {
		return nil, apperr.New("Unresolved " + tag + " " + noneOr(id, present))
	}
	if !present {
		return nil, apperr.New("Unresolved " + tag + ": the type attribute is missing")
	}
	return nil, apperr.New("Unresolved " + tag + " " + id + " (not in DataTypeTemplates)")
}

// noneOr formats an optional attribute the way an f-string formats None.
func noneOr(value string, present bool) string {
	if !present {
		return "None"
	}
	return value
}

// pyCount is int(element.get("count", "0")) with Python's ValueError text.
func pyCount(e *xmldoc.Element) (int64, error) {
	text := e.GetOr("count", "0")
	value, err := pyrules.Int(text, 10)
	if err != nil {
		if pyrules.IsOverflow(err) {
			return 1 << 62, nil
		}
		return 0, apperr.ValueError(pyrules.IntMessage(text))
	}
	return value, nil
}

func (b *builder) dataObject(e *xmldoc.Element, reference string, instance *xmldoc.Element, depth int) error {
	if depth > 32 {
		return apperr.New("SCL type nesting exceeds 32 levels")
	}
	value, present := e.Get("name")
	name, err := Identifier(value, present, "data object")
	if err != nil {
		return located(err, reference)
	}
	ref := reference + "." + name
	count, err := pyCount(e)
	if err != nil {
		return located(err, ref)
	}
	if count < 0 || count > 256 {
		return located(apperr.New("Array element count must be between 0 and 256"), ref)
	}
	typeID, hasType := e.Get("type")
	definition, err := b.declaration("DOType", typeID, hasType)
	if err != nil {
		return located(err, ref)
	}
	b.lines = append(b.lines, "DO("+name+" "+strconv.FormatInt(count, 10)+"){")
	description := e.GetOr("desc", "")
	if count > 0 {
		for index := int64(0); index < count; index++ {
			b.lines = append(b.lines, "["+strconv.FormatInt(index, 10)+"]{")
			indexed := ref + "(" + strconv.FormatInt(index, 10) + ")"
			if err := b.objectContent(definition, indexed, instance, depth, description); err != nil {
				return err
			}
			b.lines = append(b.lines, "}")
		}
	} else if err := b.objectContent(definition, ref, instance, depth, description); err != nil {
		return err
	}
	b.lines = append(b.lines, "}")
	return nil
}

func (b *builder) objectContent(definition *xmldoc.Element, ref string, instance *xmldoc.Element, depth int, description string) error {
	description = firstTruthy(instanceDesc(instance), description, definition.GetOr("desc", ""))
	for _, child := range definition.Children {
		name, hasName := child.Get("name")
		override := overrideOf(instance, name, hasName)
		switch child.Tag {
		case "SDO":
			if err := b.dataObject(child, ref, override, depth+1); err != nil {
				return err
			}
		case "DA":
			if err := b.attribute(child, ref, override, "", false, depth+1, 0, description); err != nil {
				return err
			}
		}
	}
	b.control(definition, ref)
	b.extendedControl(definition, ref)
	return nil
}

// control exposes an SPC/DPC control whose model, feedback and ctlVal are consistent.
func (b *builder) control(definition *xmldoc.Element, ref string) {
	model := b.byReference[ref+".ctlModel"]
	feedback := b.byReference[ref+".stVal"]
	if model == nil || !truthy(get(model, "value")) {
		return
	}
	cdc, _ := definition.Get("cdc")
	switchClass := cdc == "SPC" || cdc == "DPC"
	if feedback == nil {
		if switchClass {
			b.notes.commandOnly = append(b.notes.commandOnly, ref) // ADR-0114: never silent
		}
		return
	}
	feedbackKind := get(feedback, "kind")
	if feedbackKind != "BOOLEAN" && feedbackKind != "Dbpos" {
		if switchClass {
			b.notes.odd(ref, cdc, "stVal", feedbackKind)
		}
		return
	}
	ctlVal := b.byReference[ref+".Oper.ctlVal"]
	expectedOK := cdc == "SPC" || cdc == "DPC"
	feedbackOK := cdc == "SPC" && feedbackKind == "BOOLEAN" || cdc == "DPC" && feedbackKind == "Dbpos"
	if expectedOK && feedbackOK && ctlVal != nil && get(ctlVal, "kind") == "BOOLEAN" {
		timeout := any(int64(0))
		if found := b.byReference[ref+".sboTimeout"]; found != nil {
			timeout = get(found, "value")
		}
		control := pyjson.NewObject().Set("ref", ref).Set("status", get(feedback, "ref")).Set("cdc", cdc).
			Set("ctl_val_kind", get(ctlVal, "kind")).Set("model", get(model, "value")).
			Set("sbo_timeout_ms", timeout).Set("delay_ms", int64(0)).Set("blocked", false)
		b.controls = append(b.controls, control)
		return
	}
	b.controlWarnings = append(b.controlWarnings,
		"Control "+ref+" is not exposed: expected a consistent SPC/DPC Boolean control model")
	b.notes.markReported(ref)
}

func get(o *pyjson.Object, key string) any {
	value, _ := o.Get(key)
	return value
}

// truthy is Python truthiness for model values.
func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int64:
		return v != 0
	case int:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	}
	return true
}

func instanceDesc(instance *xmldoc.Element) string {
	if instance == nil {
		return ""
	}
	return instance.GetOr("desc", "")
}

func firstTruthy(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// overrideOf is ModelBuilder._override: the first instance child with that name.
func overrideOf(parent *xmldoc.Element, name string, hasName bool) *xmldoc.Element {
	if parent == nil {
		return nil
	}
	for _, child := range parent.Children {
		value, present := child.Get("name")
		if present == hasName && value == name {
			return child
		}
	}
	return nil
}
