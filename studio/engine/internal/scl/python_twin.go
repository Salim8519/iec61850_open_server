package scl

// pythonTwin switches the Go-only import rules of ADR-0114 off, so the golden tests still
// prove that every other byte of preview, validate, import and export matches what the frozen
// Python engine recorded. Only tests set it; the application always runs with it false.
var pythonTwin bool

// adr0114 reports whether the ADR-0114 import rules apply (always, outside the golden tests).
func adr0114() bool { return !pythonTwin }

// AsPythonTwin turns the Go-only import rules off for a golden test of another package and
// returns the function that turns them back on. The application never calls it.
func AsPythonTwin() (restore func()) {
	pythonTwin = true
	return func() { pythonTwin = false }
}
