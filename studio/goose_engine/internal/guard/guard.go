// Package guard decides, frame by frame, what may reach the app and the wire.
// It holds the simulated sources (IED, MAC, gocbRefs), one outward gate per IED
// and the conflicts seen when a real device sends a simulated IED's gocbRef.
package guard

import (
	"fmt"
	"sync"
	"time"

	"studio-goose-engine/internal/frame"
)

// Gate is the outward state of one simulated IED.
type Gate string

const (
	Open    Gate = "open"    // may go outside while sharing
	Held    Gate = "held"    // conflict seen, waiting for the user's answer
	Blocked Gate = "blocked" // user chose to stop sending this IED outside
	Forced  Gate = "forced"  // user chose to continue despite the conflict
)

// Conflict records a real device sending a gocbRef that a simulated IED owns.
type Conflict struct {
	IED     string `json:"ied_id"`
	GocbRef string `json:"gocb_ref"`
	Src     string `json:"src_mac"`
}

type source struct {
	ied  string
	mac  frame.MAC
	refs map[string]bool
}

// Guard is safe for concurrent use by the bus and uplink goroutines.
type Guard struct {
	mu        sync.Mutex
	byMAC     map[frame.MAC]*source
	byRef     map[string]*source
	gates     map[string]Gate
	conflicts map[string]Conflict
	limits    *Limits
}

// New returns a guard with the per-IED outward rate cap.
func New(perSecond int) *Guard {
	return &Guard{byMAC: map[frame.MAC]*source{}, byRef: map[string]*source{}, gates: map[string]Gate{},
		conflicts: map[string]Conflict{}, limits: NewLimits(perSecond)}
}

// Register adds a simulated IED, or more gocbRefs for one already known.
func (g *Guard) Register(ied string, mac frame.MAC, refs []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	src := g.byMAC[mac]
	if src == nil {
		src = &source{ied: ied, mac: mac, refs: map[string]bool{}}
		g.byMAC[mac] = src
		g.gates[ied] = Open
	}
	for _, ref := range refs {
		src.refs[ref] = true
		g.byRef[ref] = src
	}
}

// FromApp reports whether a bus frame comes from a registered simulated IED.
func (g *Guard) FromApp(info frame.Info) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.byMAC[info.Src] != nil
}

// MaySend checks one app frame for the wire. The reason names what refused it.
func (g *Guard) MaySend(info frame.Info, now time.Time) (bool, string) {
	if err := ValidOutbound(info); err != nil {
		return false, "invalid"
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	src := g.byMAC[info.Src]
	if src == nil {
		return false, "unknown_source"
	}
	if !src.refs[info.GocbRef] {
		return false, "foreign_ref"
	}
	switch g.gates[src.ied] {
	case Held:
		return false, "held"
	case Blocked:
		return false, "blocked"
	}
	if !g.limits.Allow(src.ied, now) {
		return false, "rate"
	}
	return true, ""
}

// FromWire decides a frame heard on the adapter. A frame carrying one of our MACs is
// our own reflection. A frame carrying a simulated gocbRef is a conflict: it is never
// delivered inside, and the first one holds that IED's outward gate.
func (g *Guard) FromWire(info frame.Info) (deliver bool, reason string, fresh *Conflict) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.byMAC[info.Src] != nil {
		return false, "reflection", nil
	}
	src := g.byRef[info.GocbRef]
	if src == nil {
		return true, "", nil
	}
	if g.gates[src.ied] != Open {
		return false, "conflict", nil
	}
	g.gates[src.ied] = Held
	conflict := Conflict{IED: src.ied, GocbRef: info.GocbRef, Src: info.Src.String()}
	g.conflicts[src.ied] = conflict
	return false, "conflict", &conflict
}

// Choose applies the user's answer: "stop" blocks the IED outside, "continue" forces it.
func (g *Guard) Choose(ied, choice string) (Gate, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, known := g.gates[ied]; !known {
		return "", fmt.Errorf("IED %s is not sending GOOSE", ied)
	}
	switch choice {
	case "stop":
		g.gates[ied] = Blocked
	case "continue":
		g.gates[ied] = Forced
	default:
		return "", fmt.Errorf("choice must be stop or continue")
	}
	return g.gates[ied], nil
}

// Waiting lists conflicts the user has not answered yet.
func (g *Guard) Waiting() []Conflict {
	g.mu.Lock()
	defer g.mu.Unlock()
	var waiting []Conflict
	for ied, conflict := range g.conflicts {
		if g.gates[ied] == Held {
			waiting = append(waiting, conflict)
		}
	}
	return waiting
}

// Gates returns a copy of every IED's outward gate.
func (g *Guard) Gates() map[string]Gate {
	g.mu.Lock()
	defer g.mu.Unlock()
	copied := make(map[string]Gate, len(g.gates))
	for ied, gate := range g.gates {
		copied[ied] = gate
	}
	return copied
}
