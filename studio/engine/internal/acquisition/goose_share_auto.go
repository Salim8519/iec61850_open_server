package acquisition

import (
	"strings"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// aloneAfter is how long GOOSE must have gone out with nothing heard back before the card
// says the other PC or VM may be on another network (ADR-0119).
const aloneAfter = 10 * time.Second

// aloneClock remembers since when GOOSE has been going out; zero while it is not.
type aloneClock struct{ since time.Time }

// applyAuto saves "auto" or "local" (ADR-0119; SaveReach maps them onto the stored keys).
// confirm:true with the adapter shown records that the user accepts sending next to real
// devices there; other changes keep that answer.
func (g *GooseShare) applyAuto(body *pyjson.Object) (*pyjson.Object, error) {
	network := pyv.Shallow(pyv.GetObj(g.deps.Projects.Current(), "network"))
	reach := pyv.GetOr(body, "reach", "auto")
	network.Set("goose_local", pyv.Equal(reach, "local"))
	if uplinkGUID := pyv.Get(body, "uplink_guid"); uplinkGUID != nil {
		network.Set("goose_uplink_guid", strings.ToUpper(pyv.PyStr(uplinkGUID)))
	}
	shareGUID := pyv.PyStr(pyv.GetOr(network, "goose_share_guid", ""))
	if confirm, ok := pyv.Get(body, "confirm").(bool); ok && confirm {
		_, uplink := g.resolve(network)
		adapter := pyv.Get(body, "adapter_guid")
		if !pyv.Truthy(adapter) {
			adapter = ""
		}
		if !pyv.Truthy(pyv.Get(uplink, "guid")) || strings.ToUpper(pyv.PyStr(adapter)) != pyv.GetStr(uplink, "guid") {
			return nil, apperr.New(ShareMoved)
		}
		shareGUID = pyv.GetStr(uplink, "guid")
	}
	return g.save(reach, pyv.PyStr(pyv.GetOr(network, "goose_uplink_guid", "")), shareGUID)
}

// nobodyThere reports whether GOOSE has gone out for aloneAfter, nothing came back from
// the network and a GOOSE the project expects is still waiting.
func (g *GooseShare) nobodyThere(state string) (bool, error) {
	g.mu.Lock()
	if state != "sharing" {
		g.alone.since = time.Time{}
		g.mu.Unlock()
		return false, nil
	}
	if g.alone.since.IsZero() {
		g.alone.since = time.Now()
	}
	long := time.Since(g.alone.since) >= aloneAfter
	g.mu.Unlock()
	if !long || g.Waiting == nil || !g.Waiting() {
		return false, nil
	}
	stats, err := g.engine.Stats()
	if err != nil {
		return false, err
	}
	return !pyv.Truthy(pyv.Get(stats, "delivered_wire")), nil
}
