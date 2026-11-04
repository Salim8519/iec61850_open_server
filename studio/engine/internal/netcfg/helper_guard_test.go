package netcfg

import "testing"

// With the network helper off, a real card gets no address that Stop could not take off again
// (ADR-0136); an address already there, and the lab card, are left alone.
func TestHelperOffBlocksNewAddressesOnARealCard(t *testing.T) {
	ready := false
	service := &NetworkService{RealCards: func(string) bool { return ready }}
	rows := func() []Row {
		return []Row{{Name: "C04_BCU", Status: "ok"}, {Name: "own", Status: "already_on_adapter"}}
	}
	ethernet := Adapter{GUID: ethGUID, Name: "Ethernet", Kind: "ethernet"}

	off := rows()
	service.guardHelper(off, ethernet)
	if off[0].Status != StatusHelperOff || off[0].usable() || off[1].Status != "already_on_adapter" {
		t.Fatalf("helper off: %+v", off)
	}
	lab := rows()
	service.guardHelper(lab, Adapter{GUID: "{LAB}", Kind: labKind})
	if lab[0].Status != "ok" {
		t.Fatalf("lab card blocked: %+v", lab)
	}
	ready = true
	on := rows()
	service.guardHelper(on, ethernet)
	if on[0].Status != "ok" {
		t.Fatalf("helper on but blocked: %+v", on)
	}
}
