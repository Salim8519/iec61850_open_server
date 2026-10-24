package scl

import (
	"errors"
	"strconv"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

// errTypeError marks Python TypeErrors (no Flask handler: HTTP 500 in Python too).
var errTypeError = errors.New("int() argument must be a string, a bytes-like object or a real number, not 'NoneType'")

var writableFCs = map[string]bool{"ST": true, "MX": true, "SP": true, "CF": true, "DC": true}

// attribute is ModelBuilder.attribute (DA and BDA, Struct recursion, signal metadata).
func (b *builder) attribute(e *xmldoc.Element, reference string, instance *xmldoc.Element,
	inheritedFC string, hasInheritedFC bool, depth int, inheritedTriggers int, description string) error {
	if depth > 32 {
		return apperr.New("SCL attribute nesting exceeds 32 levels")
	}
	value, present := e.Get("name")
	name, err := Identifier(value, present, "data attribute")
	if err != nil {
		return located(err, reference)
	}
	kind, hasKind := e.Get("bType")
	typeNumber, known := basicTypes[kind]
	if !hasKind && adr0114() {
		return located(apperr.New("the attribute has no bType"), reference+"."+name)
	}
	if !hasKind || !known {
		return apperr.New("Unsupported basic type " + noneOr(kind, hasKind) + " at " + reference + "." + name)
	}
	fc, hasFC := e.Get("fc")
	if !hasFC {
		fc, hasFC = inheritedFC, hasInheritedFC
	}
	fcNumber, knownFC := functionalConstraints[fc]
	if !hasFC && adr0114() {
		return located(apperr.New("the attribute has no functional constraint (fc)"), reference+"."+name)
	}
	if !hasFC || !knownFC {
		return located(apperr.New("Unknown functional constraint "+noneOr(fc, hasFC)), reference+"."+name)
	}
	count, err := pyCount(e)
	if err != nil {
		return located(err, reference+"."+name)
	}
	if count != 0 {
		return apperr.New("Array attribute " + reference + "." + name + " needs array-attribute support")
	}
	ref := reference + "." + name
	description = firstTruthy(instanceDesc(instance), e.GetOr("desc", ""), description)
	triggers := triggerBits(e.Get) | inheritedTriggers
	if standard := defaultTriggers(e, name, fc); standard != 0 {
		triggers |= standard
		b.notes.defaultTriggers++
	}
	header := "DA(" + name + " 0 " + strconv.Itoa(typeNumber) + " " + strconv.Itoa(fcNumber) + " " +
		strconv.Itoa(triggers) + " 0)"
	if kind == "Struct" {
		b.lines = append(b.lines, header+"{")
		typeID, hasType := e.Get("type")
		definition, err := b.declaration("DAType", typeID, hasType)
		if err != nil {
			return located(err, ref)
		}
		for _, child := range definition.Children {
			if child.Tag != "BDA" {
				continue
			}
			childName, hasName := child.Get("name")
			override := overrideOf(instance, childName, hasName)
			if err := b.attribute(child, ref, override, fc, true, depth+1, triggers, description); err != nil {
				return err
			}
		}
		b.lines = append(b.lines, "}")
		return nil
	}
	enums := b.emptyEnums
	if kind == "Enum" {
		if enums, err = b.enumValues(e); err != nil {
			return located(err, ref)
		}
	}
	text, found := "", false
	if instance != nil {
		text, found = instance.FindText("Val")
	}
	if !found {
		text, found = e.FindText("Val")
	}
	initial, err := initialValue(kind, text, found, enums)
	if err != nil {
		return located(err, ref)
	}
	if value, isText := initial.(string); isText {
		if simulated, changed := simulatedText(value); changed {
			initial = simulated
			b.notes.textReplaced = append(b.notes.textReplaced, ref)
		}
	}
	suffix, err := configInitial(kind, initial)
	if err != nil {
		return located(err, ref)
	}
	b.lines = append(b.lines, header+suffix+";")
	signal := pyjson.NewObjectSize(9).Set("ref", ref).Set("fc", fc).Set("kind", kind).Set("value", initial).
		Set("description", description).Set("enums", enums).Set("writable", writableFCs[fc]).
		Set("quality", int64(0)).Set("timestamp", int64(0))
	b.signals = append(b.signals, signal)
	if b.byReference[ref] != nil {
		return apperr.New("Ambiguous duplicate data reference " + ref)
	}
	b.byReference[ref] = signal
	return nil
}

// enumValues is {EnumVal text.strip(): int(ord)} for the DA's EnumType.
func (b *builder) enumValues(e *xmldoc.Element) (*pyjson.Object, error) {
	typeID, hasType := e.Get("type")
	if !hasType && adr0114() {
		return nil, apperr.New("bType is Enum but the attribute names no EnumType (type attribute missing)")
	}
	if shared := b.enumTypes[typeID]; shared != nil && hasType {
		return shared, nil
	}
	definition, err := b.declaration("EnumType", typeID, hasType)
	if err != nil {
		return nil, err
	}
	enums := pyjson.NewObject()
	for _, item := range definition.Children {
		if item.Tag != "EnumVal" {
			continue
		}
		label := pyrules.Strip(item.Text)
		if label == "" {
			continue
		}
		ord, present := item.Get("ord")
		if !present && !adr0114() {
			return nil, errTypeError
		}
		if !present {
			return nil, apperr.New("EnumType " + typeID + " value " + pyrules.Repr(label) + " has no ord")
		}
		number, err := pyrules.IntBig(ord, 10)
		if err != nil {
			return nil, apperr.ValueError(pyrules.IntMessage(ord))
		}
		enums.Set(label, intValue(number))
	}
	if hasType {
		b.enumTypes[typeID] = enums
	}
	return enums, nil
}
