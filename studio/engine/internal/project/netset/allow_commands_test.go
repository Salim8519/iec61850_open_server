package netset

import (
	"testing"

	"studio-engine/internal/pyjson"
)

// Allow commands is a real IED's own switch (ADR-0138): off by default, only on/off, and
// stored only while on, so settings without it keep their exact bytes.
func TestAllowCommandsIsOffUntilTurnedOn(t *testing.T) {
	plain, err := ValidateLive(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := plain.Get("allow_commands"); found {
		t.Fatal("the default settings do not carry the switch")
	}
	ied := pyjson.NewObject().Set("role", "real").Set("live", plain)
	if AllowsCommands(ied) || AllowsCommands(pyjson.NewObject().Set("role", "real")) {
		t.Fatal("a real IED without the switch takes no commands")
	}
	on, err := ValidateLive(pyjson.NewObject().Set("allow_commands", true))
	if err != nil {
		t.Fatal(err)
	}
	if !AllowsCommands(pyjson.NewObject().Set("live", on)) {
		t.Fatal("the switch turned on is kept")
	}
	off, err := ValidateLive(pyjson.NewObject().Set("allow_commands", false))
	if err != nil {
		t.Fatal(err)
	}
	if _, found := off.Get("allow_commands"); found {
		t.Fatal("the switch turned off is not stored")
	}
	for _, wrong := range []any{"yes", int64(1), nil} {
		if _, err := ValidateLive(pyjson.NewObject().Set("allow_commands", wrong)); err == nil ||
			err.Error() != "Allow commands must be on or off" {
			t.Fatalf("%v is refused with a plain reason, got %v", wrong, err)
		}
	}
}
