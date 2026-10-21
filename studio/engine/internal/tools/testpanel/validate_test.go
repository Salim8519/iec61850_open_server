package testpanel

import (
	"errors"
	"path/filepath"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pyoracle"
)

// TestValidateAgainstPython replays validate_panel on public template IEDs.
func TestValidateAgainstPython(t *testing.T) {
	cases := pyoracle.Cases(t, filepath.Join("testdata", "panel_oracle.py"), nil, "61850", "1500")
	var ieds []*pyjson.Object
	checked, refused := 0, 0
	for _, item := range cases {
		if item.Fn == "" {
			list, _ := item.Raw.Get("ieds")
			for _, ied := range list.([]any) {
				ieds = append(ieds, ied.(*pyjson.Object))
			}
			continue
		}
		checked++
		got, err := Validate(item.Args[0], ieds)
		if item.Error != "" {
			refused++
			var studio *apperr.Error
			text := "?"
			if errors.As(err, &studio) {
				text = map[apperr.Kind]string{apperr.Studio: "StudioError", apperr.Value: "ValueError"}[studio.Kind] +
					": " + studio.Message
			}
			if text != item.Error {
				t.Errorf("%s: got %v want %s", pyoracle.Dumps(t, item.Args), err, item.Error)
			}
			continue
		}
		if err != nil || pyoracle.Dumps(t, got) != pyoracle.Dumps(t, item.Result) {
			t.Errorf("%s: got %v %v want %s", pyoracle.Dumps(t, item.Args), got, err, pyoracle.Dumps(t, item.Result))
		}
	}
	if checked < 1000 || refused == checked || refused == 0 {
		t.Fatalf("oracle balance: %d cases, %d refused", checked, refused)
	}
}
