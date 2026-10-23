package project

import (
	"sort"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// preferenceKeys are the stored whole-number preferences in DEFAULTS order.
var preferenceKeys = []string{"tour_seen_version", "sld_goose_layer", "sld_readers_layer"}

// goOnlyPreferenceKeys are stored only once set (the frozen Python engine has none, so its
// default document stays byte-identical): sld_sbo_steps 1 shows select before operate as two
// steps on the SLD and point panels instead of one click (change 0173); display_size 0 fits
// the app to the window (default) and 1 always shows it at 100%.
var goOnlyPreferenceKeys = []string{"sld_sbo_steps", "display_size"}

// TourVersion is the guided tour version the frontend compares with tour_seen_version.
const TourVersion = 1

// Preferences is PreferenceStore.get for the decoded preferences.json (nil when the file
// is missing or unreadable): the defaults overridden by valid stored values.
func Preferences(stored any) *pyjson.Object {
	table, isDict := stored.(*pyjson.Object)
	result := pyjson.NewObject()
	for _, key := range preferenceKeys {
		result.Set(key, int64(0))
		if isDict {
			if value := pyval.Lookup(table, key, nil); pyval.IsInt(value) {
				result.Set(key, value)
			}
		}
	}
	var filters any
	if isDict {
		filters = pyval.Lookup(table, "list_filters", nil)
		for _, key := range goOnlyPreferenceKeys {
			if value := pyval.Lookup(table, key, nil); pyval.IsInt(value) {
				result.Set(key, value)
			}
		}
	}
	return result.Set("list_filters", CleanListFilters(filters)).Set("tour_version", int64(TourVersion))
}

// UpdatePreferences is PreferenceStore.update without the file write: it checks changes
// and returns the document to store (PreferencesFile); the answer is Preferences of it.
func UpdatePreferences(stored, changes any) (*pyjson.Object, error) {
	table, isDict := changes.(*pyjson.Object)
	if !isDict {
		return nil, apperr.ValueError("Preferences must be a JSON object")
	}
	keys := table.Keys()
	sort.Strings(keys)
	unknown := []string{}
	for _, key := range keys {
		if key != "list_filters" && !pyval.InTuple(key, preferenceKeys...) && !pyval.InTuple(key, goOnlyPreferenceKeys...) {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		return nil, apperr.ValueError("Unknown preference: " + strings.Join(unknown, ", "))
	}
	for _, key := range table.Keys() {
		if key == "list_filters" {
			continue
		}
		value, _ := table.Get(key)
		if !pyval.IsInt(value) {
			return nil, apperr.ValueError("Preference " + key + " must be a whole number")
		}
		high := 1000.0
		if strings.HasPrefix(key, "sld_") || key == "display_size" {
			high = 1
		}
		if !pyval.Between(value, 0, high) {
			return nil, apperr.ValueError("Preference " + key + " is out of range")
		}
	}
	current := Preferences(stored)
	current.Delete("tour_version")
	if filters, found := table.Get("list_filters"); found {
		merged, err := MergeListFilters(object(pyval.Lookup(current, "list_filters", nil)), filters)
		if err != nil {
			return nil, err
		}
		current.Set("list_filters", merged)
	}
	for _, key := range table.Keys() {
		if key != "list_filters" {
			value, _ := table.Get(key)
			current.Set(key, value)
		}
	}
	return current, nil
}

// PreferencesFile is json.dumps(document, indent=2), the preferences.json content.
func PreferencesFile(document *pyjson.Object) ([]byte, error) {
	return pyval.DumpsIndent(document, 2)
}
