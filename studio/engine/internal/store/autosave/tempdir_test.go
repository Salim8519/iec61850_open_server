package autosave

import (
	"testing"

	"studio-engine/internal/testutil"
)

// tempDir keeps the package's tests on the shared patient cleanup (Windows delete-pending).
func tempDir(t *testing.T) string { return testutil.TempDir(t) }
