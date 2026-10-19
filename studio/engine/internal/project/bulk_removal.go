package project

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// selectedKeys checks the bulk selection: 1 to 20 unique str ids, returned sorted.
func selectedKeys(keys any) ([]string, error) {
	list, isList := keys.([]any)
	fail := apperr.New("Select 1 to 20 unique IEDs")
	if !isList || len(list) < 1 || len(list) > 20 {
		return nil, fail
	}
	sorted := make([]string, 0, len(list))
	seen := map[string]bool{}
	for _, item := range list {
		key, isText := item.(string)
		if !isText {
			return nil, fail
		}
		seen[key] = true
		sorted = append(sorted, key)
	}
	if len(seen) != len(list) {
		return nil, fail
	}
	sort.Strings(sorted)
	return sorted, nil
}

// Fingerprint is the bulk-removal impact fingerprint: sha256 of
// json.dumps({"project": compact(project), "ids": sorted(keys)}, sort_keys=True).
func Fingerprint(project *pyjson.Object, sortedKeys []string) (string, error) {
	ids := make([]any, len(sortedKeys))
	for index, key := range sortedKeys {
		ids[index] = key
	}
	document, err := pyval.DumpsSorted(pyjson.NewObject().Set("project", Compact(project)).Set("ids", ids))
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(document)
	return hex.EncodeToString(digest[:]), nil
}

// BulkPreview is bulk_removal.preview: the selected IEDs, the combined impact and the
// fingerprint the removal must quote back.
func BulkPreview(project *pyjson.Object, keys any) (*pyjson.Object, error) {
	sorted, err := selectedKeys(keys)
	if err != nil {
		return nil, err
	}
	selected := []any{}
	ids := map[string]bool{}
	for _, key := range sorted {
		ied, err := FindIED(project, key)
		if err != nil {
			return nil, err
		}
		selected = append(selected, pyjson.NewObject().Set("id", key).Set("name", pyval.Lookup(ied, "name", nil)))
		ids[key] = true
	}
	fingerprint, err := Fingerprint(project, sorted)
	if err != nil {
		return nil, err
	}
	result := appendAll(pyjson.NewObject().Set("ieds", selected), Impacts(project, ids))
	return result.Set("fingerprint", fingerprint), nil
}

// BulkRemove is the pure part of bulk_removal.remove: the fresh preview (which must carry
// the same fingerprint) and the candidate project. The service then writes the
// "before-ied-delete-<uuid>.subsim" backup, swaps, saves and answers
// {**preview, "recovery_file": <backup path>}.
func BulkRemove(project *pyjson.Object, keys, fingerprint any) (*pyjson.Object, *pyjson.Object, error) {
	current, err := BulkPreview(project, keys)
	if err != nil {
		return nil, nil, err
	}
	expected, _ := current.Get("fingerprint")
	if fingerprint != expected {
		return nil, nil, apperr.New("Project changed. Review a fresh deletion preview")
	}
	ids := map[string]bool{}
	for _, key := range keys.([]any) {
		ids[key.(string)] = true
	}
	return current, WithoutIEDs(project, ids), nil
}
