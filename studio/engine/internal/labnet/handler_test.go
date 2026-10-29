package labnet

import (
	"errors"
	"reflect"
	"testing"
)

const card = "{CARD}"

type fakeOps struct {
	cardErr error
	all     map[string][]string
	added   []string
	removed []string
	fail    error
	failOn  string // only this address fails, when set
	reads   int
}

func (f *fakeOps) Read() (View, error) {
	f.reads++
	if f.cardErr != nil {
		return View{}, f.cardErr
	}
	return View{Card: card, Addresses: f.all, Names: map[string]string{"{WIFI}": "Wi-Fi"}}, nil
}
func (f *fakeOps) Add(view View, ip string) error {
	if view.Card != card {
		return errors.New("wrong card")
	}
	f.added = append(f.added, ip)
	if err := f.failure(ip); err != nil {
		return err
	}
	f.all[card] = append(append([]string{}, f.all[card]...), ip)
	return nil
}
func (f *fakeOps) Remove(_ View, ip string) error {
	f.removed = append(f.removed, ip)
	if err := f.failure(ip); err != nil {
		return err
	}
	f.all[card] = without(f.all[card], ip)
	return nil
}
func (f *fakeOps) failure(ip string) error {
	if f.failOn != "" && f.failOn != ip {
		return nil
	}
	return f.fail
}

func ops() *fakeOps {
	return &fakeOps{all: map[string][]string{card: {"192.168.216.1"}, "{WIFI}": {"192.168.118.112"}}}
}

func TestAddPutsAnAddressOnTheCardOnly(t *testing.T) {
	fake := ops()
	reply := Handle(fake, Request{Op: OpAdd, IP: " 10.20.0.5 "}, "1")
	if !reply.OK || !reflect.DeepEqual(fake.added, []string{"10.20.0.5"}) {
		t.Fatalf("reply %+v added %v", reply, fake.added)
	}
	if !reflect.DeepEqual(reply.Addresses, []string{"10.20.0.5", "192.168.216.1"}) {
		t.Fatalf("addresses %v", reply.Addresses)
	}
}

func TestAddressOfAnotherCardIsRefused(t *testing.T) {
	fake := ops()
	reply := Handle(fake, Request{Op: OpAdd, IP: "192.168.118.112"}, "1")
	if reply.OK || len(fake.added) != 0 || reply.Message == "" {
		t.Fatalf("an address of Wi-Fi was accepted: %+v", reply)
	}
}

func TestAddAndRemoveAreSafeToRepeat(t *testing.T) {
	fake := ops()
	if reply := Handle(fake, Request{Op: OpAdd, IP: "192.168.216.1"}, "1"); !reply.OK || len(fake.added) != 0 {
		t.Fatalf("adding a present address called Windows: %+v", fake.added)
	}
	if reply := Handle(fake, Request{Op: OpRemove, IP: "10.9.9.9"}, "1"); !reply.OK || len(fake.removed) != 0 {
		t.Fatalf("removing a missing address called Windows: %+v", fake.removed)
	}
	if reply := Handle(fake, Request{Op: OpRemove, IP: "192.168.216.1"}, "1"); !reply.OK || len(reply.Addresses) != 0 {
		t.Fatalf("remove: %+v", reply)
	}
}

func TestUnusableAddressesAreRefused(t *testing.T) {
	for _, text := range []string{"127.0.0.5", "169.254.1.1", "224.0.0.1", "0.1.2.3", "255.255.255.255",
		"fe80::1", "300.1.1.1", "", "10.0.0.1/24"} {
		fake := ops()
		if reply := Handle(fake, Request{Op: OpAdd, IP: text}, "1"); reply.OK || len(fake.added) != 0 {
			t.Fatalf("%q was accepted", text)
		}
	}
}

func TestMissingCardChangesNothing(t *testing.T) {
	fake := ops()
	fake.cardErr = ErrNoCard
	if reply := Handle(fake, Request{Op: OpAdd, IP: "10.0.0.1"}, "1"); reply.OK || len(fake.added) != 0 {
		t.Fatalf("reply %+v", reply)
	}
}

func TestWindowsFailureIsReported(t *testing.T) {
	fake := ops()
	fake.fail = errors.New("denied")
	reply := Handle(fake, Request{Op: OpAdd, IP: "10.0.0.1"}, "1")
	if reply.OK || reply.Message != "Windows did not add 10.0.0.1: denied" {
		t.Fatalf("reply %+v", reply)
	}
}

func TestOnlyKnownRequestsDecode(t *testing.T) {
	for _, line := range []string{`{"op":"status"}`, `{"op":"add","ip":"10.0.0.1"}`,
		`{"op":"remove","ips":["10.0.0.1","10.0.0.2"]}`} {
		if _, err := DecodeRequest([]byte(line)); err != nil {
			t.Fatalf("%s: %v", line, err)
		}
	}
	for _, line := range []string{`{"op":"run","ip":"x"}`, `{"op":"add","ip":"1.2.3.4","cmd":"x"}`,
		`{"op":"status","ip":"1.2.3.4"}`, `{"op":"status","ips":["1.2.3.4"]}`, `{"op":"add","ips":[]}`,
		`{"op":"add","ip":"1.2.3.4","ips":["1.2.3.5"]}`, `not json`, ``} {
		if _, err := DecodeRequest([]byte(line)); err == nil {
			t.Fatalf("%s was accepted", line)
		}
	}
}

// ADR-0128: one request changes several addresses after one reading of Windows; each address
// gets the single-address checks and a refusal is reported for that address only.
func TestSeveralAddressesInOneRequest(t *testing.T) {
	fake := ops()
	reply := Handle(fake, Request{Op: OpAdd, IPs: []string{"10.20.0.5", "192.168.216.1", "192.168.118.112",
		"169.254.1.1", "10.20.0.6"}}, "2")
	if fake.reads != 1 || !reflect.DeepEqual(fake.added, []string{"10.20.0.5", "10.20.0.6"}) {
		t.Fatalf("reads %d added %v", fake.reads, fake.added)
	}
	if reply.OK || len(reply.Failed) != 2 || reply.Failed["192.168.118.112"] == "" || reply.Failed["169.254.1.1"] == "" {
		t.Fatalf("reply %+v", reply)
	}
	if !reflect.DeepEqual(reply.Addresses, []string{"10.20.0.5", "10.20.0.6", "192.168.216.1"}) {
		t.Fatalf("addresses %v", reply.Addresses)
	}
	reply = Handle(fake, Request{Op: OpRemove, IPs: []string{"10.20.0.5", "10.9.9.9", "10.20.0.6"}}, "2")
	if fake.reads != 2 || !reflect.DeepEqual(fake.removed, []string{"10.20.0.5", "10.20.0.6"}) {
		t.Fatalf("reads %d removed %v", fake.reads, fake.removed)
	}
	if !reply.OK || reply.Failed != nil || !reflect.DeepEqual(reply.Addresses, []string{"192.168.216.1"}) {
		t.Fatalf("remove reply %+v", reply)
	}
}

func TestOneWindowsFailureLeavesTheOtherAddresses(t *testing.T) {
	fake := ops()
	fake.fail, fake.failOn = errors.New("denied"), "10.0.0.2"
	reply := Handle(fake, Request{Op: OpAdd, IPs: []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}}, "2")
	if reply.OK || reply.Message != "Windows did not add 10.0.0.2: denied" || len(reply.Failed) != 1 {
		t.Fatalf("reply %+v", reply)
	}
	if !reflect.DeepEqual(reply.Addresses, []string{"10.0.0.1", "10.0.0.3", "192.168.216.1"}) {
		t.Fatalf("addresses %v", reply.Addresses)
	}
}

func TestBatchedFromProtocolTwo(t *testing.T) {
	if Batched("") || Batched("1") || Batched("x") || !Batched("2") || !Batched("10") {
		t.Fatal("Batched is wrong")
	}
}

func TestFreeAddressesStayInTheSlash24(t *testing.T) {
	hosts := SameSlash24("192.168.216.30")
	if len(hosts) != 235 || hosts[0] != "192.168.216.130" || hosts[len(hosts)-1] != "192.168.216.129" {
		t.Fatalf("hosts %d %v %v", len(hosts), hosts[0], hosts[len(hosts)-1])
	}
	if !InNetwork("192.168.118.200", "192.168.118.112", 20) || InNetwork("192.168.216.1", "192.168.118.112", 20) {
		t.Fatal("InNetwork is wrong")
	}
	if !IsCard("Microsoft KM-TEST Loopback Adapter") || IsCard("Realtek Gaming GbE Family Controller") {
		t.Fatal("IsCard is wrong")
	}
}
