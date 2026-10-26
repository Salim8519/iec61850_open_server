package netset

// GooseLocal is the Go-only network key that keeps GOOSE inside ScadaBench (ADR-0119). It is
// stored only while true, and goose_reach keeps the Python values, so the frozen Python
// engine reads every project unchanged (it drops keys it does not know).
const GooseLocal = "goose_local"

// pythonTwin switches ADR-0119 off, so golden tests still compare every other byte with
// the frozen Python engine. Only tests set it; the application always runs with it false.
var pythonTwin bool

// AsPythonTwin turns ADR-0119 off for a golden test and returns the function that turns
// it back on. The application never calls it.
func AsPythonTwin() (restore func()) {
	pythonTwin = true
	return func() { pythonTwin = false }
}

// PythonTwin reports whether a golden test switched ADR-0119 off.
func PythonTwin() bool { return pythonTwin }
