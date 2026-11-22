package project

import (
	"net/netip"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// seedAddress keeps the user's address across a role switch (change 0233, owner 2026-11-22:
// "it supposed to suggest upon the set ip we did"). Simulation (lab_ip) and monitoring
// (live.ip) stay two separate addresses; only an empty one starts from the other, once, so
// switching never falls back to the CID address while the user has typed one.
func seedAddress(merged *pyjson.Object, before any) {
	role := pyval.Lookup(merged, "role", "simulated")
	if pyval.Equal(role, before) {
		return
	}
	labIP, _ := pyval.Lookup(merged, "lab_ip", "").(string)
	live, isDict := pyval.Lookup(merged, "live", nil).(*pyjson.Object)
	liveIP := ""
	if isDict {
		liveIP, _ = pyval.Lookup(live, "ip", "").(string)
	}
	switch role {
	case "real":
		if liveIP == "" && labIP != "" {
			next := pyjson.NewObject()
			if isDict {
				next = pyval.CopyShallow(live)
			}
			merged.Set("live", next.Set("ip", labIP))
		}
	case "simulated":
		if address, err := netip.ParseAddr(liveIP); labIP == "" && err == nil && address.Is4() && !address.IsLoopback() {
			merged.Set("lab_ip", liveIP)
		}
	}
}
