package labsetup

import (
	"strings"
	"testing"
)

const gateway = "Station gateway"

func TestAnOwnAddressGoesOnTheCard(t *testing.T) {
	p := newPC()
	address, err := p.service().AddOwn(" 10.10.0.100 ", gateway)
	if err != nil || address != "10.10.0.100" || !p.cardIPs["10.10.0.100"] {
		t.Fatalf("address %q err %v card %v", address, err, p.cardIPs)
	}
}

func TestAnOwnAddressIsRefusedWhenSomethingUsesIt(t *testing.T) {
	cases := map[string]string{
		"10.20.0.5":       "IED D60 uses",
		"192.168.118.112": "address of “Wi-Fi”",
		"10.10.0.200":     "SCADA master uses",
		"127.0.0.5":       "Not an address",
	}
	for address, want := range cases {
		p := newPC()
		p.reserved = map[string]string{"10.10.0.200": "SCADA master"}
		if _, err := p.service().AddOwn(address, gateway); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: %v, want %q", address, err, want)
		}
		if len(p.cardIPs) != 0 {
			t.Fatalf("%s changed the card: %v", address, p.cardIPs)
		}
	}
}

func TestARealDeviceOnTheWiFiBlocksAnOwnAddress(t *testing.T) {
	p := newPC()
	service := p.service()
	service.Asker = NewAsker(func(address, _ string) bool { return address == "192.168.118.50" })
	if _, err := service.AddOwn("192.168.118.50", gateway); err == nil || !strings.Contains(err.Error(), "real device on Wi-Fi") {
		t.Fatalf("real device not seen: %v", err)
	}
	if _, err := service.AddOwn("192.168.118.51", gateway); err != nil {
		t.Fatalf("a silent address was refused: %v", err)
	}
}

func TestIEDsCannotTakeAnOwnAddressAndUseKeepsIt(t *testing.T) {
	p := newPC()
	p.reserved = map[string]string{"10.10.0.100": gateway}
	p.cardIPs["10.10.0.100"] = true
	status, err := p.service().Status(map[string]string{"b": "10.10.0.100"})
	if err != nil || byID(status.Rows)["b"].Status != "duplicate" || !strings.Contains(byID(status.Rows)["b"].Hint, gateway) {
		t.Fatalf("row %+v err %v", byID(status.Rows)["b"], err)
	}
	if _, err := p.service().Use(nil); err != nil {
		t.Fatal(err)
	}
	if !p.cardIPs["10.10.0.100"] {
		t.Fatal("Use removed the gateway's address")
	}
}

func TestAnOwnAddressComesBackAfterARestartAndLeaves(t *testing.T) {
	p := newPC()
	service := p.service()
	if err := service.EnsureOwn("10.10.0.100", gateway); err != nil || !p.cardIPs["10.10.0.100"] {
		t.Fatalf("not added again: %v %v", err, p.cardIPs)
	}
	calls := len(p.calls)
	if err := service.EnsureOwn("10.10.0.100", gateway); err != nil || len(p.calls) != calls+1 {
		t.Fatalf("present address added twice: %v", p.calls)
	}
	service.RemoveOwn("10.10.0.100", gateway)
	if p.cardIPs["10.10.0.100"] {
		t.Fatal("not removed")
	}
}
