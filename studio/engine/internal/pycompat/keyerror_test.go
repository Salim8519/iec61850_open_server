package pycompat

import "testing"

func TestKeyErrorUsesPythonRepr(t *testing.T) {
	for key, want := range map[string]string{"ied_id": "'ied_id'", "it's": `"it's"`, `say "hi"`: `'say "hi"'`} {
		if got := KeyError(key).Message; got != want {
			t.Errorf("KeyError(%q) = %s, want %s", key, got, want)
		}
	}
}
