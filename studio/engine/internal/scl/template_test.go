package scl

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMakeTemplateMatchesPython compares MakeTemplate with make_template's bytes.
func TestMakeTemplateMatchesPython(t *testing.T) {
	asPythonTwin(t)
	for _, name := range []string{"BayController", "Bay_2", "X1"} {
		want, err := os.ReadFile(filepath.Join("testdata", "corpus", "template_"+name+".scd"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := MakeTemplate(name)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s: %s", name, firstDiff(string(got), string(want)))
		}
	}
	if _, err := MakeTemplate("1bad"); err == nil || err.Error() != "Invalid IED name: '1bad'" {
		t.Errorf("identifier check: %v", err)
	}
}
