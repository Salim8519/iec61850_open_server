package netcfg

import (
	"slices"
	"strconv"
	"sync"
	"testing"

	"studio-engine/internal/pyjson"
)

// TestPreviewProgressShowsEveryStepAndEachAnswer: after a preview every step is done and each
// checked row says whether its address is free or in use; the preview result is unchanged.
func TestPreviewProgressShowsEveryStepAndEachAnswer(t *testing.T) {
	s, _ := snmpService("192.168.10.21")
	before := s.PreviewProgress()
	if get(before, "running") != false || len(get(before, "rows").([]any)) != 0 {
		t.Fatalf("progress before any check: %v", before)
	}
	rows, _, _ := previewRows(t, s, nil)
	progress := s.PreviewProgress()
	if get(progress, "running") != false || get(progress, "run") != int64(1) {
		t.Fatalf("progress after the check: %v", progress)
	}
	keys := []string{}
	for _, step := range get(progress, "steps").([]any) {
		keys = append(keys, pyStr(get(step.(*pyjson.Object), "key")))
		if get(step.(*pyjson.Object), "state") != "done" {
			t.Fatalf("step not done: %v", step)
		}
	}
	if !slices.Equal(keys, []string{"cards", "addresses", "firewall", "plan"}) {
		t.Fatalf("steps %v", keys)
	}
	states := map[string]string{}
	for _, row := range get(progress, "rows").([]any) {
		object := row.(*pyjson.Object)
		states[pyStr(get(object, "id"))] = pyStr(get(object, "state"))
	}
	if states["ied1"] != "in_use" || states["snmp:snmp1"] != "free" || states["snmp:snmp2"] != "free" {
		t.Fatalf("row states %v", states)
	}
	if get(rows["ied1"], "status") != "in_use_on_network" {
		t.Fatalf("the preview itself changed: %v", rows["ied1"])
	}
	if get(progress, "asked") != get(progress, "answered") {
		t.Fatalf("asked and answered differ: %v", progress)
	}
}

// TestARPProberReportsEachAnswerAndAsksUpTo64AtOnce.
func TestARPProberReportsEachAnswerAndAsksUpTo64AtOnce(t *testing.T) {
	var mu sync.Mutex
	inFlight, most := 0, 0
	release, opened := make(chan struct{}), sync.Once{}
	addresses := []string{}
	for index := 1; index <= 100; index++ {
		addresses = append(addresses, "10.0.0."+strconv.Itoa(index))
	}
	heard := map[string]bool{}
	prober := ARPProber{
		Answers: func(address string) bool {
			mu.Lock()
			inFlight++
			most = max(most, inFlight)
			if inFlight == arpParallel {
				opened.Do(func() { close(release) })
			}
			mu.Unlock()
			<-release
			mu.Lock()
			inFlight--
			mu.Unlock()
			return address == "10.0.0.7"
		},
		Report: func(address string, used bool) {
			mu.Lock()
			heard[address] = used
			mu.Unlock()
		},
	}
	used, err := prober.InUse(addresses)
	if err != nil || len(used) != 1 || !used["10.0.0.7"] {
		t.Fatalf("in use %v %v", used, err)
	}
	if most != 64 || len(heard) != 100 || !heard["10.0.0.7"] || heard["10.0.0.8"] {
		t.Fatalf("at most %d at once, heard %d", most, len(heard))
	}
}
