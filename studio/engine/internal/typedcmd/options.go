package typedcmd

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// ParseOptions reads a request's options object {"test", "interlock_check", "synchro_check", "orcat", "orident"}.
func ParseOptions(raw any) (Options, error) {
	options := Options{}
	if raw == nil {
		return options, nil
	}
	object, ok := raw.(*pyjson.Object)
	if !ok {
		return options, apperr.ValueError("Command options must be an object")
	}
	flags := map[string]*bool{"test": &options.Test, "interlock_check": &options.InterlockCheck,
		"synchro_check": &options.SynchroCheck}
	for key, target := range flags {
		if value, present := object.Get(key); present {
			flag, isBool := value.(bool)
			if !isBool {
				return options, apperr.ValueError("Command option " + key + " is true or false")
			}
			*target = flag
		}
	}
	if value, present := object.Get("orcat"); present {
		number, isInt := value.(int64)
		if !isInt {
			return options, apperr.ValueError("Originator category must be a whole number from 0 to 8")
		}
		options.OrCat = int(number)
	}
	if value, present := object.Get("orident"); present {
		name, isText := value.(string)
		if !isText {
			return options, apperr.ValueError("Originator name must be text")
		}
		options.OrIdent = name
	}
	return options, options.Check()
}
