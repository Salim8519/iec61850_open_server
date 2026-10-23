// Package scl reads, builds and exports SCL exactly as studio_app/scl does: the same
// ordered model trees, cfg text, warnings and error texts.
package scl

import (
	"regexp"

	"studio-engine/internal/apperr"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

// MaxXML is the import size limit: 100 MB (ADR-0114; Python's MAX_XML is 20 MB).
const MaxXML = 100 * 1024 * 1024

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ldInstPattern is the SCL XSD tLDInst: a digit may come first ("1", "252L2"), at most 64.
var ldInstPattern = regexp.MustCompile(`^[A-Za-z0-9][0-9A-Za-z_]{0,63}$`)

// LDInst is Identifier for LDevice@inst and FCDA@ldInst widened to the XSD tLDInst rule
// (ADR-0114): everything Identifier accepted stays accepted.
func LDInst(value string, present bool, context string) (string, error) {
	if !adr0114() {
		return Identifier(value, present, context)
	}
	if !present || !ldInstPattern.MatchString(value) && !identifierPattern.MatchString(value) {
		return "", apperr.New("Invalid " + context + ": " + pyrules.ReprOrNone(value, present))
	}
	return value, nil
}

// Identifier is type_resolver.identifier: the value or StudioError "Invalid <context>: <repr>".
func Identifier(value string, present bool, context string) (string, error) {
	if !present || value == "" || !identifierPattern.MatchString(value) {
		return "", apperr.New("Invalid " + context + ": " + pyrules.ReprOrNone(value, present))
	}
	return value, nil
}

// ParseDocument is document.parse_document: size limit, safe parse, SCL root, valid and
// unique IED names. keepNodes keeps comments and mixed content for the exporter.
func ParseDocument(source []byte, keepNodes bool) (*xmldoc.Document, error) {
	if len(source) > MaxXML {
		return nil, apperr.New("SCL exceeds the 100 MB import limit")
	}
	doc, err := xmldoc.Parse(source, xmldoc.Options{Nodes: keepNodes})
	if err != nil {
		return nil, apperr.New("Invalid or unsafe XML: " + err.Error())
	}
	if doc.Root.Tag != "SCL" {
		return nil, apperr.New("Expected an SCL root element")
	}
	var names []string
	for _, ied := range doc.Root.FindAll("IED") {
		value, present := ied.Get("name")
		name, err := Identifier(value, present, "IED name")
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			return nil, apperr.New("Duplicate IED name in SCL")
		}
		seen[name] = true
	}
	return doc, nil
}
