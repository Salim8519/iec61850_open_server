package explorer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/memory"
	"studio-engine/internal/tools/nativetest"
	"studio-engine/internal/tools/pyoracle"
	"studio-engine/internal/tools/scan"
	"studio-engine/internal/tools/worker"
)

// Loopback range of stream E (never the owner's 127.50.0.x); ports >= 20000.
const deviceIP, devicePort = "127.65.0.32", int64(20132)

// startDevice runs studio_ied with the public ExploreTest template, as test_explorer_wire.py.
func startDevice(t *testing.T) worker.Process {
	t.Helper()
	ied := nativetest.Program(t, "studio_ied.exe")
	documents := pyoracle.Run(t, filepath.Join("testdata", "template_model.py"), nil, "ExploreTest")
	model := documents[0].(*pyjson.Object)
	cfg, _ := model.Get("cfg")
	controls, _ := model.Get("controls")
	path := filepath.Join(t.TempDir(), "device.cfg")
	if err := os.WriteFile(path, []byte(cfg.(string)), 0o644); err != nil {
		t.Fatal(err)
	}
	process, err := worker.Spawn(ied, nil)
	if err != nil {
		t.Fatal(err)
	}
	fields := pyjson.NewObject().Set("cfg", path).Set("ip", deviceIP).Set("port", devicePort).
		Set("controls", controls).Set("edition", int64(2))
	if _, err := process.Request("start", 30*time.Second, fields); err != nil {
		process.Close()
		t.Fatal(err)
	}
	t.Cleanup(process.Close)
	return process
}

func TestExplorerWatchesReadsAndOperatesARealServer(t *testing.T) {
	live := nativetest.Program(t, "studio_live.exe")
	controlExe := nativetest.Program(t, "studio_control.exe")
	startDevice(t)
	folder := t.TempDir()
	operator := control.NewClient(controlExe, nil)
	defer operator.Close()
	var events []string
	service := NewService(live, nil, func(kind string, _ *pyjson.Object) { events = append(events, kind) },
		memory.New(folder), operator)
	defer service.Close()
	status, err := service.Connect(" "+deviceIP+" ", devicePort)
	if err != nil {
		t.Fatal(err)
	}
	if get(status, "state") != "connected" || get(status, "polling") != true {
		t.Fatalf("status after connect: %v", status)
	}
	watch := asList(get(status, "watch"))
	refs := map[string]bool{}
	for _, item := range watch {
		refs[get(item.(*pyjson.Object), "ref").(string)] = true
	}
	if !refs["ExploreTestCTRL/XCBR1.Pos.stVal"] || len(watch) > MaxPoints {
		t.Fatalf("key points: %v", refs)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && get(service.Status(), "reads").(int64) == 0 {
		time.Sleep(50 * time.Millisecond)
	}
	if get(service.Status(), "reads").(int64) == 0 {
		t.Fatal("polling never read")
	}
	view, err := service.Node("ExploreTestCTRL/XCBR1")
	if err != nil || get(view, "name") != "XCBR1" || get(view, "leaf_count").(int64) == 0 {
		t.Fatalf("node view %v %v", view, err)
	}
	answer, err := service.Operate("ExploreTestCTRL/XCBR1.Pos", true, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(get(answer, "summary").(string), "The device ") || get(answer, "observed") == nil {
		t.Fatalf("operate %v", answer)
	}
	result := get(answer, "result").(*pyjson.Object)
	if get(result, "ok") != true {
		t.Fatalf("operate refused: %v", result)
	}
	remembered, _ := memory.New(folder).Get().Get("explorer")
	if get(remembered.(*pyjson.Object), "host") != deviceIP {
		t.Fatalf("not remembered: %v", remembered)
	}
	if status := service.Disconnect(); get(status, "state") != "disconnected" || get(status, "polling") != false {
		t.Fatalf("disconnect %v", status)
	}
	if strings.Join(events, ",") != "explorer_connected,explorer_command" {
		t.Fatalf("events %v", events)
	}
	finder := scan.NewService(live, nil, nil, nil)
	if _, err := finder.Start("127.65.0.30-34", devicePort); err != nil {
		t.Fatal(err)
	}
	for time.Now().Before(deadline.Add(20*time.Second)) && get(finder.Status(), "state") == "scanning" {
		time.Sleep(50 * time.Millisecond)
	}
	found := asList(get(finder.Status(), "found"))
	if len(found) != 1 || get(found[0].(*pyjson.Object), "identified") != true ||
		get(finder.Status(), "message") != "Done. 1 address(es) answer on port 20132, 1 identified as IEC 61850." {
		t.Fatalf("scan %v", finder.Status())
	}
}
