package service

import (
	"testing"

	"studio-engine/internal/snmp/profiles"
	"studio-engine/internal/testutil"
	"studio-engine/internal/tools/nativetest"
)

// TestRunOneDevice drives the real studio_snmp (change 0234): Run on one device starts only it,
// a second Run leaves the first running, and Stop on one keeps the other answering.
func TestRunOneDevice(t *testing.T) {
	executable := nativetest.Program(t, "studio_snmp.exe")
	s := New(&memoryProjects{}, executable, testutil.TempDir(t), nil, t.Logf)
	defer s.Close()
	ids := []string{}
	for index, ip := range []string{"127.66.31.1", "127.66.31.2"} {
		device, err := s.AddDevice(profiles.Switch)
		if err != nil {
			t.Fatal(err)
		}
		device.Name = []string{"A", "B"}[index]
		device.IP, device.Port = ip, testutil.FreeUDPPort(t, ip)
		if device, err = s.UpdateDevice(device, false); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, device.ID)
	}
	state := func(id string) string {
		for _, device := range s.Status().Devices {
			if device.ID == id {
				return device.State
			}
		}
		return "missing"
	}
	if status, err := s.StartDevice(ids[0]); err != nil || status.Running != 1 || state(ids[1]) == "running" {
		t.Fatalf("run A: %+v %v", status, err)
	}
	if status, err := s.StartDevice(ids[1]); err != nil || status.Running != 2 || state(ids[0]) != "running" {
		t.Fatalf("run B: %+v %v", status, err)
	}
	if status, err := s.StopDevice(ids[0]); err != nil || status.Running != 1 || state(ids[1]) != "running" ||
		state(ids[0]) == "running" {
		t.Fatalf("stop A: %+v %v", status, err)
	}
	if status, err := s.StopDevice(ids[1]); err != nil || status.Running != 0 {
		t.Fatalf("stop B (last): %+v %v", status, err)
	}
}
