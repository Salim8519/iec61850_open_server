// Package xsd validates SCL against the bundled IEC XSDs through the studio_xsd native
// helper (libxml2 2.11.9, the release lxml 6.0.2 bundles), with a result cache in front:
// the same messages and lines as scl/schema_validation.py without cgo (ADR-0074).
package xsd

import (
	"fmt"
	"path/filepath"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/scl"
)

// Libxml2Version is the libxml2 release whose messages the verdicts reproduce; it is
// part of every cache key and the helper must report it in hello.
const Libxml2Version = "2.11.9"

// Schemas are the bundled XSD folders under Config.SchemasDir.
var Schemas = []string{"SCL2003", "SCL2007B", "SCL2007B4"}

// Config locates the helper, the bundled XSDs and the data folder.
type Config struct {
	HelperPath    string        // studio_xsd.exe
	SchemasDir    string        // folder holding SCL2003, SCL2007B and SCL2007B4
	DataDir       string        // verdicts in <DataDir>/cache/xsd, work files in .../work
	Timeout       time.Duration // per validation; 0 means DefaultTimeout
	ExpectedBuild string        // when set, a helper reporting another build is refused
	MemoryEntries int           // verdicts kept in memory; 0 means DefaultMemoryEntries
}

const (
	DefaultTimeout       = 120 * time.Second
	DefaultMemoryEntries = 256
)

// Validator implements scl.SchemaValidator: cache first, the helper only on a miss.
type Validator struct {
	config  Config
	cache   *cache
	digests *schemaDigests
	client  *client
}

var _ scl.SchemaValidator = (*Validator)(nil)

// New prepares a validator; nothing is started or read until the first request.
func New(config Config) *Validator {
	if config.Timeout <= 0 {
		config.Timeout = DefaultTimeout
	}
	if config.MemoryEntries <= 0 {
		config.MemoryEntries = DefaultMemoryEntries
	}
	cacheDir := filepath.Join(config.DataDir, "cache", "xsd")
	work := &workFiles{dir: filepath.Join(cacheDir, "work")}
	return &Validator{
		config:  config,
		cache:   newCache(cacheDir, config.MemoryEntries),
		digests: newSchemaDigests(config.SchemasDir),
		client:  newClient(config.HelperPath, config.ExpectedBuild, config.Timeout, work),
	}
}

// Validate returns the verdict validate_schema produced for source against schema. A
// DOCTYPE that lxml parsed gives Python's ValueError; failures are RuntimeErrors.
func (v *Validator) Validate(source []byte, schema string) (scl.SchemaVerdict, error) {
	key, err := v.key(sha256Hex(source), schema)
	if err != nil {
		return scl.SchemaVerdict{}, err
	}
	if verdict, found := v.cache.get(key); found {
		return verdict, nil
	}
	result, err := v.client.validate(source, v.schemaPath(schema))
	if err != nil {
		return scl.SchemaVerdict{}, err
	}
	if result.Doctype {
		return scl.SchemaVerdict{}, errDoctype() // an exception in Python, never cached
	}
	_ = v.cache.put(key, result.Verdict) // a full disk only costs a later re-validation
	return copyVerdict(result.Verdict), nil
}

// Close stops the helper. Cached verdicts stay on disk.
func (v *Validator) Close() {
	v.client.close()
}

// HelperStarts counts helper processes started so far (diagnostics and tests).
func (v *Validator) HelperStarts() int {
	return v.client.startCount()
}

func (v *Validator) key(sourceSHA256, schema string) (string, error) {
	if !knownSchema(schema) {
		return "", apperr.ValueError(fmt.Sprintf("No bundled XSD named %q", schema))
	}
	digest, err := v.digests.get(schema)
	if err != nil {
		return "", err
	}
	return cacheKey(sourceSHA256, schema, digest), nil
}

func (v *Validator) schemaPath(schema string) string {
	path, err := filepath.Abs(filepath.Join(v.config.SchemasDir, schema, "SCL.xsd"))
	if err != nil {
		return filepath.Join(v.config.SchemasDir, schema, "SCL.xsd")
	}
	return path
}

func knownSchema(schema string) bool {
	for _, name := range Schemas {
		if name == schema {
			return true
		}
	}
	return false
}
