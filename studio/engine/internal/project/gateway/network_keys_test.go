package gateway

import (
	"testing"

	"studio-engine/internal/pyjson"
)

// The Network choices survive a project load (ADR-0123, ADR-0124); a project without them gains
// no key, and a damaged address stops the load instead of being dropped.
func TestNetworkAddressesSurviveALoad(t *testing.T) {
	stored := pyjson.NewObject().Set("listen", "network").Set("listen_address", "192.168.1.10").
		Set("allowed_scada", "192.168.1.50").Set("master", pyjson.NewObject().Set("local", "192.168.1.10"))
	result, err := NormalizeGateway(stored)
	if err != nil {
		t.Fatal(err)
	}
	master, _ := result.Get("master")
	local, _ := master.(*pyjson.Object).Get("local")
	address, _ := result.Get("listen_address")
	allowed, _ := result.Get("allowed_scada")
	if local != "192.168.1.10" || address != "192.168.1.10" || allowed != "192.168.1.50" {
		t.Fatalf("lost: %v", result)
	}
	plain, _ := NormalizeGateway(pyjson.NewObject())
	if _, found := plain.Get("listen_address"); found {
		t.Fatal("a project without the keys gains none")
	}
	if _, err := NormalizeGateway(pyjson.NewObject().Set("allowed_scada", "scada")); err == nil {
		t.Fatal("a damaged address stops the load")
	}
}
