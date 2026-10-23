package sld

import (
	"testing"

	"studio-engine/internal/scl"
)

// Change 0185 item 15: an IED that can never be loaded says why instead of "not loaded".
func TestBindSaysWhyAnIedCannotBeLoaded(t *testing.T) {
	lnodes := []scl.LNode{{IEDName: "BAD", LDInst: "LD0", LNClass: "XCBR", LNInst: "1"},
		{IEDName: "LATER", LDInst: "LD0", LNClass: "CSWI", LNInst: "1"}}
	why := map[string]string{"BAD": "it has no server model, so it cannot be simulated"}
	_, _, reasons := Bind("CB", lnodes, nil, map[[2]string]string{}, why)
	if len(reasons) < 2 || reasons[0] != "IED BAD is in the SCD but cannot be loaded: it has no server model, so it cannot be simulated" ||
		reasons[1] != "IED LATER is in the SCD but not loaded in this project" {
		t.Fatalf("reasons %q", reasons)
	}
}
