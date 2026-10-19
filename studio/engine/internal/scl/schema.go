package scl

// SchemaError is one libxml2 message of an XSD validation, as lxml reports it.
type SchemaError struct {
	Line    int
	Message string
}

// SchemaVerdict is what the XSD validator found for one document against one bundled
// schema ("SCL2003", "SCL2007B" or "SCL2007B4"): Status is "valid" or "invalid" and Errors
// holds at most 100 messages (a libxml2 syntax error is one message with its line).
type SchemaVerdict struct {
	Status string
	Errors []SchemaError
}

// SchemaValidator validates SCL bytes against a bundled XSD exactly as
// scl/schema_validation.py does with lxml 6.0.2 / libxml2 2.11.9. Implementations: the
// studio_xsd native helper behind a result cache (internal/store/xsd). An error means the
// verdict could not be produced (helper missing or failed); it is never a verdict.
type SchemaValidator interface {
	Validate(source []byte, schema string) (SchemaVerdict, error)
}
