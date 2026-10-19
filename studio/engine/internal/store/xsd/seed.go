package xsd

import (
	"fmt"
	"regexp"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// SeedEntry is one verdict produced by Python (lxml 6.0.2) for the document whose
// SHA-256 is SHA256, e.g. from scl/testdata/golden/xsd_seed.json or a migration of an
// existing installation.
type SeedEntry struct {
	SHA256  string
	Schema  string
	Verdict scl.SchemaVerdict
}

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// DecodeSeed reads a JSON list of {"sha256", "schema", "status", "errors"} objects.
// Entries whose status is neither "valid" nor "invalid" (Python's "unsupported", which
// never reaches the helper) are skipped; any other malformed entry is an error.
func DecodeSeed(data []byte) ([]SeedEntry, error) {
	value, err := pyjson.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("seed is not JSON: %v", err)
	}
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("seed must be a JSON list")
	}
	entries := make([]SeedEntry, 0, len(list))
	for index, item := range list {
		object, ok := item.(*pyjson.Object)
		if !ok {
			return nil, fmt.Errorf("seed entry %d is not an object", index)
		}
		status, _ := object.Get("status")
		if status != "valid" && status != "invalid" {
			continue
		}
		hash, _ := object.Get("sha256")
		schema, _ := object.Get("schema")
		hashText, okHash := hash.(string)
		schemaText, okSchema := schema.(string)
		if !okHash || !sha256Pattern.MatchString(hashText) || !okSchema || !knownSchema(schemaText) {
			return nil, fmt.Errorf("seed entry %d needs a lowercase sha256 and a bundled schema name", index)
		}
		errors, _ := object.Get("errors")
		verdict, err := verdictFrom(status, errors)
		if err != nil {
			return nil, fmt.Errorf("seed entry %d: %v", index, err)
		}
		entries = append(entries, SeedEntry{SHA256: hashText, Schema: schemaText, Verdict: verdict})
	}
	return entries, nil
}

// ImportSeed stores the verdicts in the cache (memory and disk) under the keys this
// installation computes (current libxml2 version and XSD digests). It returns how many
// entries were stored; it never starts the helper.
func (v *Validator) ImportSeed(entries []SeedEntry) (int, error) {
	stored := 0
	for index, entry := range entries {
		if !sha256Pattern.MatchString(entry.SHA256) {
			return stored, fmt.Errorf("seed entry %d has no lowercase sha256", index)
		}
		if entry.Verdict.Status != "valid" && entry.Verdict.Status != "invalid" {
			return stored, fmt.Errorf("seed entry %d has status %q", index, entry.Verdict.Status)
		}
		key, err := v.key(entry.SHA256, entry.Schema)
		if err != nil {
			return stored, err
		}
		if err := v.cache.put(key, entry.Verdict); err != nil {
			return stored, fmt.Errorf("seed entry %d could not be stored: %v", index, err)
		}
		stored++
	}
	return stored, nil
}
