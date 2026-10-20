package service

import (
	"strings"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/profiles"
	"studio-engine/internal/testutil"
	"studio-engine/internal/tools/nativetest"
	"studio-engine/internal/tools/pytext"
)

// memoryProjects keeps the SNMP section like the project service does.
type memoryProjects struct {
	mu      sync.Mutex
	section *pyjson.Object
	saves   int
}

func (p *memoryProjects) SNMP() *pyjson.Object {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.section
}

func (p *memoryProjects) SaveSNMP(section *pyjson.Object) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.section, p.saves = section, p.saves+1
	return nil
}

const (
	deviceIP   = "127.66.30.1"
	receiverIP = "127.66.30.2"
)

func eventually(t *testing.T, what string, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !check() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func valueOf(rows []any, oid string) string {
	for _, item := range rows {
		row := item.(*pyjson.Object)
		if pytext.Str(field(row, "oid")) == oid {
			return pytext.Str(field(row, "value"))
		}
	}
	return ""
}

// TestSimulateReadAndTrapThroughTheEngine drives the real studio_snmp: a switch is simulated,
// read in v3 by the reader, changed from the page, and its link-down trap is received.
func TestSimulateReadAndTrapThroughTheEngine(t *testing.T) {
	executable := nativetest.Program(t, "studio_snmp.exe")
	projects := &memoryProjects{}
	s := New(projects, executable, testutil.TempDir(t), nil, t.Logf)
	defer s.Close()

	device, err := s.AddDevice(profiles.Switch)
	if err != nil {
		t.Fatal(err)
	}
	trapPort := testutil.FreePort(t, receiverIP)
	device.IP, device.Port = deviceIP, testutil.FreePort(t, deviceIP)
	device.Traps = []model.TrapTarget{{Host: receiverIP, Port: trapPort, Version: "v3", User: defaultUser, Inform: true}}
	if _, err := s.UpdateDevice(device, false); err != nil {
		t.Fatal(err)
	}
	status, err := s.Start()
	if err != nil || status.Running != 1 {
		t.Fatalf("start: %+v, %v", status, err)
	}
	if _, err := s.Listen(TrapSettings{IP: receiverIP, Port: trapPort, Users: []model.User{DefaultUser()}}); err != nil {
		t.Fatal(err)
	}

	if _, err := s.SetPoll(200); err != nil {
		t.Fatal(err)
	}
	target := model.Target{Host: deviceIP, Port: device.Port, Version: "v3", User: DefaultUser(), TimeoutMS: 1000}
	if _, err := s.Connect(target); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the walk", func() bool { return s.Status().Reader.State == "connected" })
	values := s.Values()
	if len(values) != len(device.Points) || valueOf(values, "1.3.6.1.2.1.1.5.0") != device.Name {
		t.Fatalf("walk read %d values (want %d), sysName %q", len(values), len(device.Points),
			valueOf(values, "1.3.6.1.2.1.1.5.0"))
	}
	row := values[0].(*pyjson.Object)
	if name := pytext.Str(field(row, "name")); !strings.HasPrefix(name, "SNMPv2-MIB::sysDescr") {
		t.Fatalf("the first value is named %q", name)
	}

	if _, err := s.SetValue(device.ID, "1.3.6.1.2.1.1.5.0", "Bay 7 switch", false); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the polled change", func() bool { return valueOf(s.Values(), "1.3.6.1.2.1.1.5.0") == "Bay 7 switch" })

	results, err := s.Trigger(device.ID, "link_down:3")
	if err != nil || len(results) != 1 || !results[0].OK {
		t.Fatalf("trap: %+v, %v", results, err)
	}
	eventually(t, "the received inform", func() bool { return s.Traps.Sequence() > 0 })
	entry := s.Traps.After(0, 10)
	first := field(entry, "entries").([]any)[0].(*pyjson.Object)
	if field(first, "kind") != "inform" || !strings.HasSuffix(pytext.Str(field(first, "notification_name")), "linkDown") {
		t.Fatalf("trap log entry %v", pytext.Str(first))
	}
	if valueOf(s.Values(), "1.3.6.1.2.1.2.2.1.8.3") != "2" {
		eventually(t, "port 3 down", func() bool { return valueOf(s.Values(), "1.3.6.1.2.1.2.2.1.8.3") == "2" })
	}

	twin, err := s.SaveReadDevice("Copy of the switch")
	if err != nil || len(twin.Device.Points) != len(device.Points) || twin.Device.Profile != profiles.Walk {
		t.Fatalf("twin: %d points, %v", len(twin.Device.Points), err)
	}
	text, _, err := s.ExportCSV(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	imported, err := s.ImportDevice(profiles.CSV, "From CSV", text)
	if err != nil || len(imported.Device.Points) != len(device.Points) || len(imported.Problems) != 0 {
		t.Fatalf("CSV import: %+v, %v", imported.Problems, err)
	}

	s.Disconnect()
	s.StopListening()
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if status := s.Status(); status.Engine != "off" || status.Running != 0 {
		t.Fatalf("after stop the engine is %s with %d devices", status.Engine, status.Running)
	}
	stored := Load(projects.SNMP())
	if len(stored.Devices) != 3 || stored.AutoStart || stored.Reader.TrapsOn || stored.Reader.EveryMS != 200 {
		t.Fatalf("stored config %+v", stored)
	}
}

func TestDeviceChecksAreWorded(t *testing.T) {
	config := DefaultConfig()
	device := NewDevice(profiles.GPS, nil)
	if err := CheckDevice(device, nil); err != nil {
		t.Fatalf("a new GPS device is invalid: %v", err)
	}
	other := device
	other.ID = "other"
	if err := CheckDevice(other, []model.Device{device}); err == nil || !strings.Contains(err.Error(), "own address") {
		t.Fatalf("two devices on one address: %v", err)
	}
	broken := device
	broken.Users = []model.User{{Name: "u", Level: model.LevelAuth, Auth: "sha", AuthPassword: "short"}}
	if err := CheckDevice(broken, nil); err == nil || !strings.Contains(err.Error(), "8 characters") {
		t.Fatalf("a short password: %v", err)
	}
	if config.Reader.Target.Port != 161 || config.Reader.EveryMS != DefaultPollMS {
		t.Fatalf("defaults %+v", config.Reader)
	}
	second := NewDevice(profiles.Switch, []model.Device{device})
	if second.IP != "127.60.0.2" || second.ID == device.ID {
		t.Fatalf("the second device got %s %s", second.ID, second.IP)
	}
}
