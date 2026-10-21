package xsd

import (
	"fmt"
	"time"

	"studio-engine/internal/apperr"
)

// The English texts of every failure that stops a verdict. They are RuntimeErrors (HTTP
// 503): Python validated in-process and never failed here, so each one names the broken
// part and what the user can do about it.

func errMissing() error {
	return apperr.RuntimeError("SCL schema validation is not available: the studio_xsd helper is missing. Reinstall ScadaBench.")
}

func errStart(cause error) error {
	return apperr.RuntimeError(fmt.Sprintf("SCL schema validation is not available: the studio_xsd helper could not start (%v). Reinstall ScadaBench if this repeats.", cause))
}

func errProtocol(detail string) error {
	return apperr.RuntimeError(fmt.Sprintf("SCL schema validation is not available: the studio_xsd helper answered unexpectedly (%s). Reinstall ScadaBench.", detail))
}

func errLibxml2(found string) error {
	return apperr.RuntimeError(fmt.Sprintf("SCL schema validation is not available: the studio_xsd helper uses libxml2 %q, but ScadaBench needs libxml2 %s for identical results. Reinstall ScadaBench.", found, Libxml2Version))
}

func errBuild(found, expected string) error {
	return apperr.RuntimeError(fmt.Sprintf("SCL schema validation is not available: the studio_xsd helper belongs to another ScadaBench build (%q, expected %q). Reinstall ScadaBench.", found, expected))
}

func errTimeout(limit time.Duration) error {
	return apperr.RuntimeError(fmt.Sprintf("SCL schema validation took longer than %d seconds and was stopped. Try again; if it repeats, restart ScadaBench.", int(limit.Seconds())))
}

func errCrashed(detail string) error {
	return apperr.RuntimeError(fmt.Sprintf("The studio_xsd helper stopped during SCL schema validation (%s). Try again; the helper restarts by itself.", detail))
}

func errHelper(message string) error {
	return apperr.RuntimeError("SCL schema validation failed in the studio_xsd helper: " + message)
}

func errSchemaFiles(schema string, cause error) error {
	return apperr.RuntimeError(fmt.Sprintf("The bundled XSD files for %s could not be read (%v). Reinstall ScadaBench.", schema, cause))
}

func errWorkFile(cause error) error {
	return apperr.RuntimeError(fmt.Sprintf("SCL schema validation could not write its working file in the data folder (%v). Check free disk space and folder permissions.", cause))
}

func errClosed() error {
	return apperr.RuntimeError("SCL schema validation is not available because ScadaBench is closing.")
}

// errDoctype is Python's ValueError for a document lxml parsed with a DOCTYPE.
func errDoctype() error {
	return apperr.ValueError("DTD declarations are not allowed in SCL")
}
