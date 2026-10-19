// Package scl reads, builds and exports SCL exactly as studio_app/scl does: the same
// ordered model trees, cfg text, warnings and error texts.
package scl

import (
	"regexp"

	"studio-engine/internal/apperr"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

// MaxXML is scl/document.py MAX_XML.
const MaxXML = 20 * 1024 * 1024

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

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
		return nil, apperr.New("SCL exceeds the 20 MB import limit")
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
