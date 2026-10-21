package native

import (
	"encoding/json"
	"os/exec"
	"sort"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// workerView is what the IED worker observed of one canvas command: the kinds of its
// events in order, plus the reads and writes its MMS access summary counted. Equal views for
// studio_probe and the multiplexed studio_control mean the same requests reached the server.
type workerView struct {
	kinds  []string
	reads  int64
	writes []string
}

func collect(events chan *pyjson.Object, window time.Duration) workerView {
	var view workerView
	end := time.After(window)
	for {
		select {
		case event := <-events:
			kind, _ := event.Get("event")
			name, _ := kind.(string)
			if peer, _ := event.Get("peer"); name == "report_control" && peer == "" {
				continue // the server's own buffered-report timer, not the client
			}
			if name == "mms_access" {
				clients, _ := event.Get("clients")
				for _, item := range clients.([]any) {
					client := item.(*pyjson.Object)
					total, _ := client.Get("total_reads")
					view.reads += total.(int64)
					writes, _ := client.Get("writes")
					view.writes = append(view.writes, writes.(*pyjson.Object).Keys()...)
				}
				continue
			}
			if name != "native_diagnostic" {
				view.kinds = append(view.kinds, name)
			}
		case <-end:
			sort.Strings(view.writes)
			return view
		}
	}
}

func TestCanvasOperateMatchesProbeOnTheWorker(t *testing.T) {
	control := nativeBin(t, "studio_control.exe")
	probe := nativeBin(t, "studio_probe.exe")
	events := make(chan *pyjson.Object, 1024)
	_, _ = startTemplate(t, events, "127.63.0.21", 21021)
	collect(events, 1500*time.Millisecond)
	client, err := StartControl(control)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	hello, err := client.Hello()
	if err != nil || !HasCapabilities(hello, []string{MuxCapability}) {
		t.Skipf("studio_control without %s: %v", MuxCapability, err)
	}
	for _, value := range []bool{true, false} {
		ref := "BayACTRL/XCBR1.Pos"
		output, err := exec.Command(probe, "127.63.0.21", "21021", "control", ref, map[bool]string{true: "true", false: "false"}[value]).Output()
		if err != nil {
			t.Fatalf("probe: %v %s", err, output)
		}
		var probeResult map[string]any
		_ = json.Unmarshal(output, &probeResult)
		byProbe := collect(events, 1500*time.Millisecond)
		started := time.Now()
		result, err := client.CanvasOperate("127.63.0.21", 21021, ref, value, 0, 15*time.Second)
		elapsed := time.Since(started)
		if err != nil {
			t.Fatal(err)
		}
		encoded, _ := pyjson.Compact(result)
		probeText := strings.TrimSpace(string(output))
		if string(encoded) != probeText {
			t.Fatalf("result %s, probe printed %s", encoded, probeText)
		}
		byMux := collect(events, 1500*time.Millisecond)
		if strings.Join(byProbe.kinds, ",") != strings.Join(byMux.kinds, ",") || byProbe.reads != byMux.reads ||
			strings.Join(byProbe.writes, ",") != strings.Join(byMux.writes, ",") {
			t.Fatalf("worker saw probe %+v, mux %+v", byProbe, byMux)
		}
		t.Logf("value %v: worker events %v, reads %d, writes %v; mux command %v", value, byMux.kinds,
			byMux.reads, byMux.writes, elapsed)
	}
}

// TestCanvasOperateTiming records command times with and without a kept association.
func TestCanvasOperateTiming(t *testing.T) {
	client, err := StartControl(nativeBin(t, "studio_control.exe"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if hello, _ := client.Hello(); !HasCapabilities(hello, []string{MuxCapability}) {
		t.Skip("studio_control without the multiplexed mode")
	}
	_, _ = startTemplate(t, nil, "127.63.0.23", 21023)
	for _, keep := range []time.Duration{0, 5 * time.Second} {
		var times []time.Duration
		for i := 0; i < 6; i++ {
			started := time.Now()
			result, err := client.CanvasOperate("127.63.0.23", 21023, "BayACTRL/XCBR1.Pos", i%2 == 0, keep, 15*time.Second)
			times = append(times, time.Since(started))
			if ok, _ := result.Get("ok"); err != nil || ok != true {
				t.Fatalf("operate %d: %v %v", i, result, err)
			}
		}
		t.Logf("keep %v: %v", keep, times)
	}
}

func TestCanvasOperateRefusals(t *testing.T) {
	client, err := StartControl(nativeBin(t, "studio_control.exe"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if hello, _ := client.Hello(); !HasCapabilities(hello, []string{MuxCapability}) {
		t.Skip("studio_control without the multiplexed mode")
	}
	_, err = client.CanvasOperate("127.63.0.29", 21029, "X/Y.Pos", true, 0, 15*time.Second)
	if err == nil || !strings.HasPrefix(err.Error(), "Could not connect to 127.63.0.29:21029") {
		t.Fatalf("closed port: %v", err)
	}
	_, _ = startTemplate(t, nil, "127.63.0.22", 21022)
	result, err := client.CanvasOperate("127.63.0.22", 21022, "BayACTRL/NoSuch.Pos", true, 0, 15*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := result.Get("ok"); ok != false {
		t.Fatalf("unknown control: %v", ok)
	}
}
