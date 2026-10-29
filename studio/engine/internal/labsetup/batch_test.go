package labsetup

import (
	"reflect"
	"testing"
)

// ADR-0128: with a protocol 2 service, Run puts every starting IED's address on the card in one
// request and Stop takes them off in one.
func TestRunAndStopSendOneRequestToANewService(t *testing.T) {
	p := newPC()
	p.version = "2"
	service := p.service()
	service.Use(nil)
	p.calls = nil
	service.Runtime.Start([]string{"a", "b"})
	service.Runtime.Stop(nil)
	want := []string{"start a,b", "status ", "add 192.168.216.1,10.20.0.5", "status ",
		"stop ", "status ", "remove 10.20.0.5,192.168.216.1"}
	if !reflect.DeepEqual(p.calls, want) {
		t.Fatalf("calls %v, want %v", p.calls, want)
	}
	if len(p.cardIPs) != 0 {
		t.Fatalf("card %v", p.cardIPs)
	}
	if status, _ := service.Status(nil); status.ServiceUpdate {
		t.Fatal("a protocol 2 service was offered an update")
	}
}

// An older service gets one request per address, as before, and the page offers Repair.
func TestAnOldServiceGetsOneAddressPerRequest(t *testing.T) {
	p := newPC()
	p.version = "1"
	service := p.service()
	if status, _ := service.Status(nil); status.ServiceUpdate {
		t.Fatal("Repair offered before the service answered once")
	}
	service.Use(nil)
	p.calls = nil
	service.Runtime.Start([]string{"a", "b"})
	want := []string{"start a,b", "status ", "add 192.168.216.1", "add 10.20.0.5", "status "}
	if !reflect.DeepEqual(p.calls, want) {
		t.Fatalf("calls %v, want %v", p.calls, want)
	}
	if status, _ := service.Status(nil); !status.ServiceUpdate {
		t.Fatal("Repair not offered for a protocol 1 service")
	}
}

// The card's own link-local address is never asked to go (the service refuses it).
func TestStopLeavesTheCardsOwnAddress(t *testing.T) {
	p := newPC()
	p.version = "2"
	p.cardIPs["169.254.71.48"] = true
	service := p.service()
	service.Use(nil)
	service.Runtime.Start([]string{"a"})
	p.calls = nil
	service.Runtime.Stop(nil)
	if !reflect.DeepEqual(p.calls, []string{"stop ", "status ", "remove 192.168.216.1"}) {
		t.Fatalf("calls %v", p.calls)
	}
	if !reflect.DeepEqual(p.cardIPs, map[string]bool{"169.254.71.48": true}) {
		t.Fatalf("card %v", p.cardIPs)
	}
}
