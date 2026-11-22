package project

import (
	"strings"
	"testing"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// twoIEDs is a lab-mode project: A simulated on 172.26.0.20, B monitored at 172.26.0.20.
func twoIEDs(t *testing.T) *pyjson.Object {
	t.Helper()
	value, err := pyjson.Decode([]byte(`{"network": {"mode": "lab", "adapter_guid": "g", "adapter_label": "Lab"},
	 "ieds": [
	  {"id": "a", "name": "A", "ip": "127.50.0.2", "port": 102, "role": "simulated", "lab_ip": "172.26.0.20", "goose_publish": true, "test_mode": false,
	   "live": {"ip": "", "port": 102, "access": "auto_reports", "poll_ms": 1000}, "scl_address": {"ip": "192.168.1.5"}},
	  {"id": "b", "name": "B", "ip": "127.50.0.3", "port": 102, "role": "real", "lab_ip": "", "goose_publish": true, "test_mode": false,
	   "live": {"ip": "172.26.0.20", "port": 102, "access": "auto_reports", "poll_ms": 1000}, "scl_address": {"ip": "192.168.1.6"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	return value.(*pyjson.Object)
}

func change(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	value, err := pyjson.Decode([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return value.(*pyjson.Object)
}

func TestSimulatedAndMonitoredMayShareAnAddress(t *testing.T) {
	project := twoIEDs(t)
	if _, err := UpdateIED(project, "b", change(t, `{"live": {"port": 102}}`)); err != nil {
		t.Fatalf("simulated vs monitored refused: %v", err)
	}
	// Two monitored relays on one address and port still clash.
	if _, err := UpdateIED(project, "a", change(t, `{"role": "real", "live": {"ip": "172.26.0.20"}}`)); err == nil ||
		!strings.Contains(err.Error(), "both use 172.26.0.20:102") {
		t.Fatalf("monitored vs monitored not refused: %v", err)
	}
}

func TestRoleSwitchStartsFromTheTypedAddress(t *testing.T) {
	project := twoIEDs(t)
	project.Set("ieds", []any{pyval.Lookup(project, "ieds", nil).([]any)[0]}) // A alone
	result, err := UpdateIED(project, "a", change(t, `{"role": "real"}`))
	if err != nil {
		t.Fatal(err)
	}
	if ip := pyval.Lookup(liveOf(result), "ip", nil); ip != "172.26.0.20" {
		t.Fatalf("monitoring did not start from the simulation address: %v", ip)
	}
	if lab := pyval.Lookup(result, "lab_ip", nil); lab != "172.26.0.20" {
		t.Fatalf("simulation address changed: %v", lab)
	}
	// Back to simulated and to real again: both addresses stay as they are.
	if _, err := UpdateIED(project, "a", change(t, `{"live": {"ip": "10.0.0.7"}}`)); err != nil {
		t.Fatal(err)
	}
	result, _ = UpdateIED(project, "a", change(t, `{"role": "simulated"}`))
	if lab := pyval.Lookup(result, "lab_ip", nil); lab != "172.26.0.20" {
		t.Fatalf("simulation address overwritten: %v", lab)
	}
	result, _ = UpdateIED(project, "a", change(t, `{"role": "real"}`))
	if ip := pyval.Lookup(liveOf(result), "ip", nil); ip != "10.0.0.7" {
		t.Fatalf("monitoring address overwritten: %v", ip)
	}
}

func TestMonitoredRelayIntoSimulatedSeedsTheLabAddress(t *testing.T) {
	project := twoIEDs(t)
	project.Set("ieds", []any{pyval.Lookup(project, "ieds", nil).([]any)[1]}) // B alone
	result, err := UpdateIED(project, "b", change(t, `{"role": "simulated"}`))
	if err != nil {
		t.Fatal(err)
	}
	if lab := pyval.Lookup(result, "lab_ip", nil); lab != "172.26.0.20" {
		t.Fatalf("simulation did not start from the monitoring address: %v", lab)
	}
}

func TestMonitoringNetworkCard(t *testing.T) {
	project := twoIEDs(t)
	result, err := UpdateIED(project, "b", change(t, `{"live": {"local": "172.26.0.5"}}`))
	if err != nil || pyval.Lookup(liveOf(result), "local", nil) != "172.26.0.5" {
		t.Fatalf("card not stored: %v %v", result, err)
	}
	if _, err := UpdateIED(project, "b", change(t, `{"live": {"local": "not-an-ip"}}`)); err == nil {
		t.Fatal("invalid card address accepted")
	}
	result, _ = UpdateIED(project, "b", change(t, `{"live": {"local": ""}}`))
	if pyval.Has(pyval.Lookup(result, "live", nil).(*pyjson.Object), "local") {
		t.Fatal("'Windows chooses' should store no card")
	}
}

func liveOf(result *pyjson.Object) *pyjson.Object {
	live, _ := pyval.Lookup(result, "live", nil).(*pyjson.Object)
	return live
}
