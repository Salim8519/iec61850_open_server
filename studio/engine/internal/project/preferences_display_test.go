package project

import (
	"errors"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// display_size is Go-only like sld_sbo_steps: absent from the defaults (Python parity), stored
// and answered once set, and only 0 (fit the app to the window) or 1 (always 100%).
func TestDisplaySizePreference(t *testing.T) {
	if _, present := Preferences(nil).Get("display_size"); present {
		t.Fatal("the default document must not grow a Go-only key")
	}
	stored, err := UpdatePreferences(nil, pyjson.NewObject().Set("display_size", int64(1)))
	if err != nil {
		t.Fatal(err)
	}
	if value, _ := Preferences(stored).Get("display_size"); value != int64(1) {
		t.Fatalf("stored size answered as %v", value)
	}
	for _, bad := range []any{int64(2), int64(-1), 0.5, "x"} {
		_, err := UpdatePreferences(stored, pyjson.NewObject().Set("display_size", bad))
		var refusal *apperr.Error
		if !errors.As(err, &refusal) || refusal.Kind != apperr.Value {
			t.Fatalf("%#v must be refused with a ValueError, got %v", bad, err)
		}
	}
}

// The default preferences.json stays byte-identical to the frozen Python engine's.
func TestDisplaySizeKeepsDefaultFile(t *testing.T) {
	document, err := UpdatePreferences(nil, pyjson.NewObject())
	if err != nil {
		t.Fatal(err)
	}
	data, err := PreferencesFile(document)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"tour_seen_version\": 0,\n  \"sld_goose_layer\": 0,\n  \"sld_readers_layer\": 0,\n  \"list_filters\": {}\n}"
	if string(data) != want {
		t.Fatalf("default file changed:\n%s", data)
	}
}
