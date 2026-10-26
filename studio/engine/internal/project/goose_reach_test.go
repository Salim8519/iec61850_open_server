package project

import (
	"testing"

	"studio-engine/internal/project/netset"
	"studio-engine/internal/pyjson"
)

// ADR-0119: "auto" and "local" never change the stored goose_reach (Python reads every file);
// "local" is the Go-only goose_local key, stored only while true.
func TestGooseReachChoicesKeepTheFilePythonReadable(t *testing.T) {
	if reach, _ := netset.DefaultNetwork().Get("goose_reach"); reach != "inside" {
		t.Fatalf("default goose_reach %v", reach)
	}
	project := pyjson.NewObject().Set("network", netset.DefaultNetwork())
	network, err := SaveReach(project, "local", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if reach, _ := network.Get("goose_reach"); reach != "inside" {
		t.Fatalf("local changed goose_reach to %v", reach)
	}
	if local, _ := network.Get(netset.GooseLocal); local != true {
		t.Fatal("local is not stored")
	}
	network, err = SaveReach(project, "auto", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, has := network.Get(netset.GooseLocal); has {
		t.Fatal("auto keeps goose_local; default projects must stay byte-identical")
	}
	project.Set("network", netset.DefaultNetwork().Set("goose_reach", "outside"))
	if network, err = SaveReach(project, "auto", "", ""); err != nil {
		t.Fatal(err)
	}
	if reach, _ := network.Get("goose_reach"); reach != "outside" {
		t.Fatalf("an earlier share choice was lost: %v", reach)
	}
	if _, err := SaveReach(project, "everywhere", "", ""); err == nil {
		t.Fatal("an unknown reach was accepted")
	}
	if netset.LabNotice(nil, pyjson.NewObject().Set("mode", "lab")) {
		t.Fatal("lab projects need no GOOSE notice any more")
	}
}

func TestThePythonTwinDropsGooseLocal(t *testing.T) {
	t.Cleanup(netset.AsPythonTwin())
	network, err := netset.ValidateNetwork(netset.DefaultNetwork().Set(netset.GooseLocal, true))
	if err != nil {
		t.Fatal(err)
	}
	if _, has := network.Get(netset.GooseLocal); has {
		t.Fatal("the twin must drop goose_local as Python does")
	}
}
