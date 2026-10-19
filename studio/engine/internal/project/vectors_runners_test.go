package project

import (
	"studio-engine/internal/project/gateway"
	"studio-engine/internal/project/netset"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// runner is the Go side of one vectors_run.py entry.
type runner func(args *pyjson.Object, ieds []*pyjson.Object, fx *vectorFixtures) (any, error)

func arg(args *pyjson.Object, key string) any {
	value, _ := args.Get(key)
	return value
}

func idSet(value any) map[string]bool {
	ids := map[string]bool{}
	for _, item := range value.([]any) {
		ids[item.(string)] = true
	}
	return ids
}

func objects(value any) []*pyjson.Object {
	result := []*pyjson.Object{}
	for _, item := range value.([]any) {
		result = append(result, item.(*pyjson.Object))
	}
	return result
}

func pair(first string, a any, second string, b any) *pyjson.Object {
	return pyjson.NewObject().Set(first, a).Set(second, b)
}

func withProject(call func(project, args *pyjson.Object) (any, error)) runner {
	return func(args *pyjson.Object, _ []*pyjson.Object, _ *vectorFixtures) (any, error) {
		project := object(arg(args, "project"))
		result, err := call(project, args)
		if err != nil {
			return nil, err
		}
		return pair("result", result, "project", project), nil
	}
}

func pure(call func(args *pyjson.Object, ieds []*pyjson.Object) (any, error)) runner {
	return func(args *pyjson.Object, ieds []*pyjson.Object, _ *vectorFixtures) (any, error) {
		return call(args, ieds)
	}
}

func asAny[T any](value T, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	return value, nil
}

var runners = map[string]runner{
	"compact": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return Compact(object(arg(a, "project"))), nil
	}),
	"expand": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(Expand(object(arg(a, "document"))))
	}),
	"validate_diagram": pure(func(a *pyjson.Object, ieds []*pyjson.Object) (any, error) {
		return asAny(ValidateDiagram(arg(a, "diagram"), ieds))
	}),
	"validate_sources": pure(func(a *pyjson.Object, ieds []*pyjson.Object) (any, error) {
		return asAny(ValidateSources(arg(a, "sources"), ieds))
	}),
	"validate_scenarios": pure(func(a *pyjson.Object, ieds []*pyjson.Object) (any, error) {
		return asAny(ValidateScenarios(arg(a, "scenarios"), ieds))
	}),
	"validate_panel": pure(func(a *pyjson.Object, ieds []*pyjson.Object) (any, error) {
		return asAny(ValidatePanel(arg(a, "entries"), ieds))
	}),
	"migrate_diagram": pure(func(a *pyjson.Object, ieds []*pyjson.Object) (any, error) {
		result, notices, err := MigrateDiagram(arg(a, "diagram"), ieds, arg(a, "version").(int64))
		return []any{result, notices}, err
	}),
	"impacts": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return Impacts(object(arg(a, "project")), idSet(arg(a, "ids"))), nil
	}),
	"without_ieds": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return WithoutIEDs(object(arg(a, "project")), idSet(arg(a, "ids"))), nil
	}),
	"removal_preview": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(RemovalPreview(object(arg(a, "project")), arg(a, "key").(string)))
	}),
	"bulk_preview": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(BulkPreview(object(arg(a, "project")), arg(a, "keys")))
	}),
	"bulk_remove": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		current, candidate, err := BulkRemove(object(arg(a, "project")), arg(a, "keys"), arg(a, "fingerprint"))
		return pair("result", current, "project", candidate), err
	}),
	"validate_network": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(netset.ValidateNetwork(arg(a, "value")))
	}),
	"validate_ied_settings": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(netset.ValidateIEDSettings(object(arg(a, "value"))))
	}),
	"validate_live": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(netset.ValidateLive(arg(a, "value")))
	}),
	"validate_test_mode": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(netset.ValidateTestMode(arg(a, "value")))
	}),
	"update_network": withProject(func(p, a *pyjson.Object) (any, error) { return asAny(UpdateNetwork(p, arg(a, "value"))) }),
	"update_ied": withProject(func(p, a *pyjson.Object) (any, error) {
		return asAny(UpdateIED(p, arg(a, "key").(string), arg(a, "changes")))
	}),
	"save_reach": withProject(func(p, a *pyjson.Object) (any, error) {
		return asAny(SaveReach(p, arg(a, "reach"), arg(a, "uplink"), arg(a, "share")))
	}),
	"endpoint_conflicts": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(EndpointConflicts(objects(arg(a, "ieds")), object(arg(a, "network"))))
	}),
	"normalize_gateway": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(gateway.NormalizeGateway(arg(a, "raw")))
	}),
	"normalize_master": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(gateway.NormalizeMaster(arg(a, "raw")))
	}),
	"empty_gateway": pure(func(*pyjson.Object, []*pyjson.Object) (any, error) { return gateway.EmptyGateway(), nil }),
	"preferences_get": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return Preferences(storedPreferences(arg(a, "stored_text"))), nil
	}),
	"preferences_update": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		current, err := UpdatePreferences(storedPreferences(arg(a, "stored_text")), arg(a, "changes"))
		if err != nil {
			return nil, err
		}
		file, err := PreferencesFile(current)
		if err != nil {
			return nil, err
		}
		return pair("result", Preferences(storedPreferences(string(file))), "file", string(file)), nil
	}),
	"list_filters_clean": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return CleanListFilters(arg(a, "stored")), nil
	}),
	"list_filters_merge": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return asAny(MergeListFilters(object(arg(a, "current")), arg(a, "changes")))
	}),
	"assign_endpoints": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		ieds := arg(a, "ieds")
		return ieds, AssignEndpoints(objects(ieds))
	}),
	"snapshot": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		revision, _ := pyval.BigInt(arg(a, "revision"))
		return Snapshot(object(arg(a, "project")), object(arg(a, "notice")), arg(a, "session_id").(string),
			revision.Int64()), nil
	}),
	"new_project": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) { return NewProject(arg(a, "id").(string)), nil }),
	"source_summary": pure(func(a *pyjson.Object, _ []*pyjson.Object) (any, error) {
		return SourceSummary(object(arg(a, "spec"))), nil
	}),
	"restore": restoreRunner,
}

// storedPreferences is json.loads of preferences.json, nil when missing or invalid.
func storedPreferences(text any) any {
	source, ok := text.(string)
	if !ok {
		return nil
	}
	value, err := pyjson.Decode([]byte(source))
	if err != nil {
		return nil
	}
	return value
}
