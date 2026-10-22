package service

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/gateway"
	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
)

// plannedBay plans the bay breaker's command with its status ("From an IED", ADR-0105) and
// saves the planned points as the IEC 104 list.
func plannedBay(t *testing.T, projects *fakeProjects) []any {
	t.Helper()
	ied := addpoints.NewIndex(projects.IEDs()).ByID("bay")
	results := addpoints.FromIED("iec104", ied, "command_double", []addpoints.Pick{{Ref: "BayCTRL/XCBR1.Pos"}}, true,
		nil, projects.IEDs())
	if !addpoints.OK(results) {
		t.Fatalf("plan %v", results)
	}
	clean, problems, err := gateway.ValidatePoints("iec104", addpoints.Points(results), projects.IEDs())
	if err != nil || len(problems) != 0 {
		t.Fatal(problems, err)
	}
	section := projects.Gateway()
	projects.SaveGateway(with(section, "points", with(get(section, "points").(*pyjson.Object), "iec104", clean)))
	return clean
}

// TestLinkedCommandPointOperatesTheIED: a command point added from an IED is linked to the
// IED command at once: a command on it goes through the gateway command path to the IED
// (studio_control's Operate) and the paired status point is observed.
func TestLinkedCommandPointOperatesTheIED(t *testing.T) {
	hub := newHub()
	projects := &fakeProjects{ieds: []*pyjson.Object{bay()}, section: gateway.EmptyGateway()}
	operator := &fakeOperator{hub: hub, status: "BayCTRL/XCBR1.Pos.stVal"}
	manager := NewManager(projects, hub, operator, "", nil)
	clean := plannedBay(t, projects)
	if len(clean) != 2 || get(clean[0].(*pyjson.Object), "address") != int64(50001) ||
		get(clean[1].(*pyjson.Object), "address") != int64(20001) {
		t.Fatalf("planned %v", clean)
	}
	answer := manager.Send("bay:BayCTRL/XCBR1.Pos:command_double", true)
	if get(answer, "ok") != true {
		t.Fatalf("send %v", answer)
	}
	if strings.Join(operator.calls, ",") != "BayCTRL/XCBR1.Pos" {
		t.Fatalf("operate calls %v", operator.calls)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		entries := get(manager.Commands.After(0, 10), "entries").([]any)
		last := entries[len(entries)-1].(*pyjson.Object)
		if get(last, "observed") == "BayCTRL/XCBR1.Pos.stVal is now closed (ON)" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("status not observed: %v", manager.Commands.After(0, 10))
}

// TestLinkedCommandOverIEC104: an independent IEC 104 master (lib60870) sends a double
// command to the address planned for the command; the IED is operated and the paired
// status point reaches the master as a spontaneous change.
func TestLinkedCommandOverIEC104(t *testing.T) {
	port := int64(testutil.FreePort(t, "127.0.0.1"))
	manager, _, projects := setup(t, port)
	plannedBay(t, projects)
	if _, err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	frames := master104(t, int(port), "dc:50001:2", "wait:1500")
	confirmed, spontaneous := false, false
	for _, frame := range frames {
		if frame["type"] == 46.0 && frame["cot"] == 7.0 && frame["negative"] == false {
			confirmed = true
		}
		if frame["cot"] == 3.0 {
			for _, raw := range frame["objects"].([]any) {
				if object := raw.(map[string]any); object["ioa"] == 20001.0 && object["value"] == 2.0 {
					spontaneous = true
				}
			}
		}
	}
	if !confirmed || !spontaneous {
		t.Fatalf("confirmed %v spontaneous %v frames %v", confirmed, spontaneous, frames)
	}
}
