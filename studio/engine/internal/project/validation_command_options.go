package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// checkCommandOptions keeps the remembered command options (ADR-0108) on the kinds whose sheet
// offers them: a breaker, disconnector or earth switch.
func checkCommandOptions(data *pyjson.Object, kind string) error {
	if _, present := data.Get("command_options"); !present || pyval.Lookup(data, "command_options", nil) == nil {
		return nil
	}
	switch kind {
	case "CB", "DS", "ES":
		return nil
	}
	return apperr.New("Only a breaker, disconnector or earth switch remembers command options")
}
