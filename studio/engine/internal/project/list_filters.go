package project

import (
	"regexp"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Remembered list filters (change 0093). Python's re.match "$" also accepts a final "\n".
var (
	listIDPattern  = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}\n?$`)
	filterKey      = regexp.MustCompile(`^[a-z][a-zA-Z0-9_]{0,31}\n?$`)
	maxLists       = 40
	maxFilterKeys  = 16
	maxFilterChars = 120
)

// cleanFilterValues is list_filters._clean_values: the first 16 entries, keeping valid keys
// with true/false or short text; nil when values is not a dict.
func cleanFilterValues(values any) *pyjson.Object {
	table, isDict := values.(*pyjson.Object)
	if !isDict {
		return nil
	}
	clean := pyjson.NewObject()
	for index, key := range table.Keys() {
		if index >= maxFilterKeys {
			break
		}
		if !filterKey.MatchString(key) {
			continue
		}
		value, _ := table.Get(key)
		text, isText := value.(string)
		if _, isBool := value.(bool); isBool || (isText && pyval.Len(text) <= maxFilterChars) {
			clean.Set(key, value)
		}
	}
	return clean
}

// CleanListFilters is list_filters.clean_stored: every valid stored list entry.
func CleanListFilters(stored any) *pyjson.Object {
	result := pyjson.NewObject()
	table, isDict := stored.(*pyjson.Object)
	if !isDict {
		return result
	}
	for _, listID := range table.Keys() {
		if !listIDPattern.MatchString(listID) {
			continue
		}
		values, _ := table.Get(listID)
		if clean := cleanFilterValues(values); clean != nil && result.Len() < maxLists {
			result.Set(listID, clean)
		}
	}
	return result
}

// MergeListFilters is list_filters.merge_update: a PUT body part checked strictly and
// merged into the current lists (ValueError texts as in Python).
func MergeListFilters(current *pyjson.Object, changes any) (*pyjson.Object, error) {
	table, isDict := changes.(*pyjson.Object)
	if !isDict {
		return nil, apperr.ValueError("list_filters must be an object of lists")
	}
	merged := pyval.CopyShallow(current)
	for _, listID := range table.Keys() {
		if !listIDPattern.MatchString(listID) {
			return nil, apperr.ValueError("Unknown list name: " + pyval.Repr(listID))
		}
		values, _ := table.Get(listID)
		clean := cleanFilterValues(values)
		if clean == nil || clean.Len() != values.(*pyjson.Object).Len() {
			return nil, apperr.ValueError("Filters of " + listID + " must be short text or true/false values")
		}
		merged.Set(listID, clean)
	}
	if merged.Len() > maxLists {
		return nil, apperr.ValueError("Too many remembered lists")
	}
	return merged, nil
}
