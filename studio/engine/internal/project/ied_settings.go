package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/netset"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// endpoint is a (host, port) tuple key compared with Python ==.
type endpoint struct{ host, port any }

// EndpointConflicts is ied_settings.endpoint_conflicts: plain-language problems that would
// make two IEDs share one address.
func EndpointConflicts(ieds []*pyjson.Object, network *pyjson.Object) ([]any, error) {
	problems := []any{}
	var used []endpoint
	var names []any
	for _, ied := range ieds {
		var point *endpoint
		switch role := pyval.Lookup(ied, "role", "simulated"); role {
		case "simulated":
			host, port := netset.EffectiveEndpoint(ied, network)
			point = &endpoint{host, port}
		case "real":
			host, port, _ := netset.RealEndpoint(ied)
			if pyval.Truthy(host) {
				point = &endpoint{host, port}
			}
		}
		if point == nil {
			continue
		}
		for _, part := range []any{point.host, point.port} {
			if err := pyval.CheckHashable(part); err != nil {
				return nil, err
			}
		}
		found := -1
		for index, other := range used {
			if pyval.Equal(other.host, point.host) && pyval.Equal(other.port, point.port) {
				found = index
			}
		}
		name := pyval.Lookup(ied, "name", nil)
		if found >= 0 && pyval.Truthy(names[found]) {
			problems = append(problems, pyval.Str(name)+" and "+pyval.Str(names[found])+" both use "+
				pyval.Str(point.host)+":"+pyval.Str(point.port)+". Give each IED its own address.")
		}
		if found >= 0 {
			names[found] = name
		} else {
			used, names = append(used, *point), append(names, name)
		}
	}
	return problems, nil
}

func conflictError(problems []any) error {
	failure := apperr.New(problems[0].(string))
	failure.Details = problems
	return failure
}

// asValueError reports whether Python's `except ValueError` catches err (StudioError and
// ValueError do; TypeError and AttributeError do not) and returns its message.
func asValueError(err error) (string, bool) {
	failure, ok := err.(*apperr.Error)
	if ok && (failure.Kind == apperr.Value || failure.Kind == apperr.Studio) {
		return failure.Message, true
	}
	return "", false
}

// mergedNetwork is ied_settings._merged_network: a frequency-only change keeps the mode; a
// mode change keeps the frequency and where GOOSE goes.
func mergedNetwork(current *pyjson.Object, value any) (*pyjson.Object, error) {
	keys, err := pyval.Iter(value)
	if err != nil {
		return nil, err
	}
	frequencyOnly := true
	for _, key := range keys {
		if err := pyval.CheckHashable(key); err != nil {
			return nil, err
		}
		frequencyOnly = frequencyOnly && key == "nominal_hz"
	}
	if frequencyOnly {
		return pyval.Merge(current, value)
	}
	kept := pyjson.NewObject()
	for _, key := range append([]string{"nominal_hz"}, netset.ReachKeys...) {
		if stored, found := current.Get(key); found {
			kept.Set(key, stored)
		}
	}
	return pyval.Merge(kept, value)
}

// UpdateNetwork is ied_settings.update_network without the save: the checked network,
// stored into project["network"] (the caller saves and announces the change).
func UpdateNetwork(project *pyjson.Object, value any) (*pyjson.Object, error) {
	current := object(pyval.Lookup(project, "network", nil))
	merged, err := mergedNetwork(current, pyval.Or(value, pyjson.NewObject()))
	if err != nil {
		return nil, err
	}
	candidate, err := netset.ValidateNetwork(merged)
	if err != nil {
		if message, ok := asValueError(err); ok {
			return nil, apperr.New(message)
		}
		return nil, err
	}
	problems, err := EndpointConflicts(projectIEDs(project), candidate)
	if err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		return nil, conflictError(problems)
	}
	project.Set("network", candidate)
	return candidate, nil
}

// SaveReach is projects/goose_reach.save_reach without the save: where GOOSE goes.
func SaveReach(project *pyjson.Object, reach, uplinkGUID, shareGUID any) (*pyjson.Object, error) {
	network, err := pyval.Merge(pyval.Lookup(project, "network", nil), pyjson.NewObject().
		Set("goose_reach", reach).Set("goose_uplink_guid", pyval.Or(uplinkGUID, "")).
		Set("goose_share_guid", pyval.Or(shareGUID, "")))
	if err != nil {
		return nil, err
	}
	candidate, err := netset.ValidateNetwork(network)
	if err != nil {
		if message, ok := asValueError(err); ok {
			return nil, apperr.New(message)
		}
		return nil, err
	}
	project.Set("network", candidate)
	return candidate, nil
}

// projectIEDs returns project["ieds"] as objects.
func projectIEDs(project *pyjson.Object) []*pyjson.Object {
	result := []*pyjson.Object{}
	for _, item := range items(pyval.Lookup(project, "ieds", nil)) {
		result = append(result, object(item))
	}
	return result
}
