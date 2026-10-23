package scl

import (
	"math"
	"strconv"
	"strings"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/value"
	"studio-engine/internal/scl/xmldoc"
)

// Export is exporter.export_ied(source, name, overrides): the original document with the
// named IED's explicit scalar values edited, re-validated and round-trip verified.
func (im *Importer) Export(source string, name string, overrides any) ([]byte, error) {
	edits, ok := overrides.(*pyjson.Object)
	if !ok || edits.Len() > 50000 {
		return nil, apperr.New("SCL value overrides must be an object of at most 50,000 references")
	}
	raw := []byte(source)
	models, err := im.Import(raw, "import.scd", Selection{Names: []any{name}})
	if err != nil {
		return nil, err
	}
	original := models[0]
	if err := requireValid(get(original, "validation").(*pyjson.Object)); err != nil {
		return nil, err
	}
	doc, err := xmldoc.Parse(raw, xmldoc.Options{Nodes: true})
	if err != nil {
		return nil, apperr.New("Invalid or unsafe XML: " + err.Error())
	}
	var ied *xmldoc.Element
	for _, item := range doc.Root.Children {
		if inSCL(item, "IED") && item.GetOr("name", "\x00") == name {
			ied = item
			break
		}
	}
	nodes := logicalNodes(doc.Root, ied, name)
	signals := map[string]*pyjson.Object{}
	expected := map[string]any{}
	for _, item := range get(original, "signals").([]any) {
		signal := item.(*pyjson.Object)
		ref := get(signal, "ref").(string)
		signals[ref] = signal
		expected[ref] = get(signal, "value")
	}
	for _, reference := range edits.Keys() {
		requested, _ := edits.Get(reference)
		if err := applyOverride(nodes, signals, reference, requested, expected); err != nil {
			return nil, err
		}
	}
	payload := doc.SerializeLXML()
	validation, err := Validate(payload, im.Validator)
	if err != nil {
		return nil, err
	}
	if err := requireValid(validation); err != nil {
		return nil, err
	}
	restored, err := im.Import(payload, "import.scd", Selection{Names: []any{name}})
	if err != nil {
		return nil, err
	}
	if !sameSignals(get(restored[0], "signals").([]any), signals, expected) {
		return nil, apperr.New("Export failed signal type/reference/value round-trip verification")
	}
	for _, key := range []string{"datasets", "reports"} {
		if !pyEqual(get(original, key), get(restored[0], key)) {
			return nil, apperr.New("Export changed dataset or report definitions unexpectedly")
		}
	}
	return payload, nil
}

func requireValid(validation *pyjson.Object) error {
	if get(validation, "status") == "valid" {
		return nil
	}
	first := get(validation, "errors").([]any)[0].(*pyjson.Object)
	detail := get(first, "message").(string)
	if !adr0114() {
		return apperr.New("Verified SCL export requires a supported, schema-valid document: " + detail)
	}
	if line, ok := get(first, "line").(int64); ok && line > 0 {
		detail = "line " + strconv.FormatInt(line, 10) + ": " + detail
	}
	// ADR-0114: plain words first, the raw XSD message after "Detail".
	if get(validation, "status") == "unsupported" {
		return apperr.New("This file cannot be exported: no bundled IEC 61850 schema (XSD) matches its SCL " +
			"version, so ScadaBench cannot check that the exported file is correct. Detail: " + detail)
	}
	return apperr.New("This file cannot be exported: it does not pass the IEC 61850 schema check (XSD), so " +
		"ScadaBench cannot prove the exported file is correct. Fix the file in the engineering tool and import " +
		"it again. Detail: " + detail)
}

func applyOverride(nodes map[[2]string]path, signals map[string]*pyjson.Object, reference string,
	requested any, expected map[string]any) error {
	signal := signals[reference]
	if signal == nil {
		return apperr.New("Unknown export reference: " + reference)
	}
	checked, err := value.Validate(signal, requested)
	if err != nil {
		return apperr.New(reference + ": " + err.Error())
	}
	text, err := valueText(signal, checked)
	if err != nil {
		return err
	}
	domain, rest, _ := strings.Cut(reference, "/")
	parts := strings.Split(rest, ".")
	for _, part := range parts {
		if strings.Contains(part, "(") {
			return apperr.New("Per-index SCL export is not supported yet: " + reference)
		}
	}
	current, found := nodes[[2]string{domain, parts[0]}]
	if !found {
		return &apperr.Error{Kind: apperr.Key, Message: "('" + domain + "', '" + parts[0] + "')"}
	}
	for index, part := range parts[1:] {
		tag := "SDI"
		if index == 0 {
			tag = "DOI"
		} else if index == len(parts)-2 {
			tag = "DAI"
		}
		if current, err = instance(current, tag, part); err != nil {
			return err
		}
	}
	owner := current.last()
	var values []*xmldoc.Element
	for _, child := range owner.Children {
		if inSCL(child, "Val") {
			values = append(values, child)
		}
	}
	grouped := false
	for _, item := range values {
		if _, ok := item.Get("sGroup"); ok {
			grouped = true
		}
	}
	if len(values) > 1 || grouped {
		return apperr.New("Setting-group values need an explicit group selection: " + reference)
	}
	target := (*xmldoc.Element)(nil)
	if len(values) == 1 {
		target = values[0]
	} else {
		target = &xmldoc.Element{Tag: "Val", Space: sclNamespace, Prefix: current.sclPrefix()}
		owner.Nodes = append(owner.Nodes, xmldoc.Node{Kind: xmldoc.ElementNode, Elem: target})
		owner.Children = append(owner.Children, target)
	}
	setText(target, text)
	expected[reference] = checked
	return nil
}

// valueText is exporter._value_text.
func valueText(signal *pyjson.Object, v any) (string, error) {
	kind := get(signal, "kind").(string)
	ref := get(signal, "ref").(string)
	switch {
	case kind == "BOOLEAN":
		if v.(bool) {
			return "true", nil
		}
		return "false", nil
	case kind == "Enum":
		enums := get(signal, "enums").(*pyjson.Object)
		for _, label := range enums.Keys() {
			if ordinal, _ := enums.Get(label); pyEqual(ordinal, v) {
				return label, nil
			}
		}
		return "", apperr.New("Export requires a declared enumeration value: " + ref)
	case kind == "Timestamp":
		return isoMillis(v.(int64)), nil
	case strings.HasPrefix(kind, "INT") || strings.HasPrefix(kind, "FLOAT") || isTextKind(kind):
		return pyStr(v), nil
	}
	return "", apperr.New("SCL value encoding for " + kind + " is not supported yet: " + ref)
}

// isoMillis is datetime.fromtimestamp(ms / 1000, timezone.utc).isoformat(timespec="milliseconds").
func isoMillis(ms int64) string {
	seconds := float64(ms) / 1000
	whole, fraction := math.Modf(seconds)
	micros := math.RoundToEven(fraction * 1e6)
	if micros >= 1e6 {
		micros -= 1e6
		whole++
	}
	t := time.Unix(int64(whole), int64(micros)*1000).UTC()
	return t.Format("2006-01-02T15:04:05") + "." +
		threeDigits(int64(micros)/1000) + "+00:00"
}

func threeDigits(n int64) string {
	return string([]byte{byte('0' + n/100), byte('0' + n/10%10), byte('0' + n%10)})
}
