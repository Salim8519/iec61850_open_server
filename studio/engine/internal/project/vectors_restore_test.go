package project

import (
	"crypto/sha256"
	"encoding/hex"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// lightKeys are the IED fields restore vectors keep; others are compared by digest.
var lightKeys = map[string]bool{"id": true, "name": true, "filename": true, "ip": true, "port": true,
	"state": true, "edition": true, "role": true, "live": true, "lab_ip": true, "goose_publish": true,
	"test_mode": true, "scl_address": true}

// emulateImport stands in for import_scl(source, filename, names): the fixture models with
// those names whose SCL text is the group's source.
func emulateImport(fx *vectorFixtures, group *StoredGroup, names []any) []*pyjson.Object {
	models := []*pyjson.Object{}
	for _, name := range names {
		text, _ := name.(string)
		model, found := fx.models[text]
		if found && group.Source == fx.sources[text] {
			copied := pyval.CopyObject(model)
			copied.Set("filename", group.Filename)
			models = append(models, copied)
		}
	}
	return models
}

func restoreRunner(args *pyjson.Object, _ []*pyjson.Object, fx *vectorFixtures) (any, error) {
	raw, err := Expand(object(arg(args, "document")))
	if err != nil {
		return nil, err
	}
	version, items, err := StoredIEDs(raw)
	if err != nil {
		return nil, err
	}
	groups, err := StoredGroups(items)
	if err != nil {
		return nil, err
	}
	models := map[string]*pyjson.Object{}
	for _, group := range groups {
		names, err := group.Names()
		if err != nil {
			return nil, err
		}
		if err := group.Assign(emulateImport(fx, group, names), models); err != nil {
			return nil, err
		}
	}
	newID := arg(args, "new_id").(string)
	restored, err := PrepareRestore(RestoreInput{Raw: raw, Version: version, Models: models,
		Preserve: func() (string, error) { return "@backup", nil }, NewID: func() string { return newID }})
	if err != nil {
		return nil, err
	}
	var notice any
	if restored.Notice != nil {
		notice = restored.Notice
	}
	return pyjson.NewObject().Set("project", reduceProject(restored.Project)).Set("notice", notice), nil
}

func reduceProject(project *pyjson.Object) *pyjson.Object {
	result := pyval.CopyShallow(project)
	ieds := []any{}
	for _, item := range items(pyval.Lookup(project, "ieds", nil)) {
		ied := object(item)
		reduced := pyjson.NewObject()
		for _, key := range ied.Keys() {
			value, _ := ied.Get(key)
			if !lightKeys[key] {
				data, _ := pyjson.Dumps(value)
				digest := sha256.Sum256(data)
				value = "sha256:" + hex.EncodeToString(digest[:])
			}
			reduced.Set(key, value)
		}
		ieds = append(ieds, reduced)
	}
	return result.Set("ieds", ieds)
}
