package project

import (
	"studio-engine/internal/project/gateway"
	"studio-engine/internal/project/netset"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// NewProject is the project of a fresh ProjectService; id is uuid4().hex.
func NewProject(id string) *pyjson.Object {
	return pyjson.NewObject().Set("id", id).Set("name", "Untitled substation").Set("ieds", []any{}).
		Set("diagram", pyjson.NewObject().Set("nodes", []any{}).Set("edges", []any{})).
		Set("sources", []any{}).Set("scenarios", []any{}).Set("test_panel", []any{}).
		Set("network", netset.DefaultNetwork()).Set("gateway", gateway.EmptyGateway())
}

// Snapshot is ProjectService.snapshot: the public project without SCL sources, runtime
// models and the gateway point list, with signal counts, the migration notice (nil when
// none), the session id and the revision.
func Snapshot(project, notice *pyjson.Object, sessionID string, revision int64) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range project.Keys() {
		if key == "ieds" || key == "sources" || key == "gateway" {
			continue
		}
		value, _ := project.Get(key)
		result.Set(key, pyval.DeepCopy(value))
	}
	sources := []any{}
	for _, item := range items(pyval.Lookup(project, "sources", nil)) {
		entry := pyval.CopyShallow(object(item))
		entry.Set("spec", SourceSummary(object(pyval.Lookup(entry, "spec", nil))))
		sources = append(sources, pyval.DeepCopy(entry))
	}
	result.Set("sources", sources)
	ieds := []any{}
	for _, item := range items(pyval.Lookup(project, "ieds", nil)) {
		ied := object(item)
		summary := pyjson.NewObject()
		for _, key := range ied.Keys() {
			if key != "source" && key != "cfg" && key != "signals" {
				value, _ := ied.Get(key)
				summary.Set(key, pyval.DeepCopy(value))
			}
		}
		summary.Set("signal_count", int64(len(items(pyval.Lookup(ied, "signals", nil)))))
		ieds = append(ieds, summary)
	}
	result.Set("ieds", ieds)
	if pyval.Truthy(notice) {
		result.Set("migration_notice", pyval.CopyObject(notice))
	}
	return result.Set("session_id", sessionID).Set("revision", revision)
}
