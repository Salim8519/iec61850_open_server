package project

import (
	"testing"

	"studio-engine/internal/pyjson"
)

// sld_sbo_steps (change 0173) is Go-only: absent from the defaults (Python parity), stored and
// answered once set, and checked like the other SLD switches (0 or 1).
func TestSboStepsPreference(t *testing.T) {
	if _, present := Preferences(nil).Get("sld_sbo_steps"); present {
		t.Fatal("the default document must not grow a Go-only key")
	}
	stored, err := UpdatePreferences(nil, pyjson.NewObject().Set("sld_sbo_steps", int64(1)))
	if err != nil {
		t.Fatal(err)
	}
	if value, _ := Preferences(stored).Get("sld_sbo_steps"); value != int64(1) {
		t.Fatalf("stored switch answered as %v", value)
	}
	if _, err := UpdatePreferences(stored, pyjson.NewObject().Set("sld_sbo_steps", int64(2))); err == nil {
		t.Fatal("only 0 or 1")
	}
	if _, err := UpdatePreferences(nil, pyjson.NewObject().Set("sld_nope", int64(1))); err == nil {
		t.Fatal("other unknown keys are still refused")
	}
}
