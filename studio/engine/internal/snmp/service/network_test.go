package service

import (
	"strings"
	"sync/atomic"
	"testing"

	"studio-engine/internal/snmp/manager"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/profiles"
	"studio-engine/internal/snmp/proto"
	"studio-engine/internal/testutil"
	"studio-engine/internal/tools/nativetest"
)

// Test addresses: 127.66.31.1 plays the device's own 127.x address, 127.66.31.10 its lab
// address (a real lab address is on the network card; the service treats both the same).
const (
	ownIP = "127.66.31.1"
	labIP = "127.66.31.10"
)

func sysName(t *testing.T, host string, port int) (string, error) {
	t.Helper()
	session, err := manager.Open(model.Target{Host: host, Port: port, Version: "v2c", Community: "public",
		TimeoutMS: 400, Retries: 0})
	if err != nil {
		return "", err
	}
	defer session.Close()
	answer, err := session.Get([]proto.OID{proto.MustOID("1.3.6.1.2.1.1.5.0")})
	if err != nil {
		return "", err
	}
	_, text, _ := proto.ValueText(answer[0].Value)
	return text, nil
}

// TestDevicesFollowTheLabNetwork: on the lab network a device answers at its lab address and
// at its own 127.x address; back on "this PC only" it answers at its 127.x address alone.
func TestDevicesFollowTheLabNetwork(t *testing.T) {
	executable := nativetest.Program(t, "studio_snmp.exe")
	s := New(&memoryProjects{}, executable, testutil.TempDir(t), nil, t.Logf)
	defer s.Close()
	var lab atomic.Bool
	s.SetNetwork(lab.Load)

	device, err := s.AddDevice(profiles.Switch)
	if err != nil {
		t.Fatal(err)
	}
	device.IP, device.Port = ownIP, testutil.FreeUDPPort(t, ownIP)
	if _, err := s.UpdateDevice(device, false); err != nil {
		t.Fatal(err)
	}
	lab.Store(true)
	if _, err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if state := s.Status().Devices[0]; state.State != "failed" || !strings.Contains(state.Error, "no address on the lab network") {
		t.Fatalf("a device without a lab address must not start: %+v", state)
	}

	if err := s.SetLabIPs(map[string]string{device.ID: labIP}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(); err != nil {
		t.Fatal(err)
	}
	state := s.Status().Devices[0]
	if state.State != "running" || !state.Network || !strings.HasPrefix(state.Address, labIP+":") {
		t.Fatalf("status on the lab network: %+v", state)
	}
	for _, host := range []string{labIP, ownIP} {
		if name, err := sysName(t, host, device.Port); err != nil || name == "" {
			t.Fatalf("no answer at %s: %q %v", host, name, err)
		}
	}

	lab.Store(false)
	s.NetworkChanged() // what project_changed does when Network setup is cleaned up
	eventually(t, "the move back to this PC only", func() bool {
		_, err := sysName(t, labIP, device.Port)
		return err != nil && s.Status().Devices[0].State == "running"
	})
	if name, err := sysName(t, ownIP, device.Port); err != nil || name == "" {
		t.Fatalf("no answer at its own address after the move: %v", err)
	}
	if s.Status().Devices[0].Network {
		t.Fatal("status still says lab network")
	}
}
