package xsd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/apperr"
)

// bundledSchemas is studio/schemas, seen from this package folder.
var bundledSchemas = filepath.Join("..", "..", "..", "..", "schemas")

type fixture struct {
	validator *Validator
	log       string
}

// newFixture returns a validator whose helper is this test binary in fake mode.
func newFixture(t *testing.T, timeout time.Duration) *fixture {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return newFixtureWithHelper(t, executable, timeout)
}

func newFixtureWithHelper(t *testing.T, helper string, timeout time.Duration) *fixture {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "fake.log")
	t.Setenv("STUDIO_XSD_FAKE", "1")
	t.Setenv("STUDIO_XSD_FAKE_LOG", log)
	validator := New(Config{HelperPath: helper, SchemasDir: bundledSchemas,
		DataDir: filepath.Join(dir, "data"), Timeout: timeout})
	t.Cleanup(validator.Close)
	return &fixture{validator: validator, log: log}
}

// helperCalls counts validate requests the fake helper received.
func (f *fixture) helperCalls(t *testing.T) int {
	t.Helper()
	data, err := os.ReadFile(f.log)
	if errors.Is(err, os.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(data), "\n")
}

// requireRuntime checks an apperr.RuntimeError whose text contains fragment.
func requireRuntime(t *testing.T, err error, fragment string) {
	t.Helper()
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Kind != apperr.Runtime {
		t.Fatalf("want RuntimeError containing %q, got %#v", fragment, err)
	}
	if !strings.Contains(appErr.Message, fragment) {
		t.Fatalf("want text containing %q, got %q", fragment, appErr.Message)
	}
}
