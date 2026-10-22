package runtime

import (
	"math"
	"sort"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Command options of the equipment sheet (ADR-0108): test, interlock check, synchro check and
// the originator. A request without "options" uses the defaults and keeps today's exact path.
// The project remembers what was last sent per node as data.command_options (contract
// command_options.go), the same five keys with the same limits.

// maxOrIdent is the originator identification limit (Octet64).
const maxOrIdent = 64

// commandOptionsFrom reads the "options" object of a request; nil means none was sent.
func commandOptionsFrom(raw any) (CommandOptions, bool, error) {
	if raw == nil {
		return CommandOptions{}, false, nil
	}
	table, isObject := raw.(*pyjson.Object)
	if !isObject {
		return CommandOptions{}, false, apperr.ValueError("Command options must be a JSON object")
	}
	var unknown []string
	for _, key := range table.Keys() {
		switch key {
		case "test", "interlock_check", "synchro_check", "orcat", "orident":
		default:
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return CommandOptions{}, false, apperr.ValueError("Unknown command option: " + strings.Join(unknown, ", "))
	}
	var options CommandOptions
	flags := map[string]*bool{"test": &options.Test, "interlock_check": &options.InterlockCheck,
		"synchro_check": &options.SynchroCheck}
	for key, target := range flags {
		if value, present := table.Get(key); present {
			flag, isBool := value.(bool)
			if !isBool {
				return CommandOptions{}, false, apperr.ValueError("Command option " + key + " must be true or false")
			}
			*target = flag
		}
	}
	if value, present := table.Get("orcat"); present {
		number, ok := wholeNumber(value)
		if !ok || pyval.IsBool(value) || number < 0 || number > 8 {
			return CommandOptions{}, false, apperr.ValueError("Originator category (orCat) must be a whole number from 0 to 8")
		}
		options.OrCat = int(number)
	}
	if value, present := table.Get("orident"); present {
		text, isText := value.(string)
		if !isText || len(text) > maxOrIdent {
			return CommandOptions{}, false, apperr.ValueError("Originator identification (orIdent) must be text of at most 64 characters")
		}
		options.OrIdent = text
	}
	return options, true, nil
}

// object is the options as the node stores them.
func optionsObject(o CommandOptions) *pyjson.Object {
	return pyval.Obj("test", o.Test, "interlock_check", o.InterlockCheck, "synchro_check", o.SynchroCheck,
		"orcat", int64(o.OrCat), "orident", o.OrIdent)
}

// storedOptions reads data.command_options of a node (the defaults when it has none).
func storedOptions(data *pyjson.Object) CommandOptions {
	stored := pyval.Map(data, "command_options")
	if stored == nil {
		return CommandOptions{}
	}
	orcat, _ := pyval.Float(pyval.Get(stored, "orcat"))
	return CommandOptions{Test: pyval.Get(stored, "test") == true, InterlockCheck: pyval.Get(stored, "interlock_check") == true,
		SynchroCheck: pyval.Get(stored, "synchro_check") == true, OrCat: int(math.Round(orcat)), OrIdent: pyval.Str(stored, "orident")}
}

// optionChanges is the node change that remembers sent options: nil when nothing was sent or
// the node already holds them (a node without options that sends defaults stays without).
func optionChanges(data *pyjson.Object, options CommandOptions, sent bool) *pyjson.Object {
	if !sent || storedOptions(data) == options {
		return nil
	}
	return pyval.Obj("command_options", optionsObject(options))
}
