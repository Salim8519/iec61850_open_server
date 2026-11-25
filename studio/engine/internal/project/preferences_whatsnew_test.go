package project

import (
	"errors"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// whats_new_seen is Go-only (ADR-0148): absent from the default document, stored once set, and a
// version code from 0 to 999.999.999.
func TestWhatsNewSeenPreference(t *testing.T) {
	if _, present := Preferences(nil).Get("whats_new_seen"); present {
		t.Fatal("the default document must not grow a Go-only key")
	}
	stored, err := UpdatePreferences(nil, pyjson.NewObject().Set("whats_new_seen", int64(1037)))
	if err != nil {
		t.Fatal(err)
	}
	if value, _ := Preferences(stored).Get("whats_new_seen"); value != int64(1037) {
		t.Fatalf("stored version answered as %v", value)
	}
	if _, err := UpdatePreferences(stored, pyjson.NewObject().Set("whats_new_seen", int64(999999999))); err != nil {
		t.Fatalf("the largest version code must be kept: %v", err)
	}
	for _, bad := range []any{int64(1000000000), int64(-1), 1.5, "0.1.37"} {
		_, err := UpdatePreferences(stored, pyjson.NewObject().Set("whats_new_seen", bad))
		var refusal *apperr.Error
		if !errors.As(err, &refusal) || refusal.Kind != apperr.Value {
			t.Fatalf("%#v must be refused with a ValueError, got %v", bad, err)
		}
	}
}
