package service

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
)

func own(kind string, address int64, description string, extra ...any) *pyjson.Object {
	point := pyjson.NewObject().Set("kind", kind).Set("source", gateway.OwnSource).Set("address", address).
		Set("scale", 1.0).Set("description", description)
	for index := 0; index+1 < len(extra); index += 2 {
		point.Set(extra[index].(string), extra[index+1])
	}
	return point
}

// ioaValues are the information objects of the frames with one cause of transmission.
func ioaValues(frames []map[string]any, cot float64) map[float64]any {
	values := map[float64]any{}
	for _, frame := range frames {
		if frame["cot"] != cot {
			continue
		}
		objects, _ := frame["objects"].([]any)
		for _, raw := range objects {
			object := raw.(map[string]any)
			values[object["ioa"].(float64)] = object["value"]
		}
	}
	return values
}

// TestOwnPointsOverIEC104: an independent IEC 104 master (lib60870) reads own points next to
// an IED point, sees a typed value change arrive spontaneously, and a double command on an
// own command point moves the own breaker it drives (lane G, Go-only).
func TestOwnPointsOverIEC104(t *testing.T) {
	port := int64(testutil.FreePort(t, "127.0.0.1"))
	manager, _, projects := setup(t, port)
	raw := []any{point("float", 30001, "BayCTRL/MMXU1.TotW.mag.f"),
		own("single", 10001, "Pump running", "value", true),
		own("float", 30002, "Tank level", "value", 12.5),
		own("double", 20002, "Spare breaker", "value", int64(1)),
		own("command_double", 50002, "Spare breaker command", "target", "own:3")}
	clean, problems, err := gateway.ValidatePoints("iec104", raw, projects.IEDs())
	if err != nil || len(problems) != 0 {
		t.Fatal(problems, err)
	}
	section := projects.Gateway()
	projects.SaveGateway(with(section, "points", with(get(section, "points").(*pyjson.Object), "iec104", clean)))
	if _, err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	read := ioaValues(master104(t, int(port), "gi"), 20)
	if read[30001] != 230.5 || read[30002] != 12.5 || read[20002] != 1.0 || (read[10001] != true && read[10001] != 1.0) {
		t.Fatalf("interrogation %v", read)
	}
	done := make(chan []map[string]any)
	go func() { done <- master104(t, int(port), "wait:2500") }()
	time.Sleep(1200 * time.Millisecond) // the master is connected and waiting
	row, err := manager.SetOwnValue("own:2", 99.5)
	if err != nil || get(row, "value_text") != "99.5" || get(row, "online") != true || get(row, "ied_name") != "Own point" {
		t.Fatalf("set value %v %v", row, err)
	}
	if spontaneous := ioaValues(<-done, 3); spontaneous[30002] != 99.5 {
		t.Fatalf("no spontaneous 99.5: %v", spontaneous)
	}
	frames := master104(t, int(port), "dc:50002:2:select", "dc:50002:2", "wait:1500")
	var answers []string
	for _, frame := range frames {
		if frame["type"] == 46.0 && frame["negative"] == false {
			answers = append(answers, "ok")
		}
	}
	if len(answers) < 2 || ioaValues(frames, 3)[20002] != 2.0 {
		t.Fatalf("command answers %v spontaneous %v", answers, ioaValues(frames, 3))
	}
	entries := get(manager.Commands.After(0, 200), "entries").([]any)
	last := entries[len(entries)-1].(*pyjson.Object)
	if get(last, "applied") != "done: own point, no device is operated" ||
		get(last, "observed") != "Spare breaker is now closed (ON)" || get(last, "label") != "Spare breaker command ON / close" {
		t.Fatalf("command log %v", last)
	}
	if _, err := manager.SetOwnValue("own:4", 1); err == nil {
		t.Fatal("a command point took a typed value")
	}
	if _, err := manager.SetOwnValue("own:1", int64(5)); err == nil || !strings.Contains(err.Error(), "OFF (0) or ON (1)") {
		t.Fatalf("bad value %v", err)
	}
	table, _ := manager.Table("iec104")
	texts := []string{}
	for _, raw := range table[1:] {
		texts = append(texts, get(raw.(*pyjson.Object), "value_text").(string))
	}
	if strings.Join(texts, "|") != "ON|99.5|ON (closed)|Last command: close" {
		t.Fatalf("table %v", texts)
	}
	manager.Stop(true)
	saved := pointsOf(projects.Gateway(), "iec104")
	if get(saved[2].(*pyjson.Object), "value") != 99.5 || get(saved[3].(*pyjson.Object), "value") != int64(2) ||
		get(saved[4].(*pyjson.Object), "value") != int64(2) {
		t.Fatalf("saved values %v", saved)
	}
}
