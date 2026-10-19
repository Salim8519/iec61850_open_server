package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

func dumps(t *testing.T, value any) string {
	t.Helper()
	data, err := pyjson.Dumps(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPreferencesUpdateWritesOnlyValidResults(t *testing.T) {
	dir := t.TempDir()
	store := NewPreferences(dir)
	if store.Load() != nil {
		t.Fatal("missing file should load as nil")
	}
	var seen []any
	accept := func(stored, changes any) (*pyjson.Object, error) {
		seen = append(seen, stored)
		return pyjson.NewObject().Set("tour_seen_version", changes), nil
	}
	after, err := store.Update(int64(1), accept)
	if err != nil || dumps(t, after) != `{"tour_seen_version": 1}` || seen[0] != nil {
		t.Fatalf("update: %v %v", after, err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, PreferencesName))
	if string(data) != "{\r\n  \"tour_seen_version\": 1\r\n}" {
		t.Fatalf("file %q", data)
	}
	refuse := func(stored, changes any) (*pyjson.Object, error) {
		if dumps(t, stored) != `{"tour_seen_version": 1}` {
			t.Errorf("stored %v", stored)
		}
		return nil, apperr.ValueError("Preferences must be a JSON object")
	}
	if _, err := store.Update("x", refuse); err == nil || err.Error() != "Preferences must be a JSON object" {
		t.Fatalf("refusal: %v", err)
	}
	if again, _ := os.ReadFile(filepath.Join(dir, PreferencesName)); string(again) != string(data) {
		t.Fatal("a refused update changed the file")
	}
}

const defaults = `{"explorer": {"host": "", "port": 102, "interval_ms": 1000, "watch_by_host": {}}, ` +
	`"find": {"range": "", "port": 102, "choice": ""}}`

func TestToolMemoryDefaultsAndDamage(t *testing.T) {
	dir := t.TempDir()
	memory := NewToolMemory(dir)
	if got := dumps(t, memory.Get()); got != defaults {
		t.Fatalf("defaults %s", got)
	}
	for _, content := range []string{"[1]", "{", "\xef\xbb\xbf{}", `{"explorer": 5, "find": []}`} {
		_ = os.WriteFile(filepath.Join(dir, ToolSettingsName), []byte(content), 0o666)
		if got := dumps(t, memory.Get()); got != defaults {
			t.Errorf("%q gave %s", content, got)
		}
	}
	if _, err := memory.Update("other", pyjson.NewObject()); err == nil || err.Error() != "'other'" {
		t.Fatalf("unknown tool: %v", err)
	}
}

func TestToolMemoryCleaningRules(t *testing.T) {
	var hosts []string
	for index := 0; index < 55; index++ {
		hosts = append(hosts, `"h`+strings.Repeat("x", index%3)+string(rune('A'+index%26))+string(rune('a'+index/26))+`": []`)
	}
	raw := `{"explorer": {"host": "` + strings.Repeat("h", 65) + `", "port": true, "interval_ms": 499, ` +
		`"watch_by_host": {` + strings.Join(hosts, ", ") + `, "keep": [` +
		`{"ref": "LD/LLN0.Mod", "fc": "ST", "type": null}, {"ref": "", "fc": "ST"}, {"ref": "x", "fc": "TOOLONG"},` +
		`{"ref": "y", "fc": "MX", "type": [1, "a", {"k": 2.0}]}, {"ref": "z", "fc": "CO"}, 7]}},` +
		` "find": {"range": "10.0.0.0/24", "port": 2.5, "choice": "é"}}`
	value, err := pyjson.Decode([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	cleaned := clean(value)
	explorer, _ := cleaned.Get("explorer")
	byHost, _ := explorer.(*pyjson.Object).Get("watch_by_host")
	keys := byHost.(*pyjson.Object).Keys()
	if len(keys) != 50 || keys[49] != "keep" {
		t.Fatalf("hosts %d last %q", len(keys), keys[len(keys)-1])
	}
	keep, _ := byHost.(*pyjson.Object).Get("keep")
	want := `[{"ref": "LD/LLN0.Mod", "fc": "ST", "type": "None"}, {"ref": "y", "fc": "MX", ` +
		`"type": "[1, 'a', {'k': 2.0}]"}, {"ref": "z", "fc": "CO", "type": ""}]`
	if got := dumps(t, keep); got != want {
		t.Fatalf("watch\n%s\n%s", got, want)
	}
	explorer.(*pyjson.Object).Delete("watch_by_host")
	if got := dumps(t, cleaned); got != `{"explorer": {"host": "", "port": true, "interval_ms": 1000}, `+
		`"find": {"range": "10.0.0.0/24", "port": 102, "choice": "`+"\\"+`u00e9"}}` {
		t.Fatalf("cleaned %s", got)
	}
}

func TestToolMemoryUpdateWritesCleanedSection(t *testing.T) {
	dir := t.TempDir()
	memory := NewToolMemory(dir)
	section, err := memory.Update("explorer", pyjson.NewObject().Set("host", "10.0.0.5").
		Set("interval_ms", int64(2000)).Set("unknown", true))
	if err != nil || dumps(t, section) != `{"host": "10.0.0.5", "port": 102, "interval_ms": 2000, "watch_by_host": {}}` {
		t.Fatalf("section %v %v", section, err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ToolSettingsName))
	if !strings.HasPrefix(string(data), "{\r\n  \"explorer\": {\r\n    \"host\": \"10.0.0.5\",") ||
		strings.HasSuffix(string(data), "\n") {
		t.Fatalf("file %q", data)
	}
}
