//go:build trial

package service

// Trial harness for N8 (speed and network trials, 2026-11-20; change 0221): the address of a
// running gateway is taken off the ScadaBench Lab card through the installed lab network service,
// and the time until the gateway's status says "Address gone" (and back) is measured with the
// real Windows reader. Run it with studio/tools/trials/n8_gateway_address_gone.ps1.

import (
	"os"
	"testing"
	"time"

	"studio-engine/internal/gateway"
	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
)

const trialAddress = "198.18.21.50" // the service refuses loopback, so not 127.67.21.x

func TestTrialN8AddressGone(t *testing.T) {
	if _, running := labnet.ServiceState(); !running || labnet.InstalledCard() == "" {
		t.Skip("the ScadaBench lab network service is not installed and running")
	}
	change := func(op string) {
		if reply, err := labnet.Call(labnet.Request{Op: op, IPs: []string{trialAddress}}); err != nil || !reply.OK {
			t.Fatalf("%s %s: %+v %v", op, trialAddress, reply, err)
		}
	}
	change(labnet.OpAdd)
	defer change(labnet.OpRemove)
	m := NewManager(&fakeProjects{section: gateway.EmptyGateway()}, newHub(), nil, "", nil)
	m.HasAddress = netcfg.HasAddress
	if os.Getenv("TRIAL_BEFORE") != "" {
		m.HasAddress = nil // today's code: nothing watches
	}
	process := &fakeGatewayProcess{}
	m.mu.Lock()
	m.process, m.state, m.message = process, "running", "Listening on "+trialAddress+":2404"
	m.watch = m.watchAddress(process, trialAddress, trialAddress+":2404")
	m.mu.Unlock()
	defer m.Stop(false)
	removed := time.Now()
	change(labnet.OpRemove)
	gone := waitStatus(m, true, 10*time.Second)
	t.Logf("[N8] address removed; status said gone after %v; message: %v", gone, get(m.Status(), "message"))
	change(labnet.OpAdd)
	back := waitStatus(m, false, 10*time.Second)
	t.Logf("[N8] address back; status cleared after %v; message: %v", back, get(m.Status(), "message"))
	if back < 0 { // someone else's Release may have taken it off again (Release is card-wide)
		reply, err := labnet.Call(labnet.Request{Op: labnet.OpStatus})
		has, _ := netcfg.HasAddress(trialAddress)
		t.Logf("[N8] not back: Windows has it %v; service lists %v %v", has, reply.Addresses, err)
	}
	_ = removed
	start, calls := time.Now(), 1000
	for index := 0; index < calls; index++ {
		netcfg.HasAddress(trialAddress)
	}
	t.Logf("[N8] one HasAddress read costs %v (mean of %d)", time.Since(start)/time.Duration(calls), calls)
}

// waitStatus is how long until the status shows gone (or not); -1 when it never did.
func waitStatus(m *Manager, gone bool, limit time.Duration) time.Duration {
	start := time.Now()
	for time.Since(start) < limit {
		if (get(m.Status(), "address_gone") == true) == gone {
			return time.Since(start).Round(time.Millisecond)
		}
		time.Sleep(10 * time.Millisecond)
	}
	return -1
}
