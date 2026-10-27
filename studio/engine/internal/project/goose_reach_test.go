package project

import (
	"testing"

	"studio-engine/internal/project/netset"
	"studio-engine/internal/pyjson"
)

// ADR-0122: goose_reach is stored as chosen ("inside" by default, "outside" once shared), exactly
// as the Python engine stores it; the 0.1.28 goose_local key is dropped when a file is read.
func TestGooseReachIsStoredAsChosen(t *testing.T) {
	if reach, _ := netset.DefaultNetwork().Get("goose_reach"); reach != "inside" {
		t.Fatalf("default goose_reach %v", reach)
	}
	project := pyjson.NewObject().Set("network", netset.DefaultNetwork())
	network, err := SaveReach(project, "outside", "", "{0123ABCD-0123-4567-89AB-0123456789AB}")
	if err != nil {
		t.Fatal(err)
	}
	if reach, _ := network.Get("goose_reach"); reach != "outside" {
		t.Fatalf("outside stored as %v", reach)
	}
	if network, err = SaveReach(project, "inside", "", ""); err != nil {
		t.Fatal(err)
	}
	if reach, _ := network.Get("goose_reach"); reach != "inside" {
		t.Fatalf("inside stored as %v", reach)
	}
	for _, old := range []string{"auto", "local", "everywhere"} {
		if _, err := SaveReach(project, old, "", ""); err == nil {
			t.Fatalf("%q was accepted", old)
		}
	}
	read, err := netset.ValidateNetwork(netset.DefaultNetwork().Set("goose_local", true))
	if err != nil {
		t.Fatal(err)
	}
	if _, has := read.Get("goose_local"); has {
		t.Fatal("goose_local from 0.1.28 must be dropped, as the Python engine does")
	}
}
