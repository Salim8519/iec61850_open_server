package native

import (
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
)

// nativeBin is the folder with the untracked native workers (STUDIO_NATIVE_BIN overrides).
func nativeBin(t *testing.T, name string) string {
	t.Helper()
	folder := os.Getenv("STUDIO_NATIVE_BIN")
	if folder == "" {
		folder = filepath.Join("..", "..", "..", "native", "build", "Release")
	}
	path, _ := filepath.Abs(filepath.Join(folder, name))
	if _, err := os.Stat(path); err != nil {
		t.Skipf("native build required: %s", path)
	}
	return path
}

// templateIed returns the first IED of the synthetic template project (public template SCL).
func templateIed(t *testing.T, index int) *pyjson.Object {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "runtime", "testdata", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := pyjson.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	ieds, _ := decoded.(*pyjson.Object).Get("ieds")
	return ieds.([]any)[index].(*pyjson.Object)
}

func get(t *testing.T, object *pyjson.Object, key string) any {
	t.Helper()
	value, ok := object.Get(key)
	if !ok {
		t.Fatalf("missing %q", key)
	}
	return value
}

// startTemplate starts studio_ied with the template IED on an isolated loopback endpoint.
func startTemplate(t *testing.T, events chan *pyjson.Object, ip string, port int64) (*Ied, *pyjson.Object) {
	t.Helper()
	ied := templateIed(t, 0)
	cfg := filepath.Join(t.TempDir(), "ied.cfg")
	if err := os.WriteFile(cfg, []byte(get(t, ied, "cfg").(string)), 0o644); err != nil {
		t.Fatal(err)
	}
	client, err := StartIed(nativeBin(t, "studio_ied.exe"), func(event *pyjson.Object) {
		if events != nil {
			events <- event
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	goose := pyjson.NewObject().Set("publish", false).Set("interface", "")
	fields := pyjson.NewObject().Set("cfg", cfg).Set("ip", ip).Set("port", port).
		Set("controls", get(t, ied, "controls")).Set("edition", get(t, ied, "edition")).Set("goose", goose)
	started, err := client.StartModel(fields)
	if err != nil {
		t.Fatal(err)
	}
	return client, started
}
