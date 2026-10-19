package launcher

import (
	"fmt"
	"os"
	"strings"
)

// Engine names (ADR-0074): Python stays the default until the final gate.
const (
	EnginePython  = "python"
	EngineGo      = "go"
	DefaultEngine = EnginePython
	// EngineVariable selects the engine when no --engine flag is given.
	EngineVariable = "SCADABENCH_ENGINE"
)

// ResolveEngine picks the engine: the --engine flag, else SCADABENCH_ENGINE, else Python.
// An unknown name is an error, never a silent fallback.
func ResolveEngine(flag string) (string, error) {
	choice, source := flag, "--engine"
	if choice == "" {
		choice, source = os.Getenv(EngineVariable), EngineVariable
	}
	switch strings.ToLower(strings.TrimSpace(choice)) {
	case "":
		return DefaultEngine, nil
	case EnginePython:
		return EnginePython, nil
	case EngineGo:
		return EngineGo, nil
	}
	return "", fmt.Errorf("%s must be \"python\" or \"go\", not %q", source, choice)
}
