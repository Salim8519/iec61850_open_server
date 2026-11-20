package service

import (
	"errors"
	"reflect"
	"sync/atomic"
	"testing"

	"studio-engine/internal/snmp/profiles"
	"studio-engine/internal/testutil"
	"studio-engine/internal/tools/nativetest"
)

// TestDevicesPutTheirAddressOnTheLabCard: on the Lab card a device asks for its lab address
// before its agent binds, the card keeps it while the device runs, and Stop releases it
// (change 0213). A refused address fails only that device, with the card's reason.
func TestDevicesPutTheirAddressOnTheLabCard(t *testing.T) {
	executable := nativetest.Program(t, "studio_snmp.exe")
	s := New(&memoryProjects{}, executable, testutil.TempDir(t), nil, t.Logf)
	defer s.Close()
	var lab atomic.Bool
	lab.Store(true)
	s.SetNetwork(lab.Load)
	var asked []string
	var held []bool // trial N5: the card keeps the address from before it goes on (ADR-0141)
	var refuse atomic.Bool
	released := 0
	s.SetLabCard(LabCard{
		Ensure: func(address, owner string) error {
			asked = append(asked, address+" "+owner)
			held = append(held, s.CardAddresses()[address] == owner)
			if refuse.Load() {
				return errors.New("The ScadaBench Lab card is missing")
			}
			return nil
		},
		Release: func() { released++ },
	})
	device, err := s.AddDevice(profiles.Switch)
	if err != nil {
		t.Fatal(err)
	}
	device.IP, device.Port, device.Name = ownIP, testutil.FreeUDPPort(t, ownIP), "SESW-A"
	if _, err := s.UpdateDevice(device, false); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLabIPs(map[string]string{device.ID: labIP}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if want := []string{labIP + " SNMP · SESW-A"}; !reflect.DeepEqual(asked, want) {
		t.Fatalf("asked %v, want %v", asked, want)
	}
	if !reflect.DeepEqual(held, []bool{true}) {
		t.Fatalf("a starting device's address was not kept while it went on: %v", held)
	}
	if got := s.CardAddresses(); !reflect.DeepEqual(got, map[string]string{labIP: "SNMP · SESW-A"}) {
		t.Fatalf("card addresses while running: %v", got)
	}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if len(s.CardAddresses()) != 0 || released != 1 {
		t.Fatalf("after Stop: addresses %v, releases %d", s.CardAddresses(), released)
	}

	refuse.Store(true)
	if _, err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if state := s.Status().Devices[0]; state.State != "failed" || state.Error != "The ScadaBench Lab card is missing" {
		t.Fatalf("refused address: %+v", state)
	}
	s.Stop()
}
