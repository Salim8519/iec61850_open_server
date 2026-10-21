package main

import (
	"os"

	"studio-engine/internal/selftest"
)

// selfTest is "--self-test" (launcher/self_test.py): the offline engine check on temporary
// data inside the data folder; it runs before the data-folder lease, as in Python.
func selfTest(paths paths) int {
	return selftest.Run(selftest.Options{DataDir: paths.data, NativeDir: paths.native,
		FrontendDir: paths.frontend, SchemasDir: paths.schemas, XSDSeed: seedFiles(),
		Version: paths.version, Enforce: packaged == "1", NativeSeal: parseSeal(nativeSeal)}, os.Stdout, os.Stderr)
}
