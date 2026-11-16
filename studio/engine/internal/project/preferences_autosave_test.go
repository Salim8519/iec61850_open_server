package project

import (
	"testing"

	"studio-engine/internal/pyjson"
)

// autosave_off is Go-only (ADR-0139): absent from the defaults (Python parity), auto save is on
// until it is stored as 1, and only 0 or 1 is taken.
func TestAutoSaveOffPreference(t *testing.T) {
	if _, present := Preferences(nil).Get("autosave_off"); present || AutoSaveOff(nil) {
		t.Fatal("auto save is off by default")
	}
	stored, err := UpdatePreferences(nil, pyjson.NewObject().Set("autosave_off", int64(1)))
	if err != nil || !AutoSaveOff(stored) {
		t.Fatalf("stored %v, err %v", stored, err)
	}
	decoded, err := pyjson.Decode([]byte(`{"autosave_off": 1}`))
	if err != nil || !AutoSaveOff(decoded) {
		t.Fatalf("read back from preferences.json: %v %v", decoded, err)
	}
	for _, bad := range []any{int64(2), true, "1"} {
		if _, err := UpdatePreferences(stored, pyjson.NewObject().Set("autosave_off", bad)); err == nil {
			t.Fatalf("autosave_off %v accepted", bad)
		}
	}
}
