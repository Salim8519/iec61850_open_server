package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/memory"
	"studio-engine/internal/tools/worker"
)

type fakeLive struct{ asked []string }

func (f *fakeLive) Request(op string, _ time.Duration, fields *pyjson.Object) (any, error) {
	ip, _ := fields.Get("ip")
	f.asked = append(f.asked, op+" "+ip.(string))
	if ip == "127.65.0.3" {
		return nil, apperr.RuntimeError("Could not connect")
	}
	return pyjson.NewObject().Set("identified", true).Set("vendor", "V").Set("model", "M").Set("revision", "1").
		Set("devices", []any{"LD0"}).Set("connect_ms", 2.5).Set("identity_note", "note"), nil
}
func (f *fakeLive) Alive() bool { return true }
func (f *fakeLive) Close()      {}

func TestScanFindsAndIdentifiesInAddressOrder(t *testing.T) {
	folder := t.TempDir()
	exe := filepath.Join(folder, "studio_live.exe")
	os.WriteFile(exe, nil, 0o644)
	live := &fakeLive{}
	probe := func(host string, port int64) (float64, bool) {
		return 1.5, strings.HasSuffix(host, ".2") || strings.HasSuffix(host, ".3")
	}
	spawn := func(string, func(*pyjson.Object)) (worker.Process, error) { return live, nil }
	store := memory.New(folder)
	service := NewService(exe, store, probe, spawn)
	if status := service.Status(); mustGet(status, "state") != "idle" || mustGet(status, "port") != int64(102) {
		t.Fatalf("idle status %v", status)
	}
	if _, err := service.Start("127.65.0.1-4", int64(20102)); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for mustGet(service.Status(), "state") == "scanning" && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	status := service.Status()
	encoded, _ := pyjson.Flask(status.Set("started", nil).Set("seconds", nil))
	want := `{"found":[{"connect_ms":2.5,"devices":["LD0"],"identified":true,"ip":"127.65.0.2","model":"M",` +
		`"port":20102,"reason":"note","revision":"1","tcp_ms":1.5,"vendor":"V"},{"devices":[],"identified":false,` +
		`"ip":"127.65.0.3","model":"","port":20102,"reason":"Could not connect","revision":"","tcp_ms":1.5,"vendor":""}],` +
		`"message":"Done. 2 address(es) answer on port 20102, 1 identified as IEC 61850.","port":20102,"probed":4,` +
		`"range":"127.65.0.1-4","seconds":null,"started":null,"state":"done","total":4}` + "\n"
	if string(encoded) != want {
		t.Fatalf("status\n%s\nwant\n%s", encoded, want)
	}
	find, _ := store.Get().Get("find")
	if mustGet(find.(*pyjson.Object), "range") != "127.65.0.1-4" || mustGet(find.(*pyjson.Object), "port") != int64(20102) {
		t.Fatalf("remembered %v", find)
	}
}

func TestScanRefusals(t *testing.T) {
	service := NewService(filepath.Join(t.TempDir(), "missing.exe"), nil, nil, nil)
	if _, err := service.Start("127.65.0.1", int64(102)); err == nil || err.Error() !=
		"The live client (studio_live.exe) is not built yet; rebuild the native programs" {
		t.Fatal(err)
	}
	if _, err := service.Start("127.65.0.1", true); err == nil || !strings.Contains(err.Error(), "studio_live") {
		t.Fatal("True is an int port in Python, so the next check refuses", err)
	}
}

func mustGet(object *pyjson.Object, key string) any {
	value, _ := object.Get(key)
	return value
}
