package netaddr

import (
	"strings"
	"testing"

	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

const (
	lab = "{LAB}"
	eth = "{ETH}"
)

type fake struct {
	mode, guid string
	running    []string
	labSet     []string
	snmpSet    []string
	manual     *pyjson.Object
	applied    bool
	rowStatus  string
}

func (f *fake) service() *Service {
	return New(Deps{
		Network: func() (string, string) { return f.mode, f.guid },
		LabCard: func() string { return lab },
		Running: func() []string { return f.running },
		IEDs: func() ([]Device, error) {
			return []Device{{ID: "c04", Name: "C04_BCU", LabIP: "192.168.235.20"}}, nil
		},
		SNMP: func() []Device {
			return []Device{{ID: "s1", Name: "SESW-A", LabIP: "192.168.235.200"}, {ID: "s2", Name: "GPS-1", LabIP: "192.168.235.3"}}
		},
		LabSetIED:   func(id, ip string) error { f.labSet = append(f.labSet, id+" "+ip); return nil },
		LabCheckOwn: func(ip, owner string) (string, error) { return ip, nil },
		SetSNMPLab:  func(id, ip string) error { f.snmpSet = append(f.snmpSet, id+" "+ip); return nil },
		Preview: func(guid any, strategy string, manual any) (*pyjson.Object, error) {
			f.manual = manual.(*pyjson.Object)
			status := f.rowStatus
			row := pyjson.NewObject().Set("ied_id", "c04").Set("name", "C04_BCU").Set("status", status).
				Set("hint", "Another device already answers on this address.")
			return pyjson.NewObject().Set("rows", []any{row}).Set("ready", status == "ok").Set("fingerprint", "fp"), nil
		},
		Apply: func(fingerprint, confirm any) (*pyjson.Object, error) {
			f.applied = fingerprint == "fp" && confirm == true
			return pyjson.NewObject().Set("applied", true), nil
		},
		Adapters: func() ([]netcfg.Adapter, error) {
			return []netcfg.Adapter{{GUID: eth, Name: "Ethernet", Addresses: []netcfg.Address{{IP: "192.168.235.1", Prefix: 25}}}}, nil
		},
	})
}

func TestLabCardMovesAtOnceWithItsChecks(t *testing.T) {
	f := &fake{mode: "lab", guid: lab}
	if _, err := f.service().SetIED("c04", " 192.168.235.40 "); err != nil || f.labSet[0] != "c04 192.168.235.40" {
		t.Fatalf("ied: %v %v", err, f.labSet)
	}
	if _, err := f.service().SetSNMP("s1", "192.168.235.3"); err == nil || !strings.Contains(err.Error(), "GPS-1 uses") {
		t.Fatalf("snmp duplicate: %v", err)
	}
	if _, err := f.service().SetSNMP("s1", "192.168.235.199"); err != nil || f.snmpSet[0] != "s1 192.168.235.199" {
		t.Fatalf("snmp: %v %v", err, f.snmpSet)
	}
}

func TestRealCardReviewsThePlanWithEveryAddressKept(t *testing.T) {
	f := &fake{mode: "lab", guid: eth, rowStatus: "ok"}
	result, err := f.service().SetIED("c04", "192.168.235.21")
	if err != nil || !f.applied {
		t.Fatalf("apply: %v %v", err, f.applied)
	}
	if message, _ := result.Get("message"); message != "C04_BCU will answer at 192.168.235.21 on Ethernet from the next Start." {
		t.Fatalf("message %v", message)
	}
	changed, _ := f.manual.Get("c04")
	kept, _ := f.manual.Get(netcfg.SNMPRowPrefix + "s2") // .3 is inside the card's 192.168.235.0/25
	if ip, _ := changed.(*pyjson.Object).Get("ip"); ip != "192.168.235.21" {
		t.Fatalf("changed %v", changed)
	}
	if prefix, _ := kept.(*pyjson.Object).Get("prefix"); prefix != 25 {
		t.Fatalf("kept %v", kept) // the card's own network decides the prefix
	}

	busy := &fake{mode: "lab", guid: eth, rowStatus: "in_use_on_network"}
	if _, err := busy.service().SetIED("c04", "192.168.235.21"); err == nil || busy.applied ||
		!strings.Contains(err.Error(), "C04_BCU: Another device") {
		t.Fatalf("busy address: %v applied %v", err, busy.applied)
	}
	running := &fake{mode: "lab", guid: eth, rowStatus: "ok", running: []string{"c04"}}
	if _, err := running.service().SetSNMP("s1", "192.168.235.210"); err == nil || running.applied {
		t.Fatalf("while running: %v", err)
	}
}

func TestThisPCOnlyHasNoNetworkAddress(t *testing.T) {
	f := &fake{mode: "loopback"}
	if _, err := f.service().SetIED("c04", "192.168.235.21"); err == nil || !strings.Contains(err.Error(), "this PC only") {
		t.Fatalf("this PC only: %v", err)
	}
	if _, err := f.service().SetIED("nobody", "192.168.235.21"); err == nil {
		t.Fatal("unknown IED accepted")
	}
}
