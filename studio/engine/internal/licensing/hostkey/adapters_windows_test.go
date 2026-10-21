//go:build windows

package hostkey

import "testing"

// The real adapter list of the PC that runs the tests (GetAdaptersAddresses, no PowerShell): whatever
// virtual switches it has, no adapter that carries a gateway or is not a virtual switch is ever served,
// and a VM on it would ask only addresses of this PC's own networks.
func TestTheRealAdapterListOfThisPCIsReadAndOnlyVirtualSwitchesWithoutAGatewayAreServed(t *testing.T) {
	adapters, err := SystemAdapters()
	if err != nil {
		t.Fatalf("the adapter list could not be read: %v", err)
	}
	if len(adapters) == 0 {
		t.Skip("this PC reports no IPv4 adapter")
	}
	served := map[string]bool{}
	for _, listen := range ListenAddresses(adapters) {
		served[listen.IP.String()] = true
		if !listen.Subnet.Contains(listen.IP) {
			t.Errorf("%v is not inside its own subnet %v", listen.IP, listen.Subnet)
		}
	}
	for _, adapter := range adapters {
		t.Logf("adapter %q (%s): up=%v virtual-switch=%v gateways=%v addresses=%v",
			adapter.Name, adapter.Description, adapter.Up, IsVirtualSwitch(adapter), adapter.Gateways, adapter.IPv4)
		for _, address := range adapter.IPv4 {
			if !served[address.IP.String()] {
				continue
			}
			if !adapter.Up || len(adapter.Gateways) > 0 || !IsVirtualSwitch(adapter) {
				t.Errorf("%q is served although it is down, carries a gateway or is not a virtual switch", adapter.Name)
			}
		}
	}
	for _, host := range CandidateHosts(adapters) {
		if served[host] {
			t.Errorf("a VM on this PC would ask its own address %s", host)
		}
	}
}
