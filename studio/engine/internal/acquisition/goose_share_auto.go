package acquisition

import (
	"time"

	"studio-engine/internal/acquisition/pyv"
)

// aloneAfter is how long GOOSE must have gone out with nothing heard back before the card
// says the other PC or VM may be on another network (ADR-0119).
const aloneAfter = 10 * time.Second

// aloneClock remembers since when GOOSE has been going out; zero while it is not.
type aloneClock struct{ since time.Time }

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
