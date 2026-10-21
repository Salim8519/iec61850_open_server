package project

import (
	"regexp"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

var storedID = regexp.MustCompile(`^[0-9a-f]{32}$`)

// StoredGroup is the stored IEDs that share one SCL source and filename; the service
// imports each group once (stored_import.import_stored).
type StoredGroup struct {
	Source   any
	Filename any
	Members  []*pyjson.Object
}

// StoredIEDs returns the archive format version and raw["ieds"] after restore()'s first
// check (a list of at most 20 IEDs).
func StoredIEDs(raw *pyjson.Object) (int64, []any, error) {
	format, err := pyval.Int(pyval.Lookup(raw, "format", int64(1)))
	if err != nil {
		return 0, nil, err
	}
	version, _ := pyval.BigInt(format)
	list, isList := pyval.Lookup(raw, "ieds", nil).([]any)
	if !isList || len(list) > 20 {
		return 0, nil, apperr.New("Project must contain a list of at most 20 IEDs")
	}
	return version.Int64(), list, nil
}

// StoredGroups is the checking and grouping part of import_stored, in Python's order.
func StoredGroups(items []any) ([]*StoredGroup, error) {
	var groups []*StoredGroup
	for _, entry := range items {
		item, isDict := entry.(*pyjson.Object)
		if !isDict {
			return nil, apperr.New("Invalid stored IED")
		}
		id := pyval.Lookup(item, "id", "")
		text, isText := id.(string)
		if !isText {
			return nil, pyval.TypeError("expected string or bytes-like object, got '" + pyval.TypeName(id) + "'")
		}
		if !storedID.MatchString(text) {
			return nil, apperr.New("Invalid stored IED identifier")
		}
		source, found := item.Get("source")
		if !found {
			return nil, apperr.KeyError("source")
		}
		filename := pyval.Lookup(item, "filename", "stored.scd")
		for _, part := range []any{source, filename} {
			if err := pyval.CheckHashable(part); err != nil {
				return nil, err
			}
		}
		var group *StoredGroup
		for _, existing := range groups {
			if pyval.Equal(existing.Source, source) && pyval.Equal(existing.Filename, filename) {
				group = existing
			}
		}
		if group == nil {
			group = &StoredGroup{Source: source, Filename: filename}
			groups = append(groups, group)
		}
		group.Members = append(group.Members, item)
	}
	ids := map[string]bool{}
	for _, entry := range items {
		ids[text(object(entry), "id")] = true
	}
	if len(ids) != len(items) {
		return nil, apperr.New("Project contains too many or duplicate IEDs")
	}
	return groups, nil
}

// Names returns the stored IED names of the group (the import selection) after checking
// they are unique; call it right before importing the group.
func (g *StoredGroup) Names() ([]any, error) {
	names := []any{}
	for _, member := range g.Members {
		name, found := member.Get("name")
		if !found {
			return nil, apperr.KeyError("name")
		}
		names = append(names, name)
	}
	for _, name := range names {
		if err := pyval.CheckHashable(name); err != nil {
			return nil, err
		}
	}
	for index, name := range names {
		for _, earlier := range names[:index] {
			if pyval.Equal(earlier, name) {
				return nil, apperr.New("Project contains too many or duplicate IEDs")
			}
		}
	}
	return names, nil
}

// Assign maps the imported models of this group to the stored IED ids: each member takes
// the model with its name (a missing one is an error) and its own validation copy.
func (g *StoredGroup) Assign(imported []*pyjson.Object, into map[string]*pyjson.Object) error {
	built := map[string]*pyjson.Object{}
	for _, model := range imported {
		built[text(model, "name")] = model
	}
	for _, member := range g.Members {
		name, _ := member.Get("name")
		label, _ := name.(string)
		model := built[label]
		if _, isText := name.(string); !isText || model == nil {
			return apperr.New("Stored IED is missing from its source XML")
		}
		model.Set("validation", pyval.DeepCopy(pyval.Lookup(model, "validation", nil)))
		into[text(member, "id")] = model
	}
	return nil
}
