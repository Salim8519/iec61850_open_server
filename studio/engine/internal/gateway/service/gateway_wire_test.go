package service

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/nativetest"
)

// bay builds a synthetic simulated IED: one breaker position, its quality and time, one
// measurement and the breaker's double-point control.
func bay() *pyjson.Object {
	signal := func(ref, fc, kind string, value any) *pyjson.Object {
		return pyjson.NewObject().Set("ref", ref).Set("fc", fc).Set("kind", kind).Set("value", value)
	}
	return pyjson.NewObject().Set("id", "bay").Set("name", "Bay").Set("role", "simulated").Set("state", "running").
		Set("signals", []any{signal("BayCTRL/XCBR1.Pos.stVal", "ST", "Dbpos", int64(1)),
			signal("BayCTRL/MMXU1.TotW.mag.f", "MX", "FLOAT32", 230.5)}).
		Set("controls", []any{pyjson.NewObject().Set("ref", "BayCTRL/XCBR1.Pos").
			Set("status", "BayCTRL/XCBR1.Pos.stVal").Set("cdc", "DPC")}).
		Set("datasets", []any{})
}

func point(kind string, address int64, ref string) *pyjson.Object {
	return pyjson.NewObject().Set("kind", kind).Set("ied", "bay").Set("ref", ref).Set("address", address).
		Set("scale", 1.0).Set("description", "")
}

// master104 runs the independent IEC 104 master (lib60870) used by the Python wire tests.
func master104(t *testing.T, port int, steps ...string) []map[string]any {
	t.Helper()
	program := nativetest.Program(t, "studio_test_iec104_master.exe")
	output, _ := exec.Command(program, append([]string{"127.0.0.1", strconv.Itoa(port), "1"}, steps...)...).Output()
	var frames []map[string]any
	for _, line := range strings.Split(string(output), "\n") {
		var frame map[string]any
		if strings.HasPrefix(line, "{") && json.Unmarshal([]byte(line), &frame) == nil {
			frames = append(frames, frame)
		}
	}
	return frames
}

func setup(t *testing.T, port int64) (*Manager, *fakeHub, *fakeProjects) {
	t.Helper()
	executable := nativetest.Program(t, "studio_gateway.exe")
	hub := newHub()
	hub.define("bay", "BayCTRL/XCBR1.Pos.stVal", Signal{Value: int64(1), QualityRef: "BayCTRL/XCBR1.Pos.q",
		TimestampRef: "BayCTRL/XCBR1.Pos.t"})
	hub.define("bay", "BayCTRL/XCBR1.Pos.q", Signal{Value: int64(0)})
	hub.define("bay", "BayCTRL/XCBR1.Pos.t", Signal{Value: int64(1700000000000)})
	hub.define("bay", "BayCTRL/MMXU1.TotW.mag.f", Signal{Value: 230.5})
	section := gateway.EmptyGateway()
	settings := get(section, "settings").(*pyjson.Object)
	settings.Set("iec104", get(settings, "iec104").(*pyjson.Object).Set("port", port))
	points := get(section, "points").(*pyjson.Object)
	points.Set("iec104", []any{point("double", 20001, "BayCTRL/XCBR1.Pos.stVal"),
		point("float", 30001, "BayCTRL/MMXU1.TotW.mag.f"), point("command_double", 50001, "BayCTRL/XCBR1.Pos")})
	projects := &fakeProjects{ieds: []*pyjson.Object{bay()}, section: section}
	operator := &fakeOperator{hub: hub, status: "BayCTRL/XCBR1.Pos.stVal"}
	manager := NewManager(projects, hub, operator, executable, nil)
	t.Cleanup(func() { manager.Stop(false) })
	return manager, hub, projects
}

func TestGatewayServesValuesAndCommandsOverIEC104(t *testing.T) {
	manager, hub, projects := setup(t, 20404)
	status, err := manager.Start()
	if err != nil {
		t.Fatal(err)
	}
	if get(status, "state") != "running" || get(status, "auto_start") != true ||
		get(get(status, "opened").(*pyjson.Object), "port") != int64(20404) {
		t.Fatalf("status %v", status)
	}
	time.Sleep(300 * time.Millisecond) // the first batch leaves after 100 ms
	objects := map[float64]map[string]any{}
	for _, frame := range master104(t, 20404, "gi") {
		if frame["cot"] == 20.0 {
			for _, raw := range frame["objects"].([]any) {
				object := raw.(map[string]any)
				objects[object["ioa"].(float64)] = object
			}
		}
	}
	if objects[20001]["value"] != 1.0 || objects[30001]["value"] != 230.5 {
		t.Fatalf("interrogation %v", objects)
	}
	frames := master104(t, 20404, "dc:50001:2:select", "dc:50001:2", "wait:1500")
	var answers []string
	spontaneous := false
	for _, frame := range frames {
		if frame["type"] == 46.0 {
			answers = append(answers, strconv.Itoa(int(frame["cot"].(float64)))+"/"+strconv.FormatBool(frame["negative"].(bool)))
		}
		if frame["cot"] == 3.0 {
			for _, raw := range frame["objects"].([]any) {
				if object := raw.(map[string]any); object["ioa"] == 20001.0 && object["value"] == 2.0 {
					spontaneous = true
				}
			}
		}
	}
	if strings.Join(answers, ",") != "7/false,7/false,10/false" || !spontaneous {
		t.Fatalf("command answers %v spontaneous %v", answers, spontaneous)
	}
	entries := get(manager.Commands.After(0, 200), "entries").([]any)
	last := entries[len(entries)-1].(*pyjson.Object)
	if get(last, "applied") != "operated: The device selected and operated it (sbo-with-enhanced-security)." ||
		get(last, "observed") != "BayCTRL/XCBR1.Pos.stVal is now closed (ON)" || get(last, "label") != "BayCTRL/XCBR1.Pos ON / close" {
		t.Fatalf("command log %v", last)
	}
	manual := manager.Send("bay:BayCTRL/XCBR1.Pos:command_double", false)
	if encoded, _ := pyjson.Flask(manual); string(encoded) != "{\"ok\":true,\"reason\":\"\"}\n" {
		t.Fatalf("manual %s", encoded)
	}
	table, _ := manager.Table("iec104")
	row := table[0].(*pyjson.Object)
	if get(row, "online") != true || !strings.HasPrefix(get(row, "value_text").(string), "OFF (open)") &&
		!strings.HasPrefix(get(row, "value_text").(string), "ON (closed)") {
		t.Fatalf("table row %v", row)
	}
	hub.set("bay", "BayCTRL/XCBR1.Pos.q", int64(2)) // invalid quality reaches the table at once
	table, _ = manager.Table("iec104")
	if text := get(table[0].(*pyjson.Object), "value_text").(string); !strings.HasSuffix(text, " \xc2\xb7 invalid") {
		t.Fatalf("quality not shown: %s", text)
	}
	stopped, err := manager.Stop(true)
	if err != nil || get(stopped, "state") != "stopped" || get(stopped, "auto_start") != false || projects.saves < 2 {
		t.Fatalf("stop %v %v", stopped, err)
	}
	if strings.Join(hub.events, ",") != "gateway_started,gateway_command,gateway_command,gateway_stopped" {
		t.Fatalf("events %v", hub.events)
	}
}

func TestGatewayRefusals(t *testing.T) {
	manager, _, projects := setup(t, 20405)
	section := projects.Gateway()
	projects.SaveGateway(copyObject(section).Set("role", "master"))
	if _, err := manager.Start(); err == nil || err.Error() != "The tool is set to SCADA master; switch it to Station gateway first" {
		t.Fatal(err)
	}
	projects.SaveGateway(section)
	if _, err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Start(); err == nil || err.Error() != "The gateway is already running; stop it first" {
		t.Fatal(err)
	}
	reason := Refusal(point("command_single", 1, "X"), copyObject(bay()).Set("state", "stopped"), true)
	if reason != "Bay is not running; press Run first" {
		t.Fatal(reason)
	}
}
