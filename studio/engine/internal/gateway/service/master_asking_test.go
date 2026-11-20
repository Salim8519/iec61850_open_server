package service

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
	"studio-engine/internal/tools/worker"
)

// slowDevice is studio_master toward an IEC 104 device that confirms its general interrogation
// late: it answers one request at a time, and "interrogate" holds until release (or Kill).
type slowDevice struct {
	slot    chan struct{}
	release chan struct{}
	killed  chan struct{}
	once    sync.Once
}

func newSlowDevice() *slowDevice {
	return &slowDevice{slot: make(chan struct{}, 1), release: make(chan struct{}), killed: make(chan struct{})}
}

func (d *slowDevice) Request(op string, timeout time.Duration, _ *pyjson.Object) (any, error) {
	deadline := time.After(timeout)
	select {
	case d.slot <- struct{}{}:
	case <-d.killed:
		return nil, errors.New("The worker exited")
	case <-deadline:
		return nil, errors.New("Native " + op + " request timed out")
	}
	defer func() { <-d.slot }()
	switch op {
	case "hello":
		return pyjson.NewObject().Set("capabilities", []any{"master_iec104"}), nil
	case "interrogate":
		select {
		case <-d.release:
			return pyjson.NewObject().Set("confirmed", true).Set("text", "confirmed"), nil
		case <-d.killed:
			return nil, errors.New("The worker exited")
		case <-deadline:
			return nil, errors.New("Native interrogate request timed out")
		}
	}
	return pyjson.NewObject(), nil
}

func (d *slowDevice) Alive() bool {
	select {
	case <-d.killed:
		return false
	default:
		return true
	}
}

func (d *slowDevice) Close() { d.Kill() }
func (d *slowDevice) Kill()  { d.once.Do(func() { close(d.killed) }) }

func slowMaster(t *testing.T, device *slowDevice) *Master {
	executable := filepath.Join(testutil.TempDir(t), "studio_master.exe")
	if err := os.WriteFile(executable, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	section := gateway.EmptyGateway().Set("role", "master").Set("master",
		gateway.EmptyMaster().Set("host", "127.67.26.30").Set("auto_connect", true))
	spawn := func(string, func(*pyjson.Object)) (worker.Process, error) { return device, nil }
	return NewMaster(&fakeProjects{section: section}, nil, executable, spawn)
}

// TestConnectAnswersBeforeTheInterrogationEnds (trial R7): Connect waited for the whole
// interrogation (up to 60 s); it answers once the link is open, and waitAsked (the protocol
// converter) still waits for the interrogation.
func TestConnectAnswersBeforeTheInterrogationEnds(t *testing.T) {
	device := newSlowDevice()
	m := slowMaster(t, device)
	answered := make(chan *pyjson.Object, 1)
	go func() {
		status, err := m.Connect()
		if err != nil {
			t.Error(err)
		}
		answered <- status
	}()
	select {
	case status := <-answered:
		if text(status, "state") != "connected" {
			t.Fatalf("state %v", get(status, "state"))
		}
	case <-time.After(2 * time.Second):
		close(device.release)
		t.Fatal("Connect still waits for the interrogation")
	}
	waited := make(chan struct{})
	go func() { m.waitAsked(askLimit); close(waited) }()
	select {
	case <-waited:
		t.Fatal("waitAsked returned while the interrogation runs")
	case <-time.After(200 * time.Millisecond):
	}
	close(device.release)
	select {
	case <-waited:
	case <-time.After(2 * time.Second):
		t.Fatal("waitAsked did not return after the interrogation")
	}
	if _, err := m.Disconnect(true); err != nil {
		t.Fatal(err)
	}
}

// TestDisconnectDuringInterrogationIsQuick: "stop" would wait behind the interrogation (10 s
// timeout); the worker is ended at once and no failure message is left behind.
func TestDisconnectDuringInterrogationIsQuick(t *testing.T) {
	device := newSlowDevice()
	m := slowMaster(t, device)
	go func() { _, _ = m.Connect() }()
	deadline := time.Now().Add(2 * time.Second)
	for {
		m.mu.Lock()
		busy := m.asking > 0 && m.process != nil
		m.mu.Unlock()
		if busy || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond) // the interrogate request is on its way
	began := time.Now()
	status, err := m.Disconnect(true)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(began); elapsed > 2*time.Second {
		t.Fatalf("Disconnect took %v during the interrogation", elapsed)
	}
	m.waitAsked(askLimit)
	if final := m.Status(); text(final, "state") != "disconnected" || text(final, "message") != "" {
		t.Fatalf("after Disconnect: %v / %v (answer %v)", get(final, "state"), get(final, "message"), get(status, "state"))
	}
}
