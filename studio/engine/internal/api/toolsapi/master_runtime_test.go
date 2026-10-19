package toolsapi

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/nativetest"
	"studio-engine/internal/tools/worker"
)

// device starts studio_gateway with the points of tests/protocol/test_gateway_wire.py.
func device(t *testing.T, port int64, commands *[]string) worker.Process {
	t.Helper()
	program := nativetest.Program(t, "studio_gateway.exe")
	var process worker.Process
	process, err := worker.Spawn(program, func(event *pyjson.Object) {
		if kind, _ := event.Get("event"); kind == "gw_command" {
			id, _ := event.Get("point_id")
			token, _ := event.Get("token")
			*commands = append(*commands, id.(string))
			go process.Request("command_result", worker.DefaultTimeout, pyjson.NewObject().Set("token", token).Set("ok", true))
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(process.Close)
	p := func(id, kind string, address int64) *pyjson.Object {
		return pyjson.NewObject().Set("id", id).Set("kind", kind).Set("address", address)
	}
	points := []any{p("sp", "single", 1001), p("dp", "double", 2001), p("mv", "float", 3001), p("iv", "integer", 4001),
		p("cb", "command_double", 6001), p("real", "command_single", 6002).Set("monitor_only", true)}
	settings := pyjson.NewObject().Set("bind", "127.0.0.1").Set("port", port).Set("common_address", int64(1))
	if _, err := process.Request("start", 15*time.Second, pyjson.NewObject().Set("protocol", "iec104").
		Set("settings", settings).Set("points", points)); err != nil {
		t.Fatal(err)
	}
	v := func(id string, value any, quality int64) *pyjson.Object {
		return pyjson.NewObject().Set("id", id).Set("value", value).Set("quality", quality)
	}
	process.Request("update", worker.DefaultTimeout, pyjson.NewObject().Set("values",
		[]any{v("sp", true, 0), v("dp", int64(2), 0), v("mv", 230.5, 0), v("iv", int64(42), 2)}))
	return process
}

func waitFor(check func() bool) bool {
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		if check() {
			return true
		}
	}
	return check()
}

// TestMasterRoleDiscoversNamesAndCommandsADevice is test_master_runtime.py through the Go routes.
func TestMasterRoleDiscoversNamesAndCommandsADevice(t *testing.T) {
	var commands []string
	device(t, 20411, &commands)
	executables := map[string]string{"gateway": nativetest.Program(t, "studio_gateway.exe"),
		"master": nativetest.Program(t, "studio_master.exe")}
	deps := newDeps(&fakeProjects{}, &fakeRuntime{state: "stopped"}, executables)
	defer deps.Master.Disconnect(false)
	srv := server(t, deps)
	status, text := call(t, srv, "POST", "/tools/master/connect", "")
	if status != 400 || !strings.Contains(text, "SCADA master first") {
		t.Fatalf("%d %s", status, text)
	}
	if _, text = call(t, srv, "POST", "/tools/gateway/role", `{"role":"master"}`); text != "{\"role\":\"master\"}\n" {
		t.Fatal(text)
	}
	if status, text = call(t, srv, "POST", "/tools/gateway/start", ""); status != 400 || !strings.Contains(text, "Station gateway first") {
		t.Fatalf("%d %s", status, text)
	}
	if status, text = call(t, srv, "PUT", "/tools/master/config", `{"protocol":"iec104","host":"bad host!"}`); status != 400 ||
		!strings.Contains(text, "device address") {
		t.Fatalf("%d %s", status, text)
	}
	_, text = call(t, srv, "PUT", "/tools/master/config", `{"protocol":"iec104","host":"127.0.0.1","settings":{"port":20411}}`)
	saved := decode(t, text)
	if host, _ := saved.Get("host"); host != "127.0.0.1" || !strings.Contains(text, `"gi_minutes":15`) {
		t.Fatal(text)
	}
	if _, text = call(t, srv, "POST", "/tools/master/connect", ""); !strings.Contains(text, `"state":"connected"`) ||
		!strings.Contains(text, `"auto_connect":true`) {
		t.Fatal(text)
	}
	rows := func() map[string]*pyjson.Object {
		_, body := call(t, srv, "GET", "/tools/master/points", "")
		result := map[string]*pyjson.Object{}
		list, _ := decode(t, body).Get("points")
		for _, raw := range list.([]any) {
			key, _ := raw.(*pyjson.Object).Get("key")
			result[key.(string)] = raw.(*pyjson.Object)
		}
		return result
	}
	if !waitFor(func() bool { return len(rows()) >= 4 }) {
		t.Fatalf("points %v", rows())
	}
	found := rows()
	check := func(key, field string, want any) {
		t.Helper()
		row := found[key]
		if row == nil {
			t.Fatalf("no row %s in %v", key, found)
		}
		if got, _ := row.Get(field); got != want {
			t.Fatalf("%s.%s = %v want %v", key, field, got, want)
		}
	}
	check("double:2001", "value_text", "closed (ON)")
	check("double:2001", "ticked", true)
	check("double:2001", "command", "double")
	check("float:3001", "command", "setpoint_float")
	invalid := "scaled:4001"
	if _, ok := found["integer:4001"]; ok {
		invalid = "integer:4001"
	}
	check(invalid, "quality_text", "invalid")
	call(t, srv, "PUT", "/tools/master/label", `{"key":"double:2001","label":"Feeder 1 breaker"}`)
	call(t, srv, "PUT", "/tools/master/selection", `{"unticked":["single:1001"]}`)
	found = rows()
	check("double:2001", "label", "Feeder 1 breaker")
	check("single:1001", "ticked", false)
	if _, csv := call(t, srv, "GET", "/tools/master/points.csv", ""); !strings.HasPrefix(csv, "group,address,label") ||
		!strings.Contains(csv, "Feeder 1 breaker") {
		t.Fatal(csv)
	}
	if status, text = call(t, srv, "POST", "/tools/master/command", `{"kind":"double","address":6001,"value":2}`); status != 400 ||
		!strings.Contains(text, "Confirm") {
		t.Fatal(text)
	}
	_, text = call(t, srv, "POST", "/tools/master/command",
		`{"kind":"double","address":6001,"value":2,"select":true,"confirm":true}`)
	if !strings.Contains(text, `"ok":true`) || !strings.Contains(text, `"terminated":true`) || strings.Join(commands, ",") != "cb" {
		t.Fatalf("%s %v", text, commands)
	}
	_, text = call(t, srv, "POST", "/tools/master/command", `{"kind":"single","address":6002,"value":1,"confirm":true}`)
	if !strings.Contains(text, `"ok":false`) || !strings.Contains(text, "refused") {
		t.Fatal(text)
	}
	if _, text = call(t, srv, "GET", "/tools/master/commands", ""); strings.Count(text, `"state":"confirmed"`) != 1 ||
		strings.Count(text, `"state":"refused"`) != 1 {
		t.Fatal(text)
	}
	if _, text = call(t, srv, "GET", "/tools/master/traffic", ""); !strings.Contains(text, "All values received") {
		t.Fatal(text)
	}
	if _, text = call(t, srv, "GET", "/activity", ""); !strings.Contains(text, `"master":{"host":"127.0.0.1","points":`) ||
		!strings.Contains(text, `"state":"connected"},"scan"`) {
		t.Fatal(text)
	}
	if _, text = call(t, srv, "POST", "/tools/master/disconnect", ""); !strings.Contains(text, `"auto_connect":false`) {
		t.Fatal(text)
	}
	call(t, srv, "PUT", "/tools/master/config", `{"protocol":"dnp3","host":"127.0.0.1"}`)
	status, text = call(t, srv, "POST", "/tools/master/command", `{"kind":"double","address":1,"value":2,"confirm":true}`)
	if status != 400 || !strings.Contains(text, "not available for dnp3") {
		t.Fatalf("%d %s", status, text)
	}
}
