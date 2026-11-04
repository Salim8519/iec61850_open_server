package labsetup

import (
	"reflect"
	"strings"
	"testing"
)

// An SNMP device's approved Ethernet address goes on when the device starts, stays while it runs
// through IED Stop, and leaves with the device; an unapproved one is refused (ADR-0136).
func TestSNMPAddressFollowsItsDeviceOnARealCard(t *testing.T) {
	p, s, requests := onEthernet(t)
	approved := s.Windows.Approved
	s.Windows.Approved = func() map[string]map[string]int {
		all := approved()
		all[eth]["192.168.40.200"] = 24
		return all
	}
	if err := s.EnsureRealOwn(eth, "192.168.40.200", "SNMP · SESW-A"); err != nil {
		t.Fatal(err)
	}
	p.reserved = map[string]string{"192.168.40.200": "SNMP · SESW-A"} // the device runs
	s.Release()                                                      // e.g. after the IEDs stop
	if !reflect.DeepEqual(*requests, []string{"add 192.168.40.200"}) {
		t.Fatalf("requests %v", *requests)
	}
	if err := s.EnsureRealOwn(eth, "192.168.40.200", "SNMP · SESW-A"); err != nil || len(*requests) != 1 {
		t.Fatalf("already on the card: %v %v", err, *requests)
	}
	p.reserved = nil // the device stopped
	s.Release()
	if !reflect.DeepEqual(*requests, []string{"add 192.168.40.200", "remove 192.168.40.200"}) {
		t.Fatalf("requests %v", *requests)
	}
	err := s.EnsureRealOwn(eth, "192.168.40.250", "SNMP · GPS-1")
	if err == nil || !strings.Contains(err.Error(), "not approved") || len(*requests) != 2 {
		t.Fatalf("unapproved address: %v %v", err, *requests)
	}
}
