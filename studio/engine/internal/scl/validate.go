package scl

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/xmldoc"
)

// SchemaVersion is schema_validation.schema_version: the bundled XSD for the root's
// version/revision/release, or "" when none matches.
func SchemaVersion(root *xmldoc.Element) string {
	version, hasVersion := root.Get("version")
	revision, _ := root.Get("revision")
	release, hasRelease := root.Get("release")
	switch {
	case !hasVersion || version == "2003":
		return "SCL2003"
	case version == "2007" && revision == "B":
		if !hasRelease || release == "" {
			return "SCL2007B"
		}
		if release == "4" {
			return "SCL2007B4"
		}
	}
	return ""
}

// ValidateSchema is schema_validation.validate_schema(source, root) for a document
// ParseDocument accepted: {"status", "schema", "errors": [{"line", "message"}]}.
func ValidateSchema(source []byte, doc *xmldoc.Document, validator SchemaValidator) (*pyjson.Object, error) {
	version := SchemaVersion(doc.Root)
	if version == "" {
		return pyjson.NewObject().Set("status", "unsupported").Set("schema", nil).Set("errors", []any{
			pyjson.NewObject().Set("line", int64(1)).Set("message", "No bundled XSD matches this SCL version/revision/release"),
		}), nil
	}
	if doc.Doctype {
		// lxml parses first; a document expat accepted parses in libxml2 too, so the DTD
		// refusal is what Python reaches (a plain ValueError, HTTP 400).
		return nil, apperr.ValueError("DTD declarations are not allowed in SCL")
	}
	if validator == nil {
		return nil, apperr.RuntimeError("SCL schema validation is not available: the studio_xsd helper is missing. Reinstall ScadaBench.")
	}
	verdict, err := validator.Validate(source, version)
	if err != nil {
		return nil, err
	}
	errors := []any{}
	for _, item := range verdict.Errors {
		errors = append(errors, pyjson.NewObject().Set("line", int64(item.Line)).Set("message", item.Message))
	}
	return pyjson.NewObject().Set("status", verdict.Status).Set("schema", version).Set("errors", errors), nil
}

// Validate is the /scl/validate route body: parse_document then validate_schema.
func Validate(source []byte, validator SchemaValidator) (*pyjson.Object, error) {
	doc, err := ParseDocument(source, false)
	if err != nil {
		return nil, err
	}
	return ValidateSchema(source, doc, validator)
}
