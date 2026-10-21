package acquisition

import (
	"sync"

	"studio-engine/internal/acquisition/livedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// OwnCommandSeconds is how long ScadaBench's own MMS control is recognised (mms_access.py:7).
const OwnCommandSeconds = 20

// MmsAccess keeps the MMS client list of every simulated IED during a Run (ADR-0066;
// simulation/mms_access.py).
type MmsAccess struct {
	bus   Publisher
	clock Clock

	mu      sync.Mutex
	order   []string
	clients map[string]*livedom.Clients
	own     map[PointKey]float64
}

// NewMmsAccess returns an empty list.
func NewMmsAccess(bus Publisher, clock Clock) *MmsAccess {
	return &MmsAccess{bus: bus, clock: clock, clients: map[string]*livedom.Clients{}, own: map[PointKey]float64{}}
}

// Reset forgets every client (Run start).
func (a *MmsAccess) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.order, a.clients, a.own = nil, map[string]*livedom.Clients{}, map[PointKey]float64{}
}

// Forget drops the client lists of IEDs that stopped while the run continues.
func (a *MmsAccess) Forget(iedIDs []string) {
	drop := map[string]bool{}
	for _, id := range iedIDs {
		drop[id] = true
	}
	a.mu.Lock()
	kept := a.order[:0:0]
	for _, id := range a.order {
		if drop[id] {
			delete(a.clients, id)
		} else {
			kept = append(kept, id)
		}
	}
	a.order = kept
	for key := range a.own {
		if drop[key.IED] {
			delete(a.own, key)
		}
	}
	a.mu.Unlock()
	for _, id := range iedIDs {
		a.bus.Publish("mms_access", pyv.O("ied_id", id, "clients", []any{}))
	}
}

// MarkOwn records that ScadaBench itself is about to operate ref on the IED.
func (a *MmsAccess) MarkOwn(iedID, ref string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.own[PointKey{IED: iedID, Ref: ref}] = a.clock.Monotonic() + OwnCommandSeconds
}

func (a *MmsAccess) tableLocked(iedID string) *livedom.Clients {
	table := a.clients[iedID]
	if table == nil {
		table = livedom.NewClients()
		a.clients[iedID] = table
		a.order = append(a.order, iedID)
	}
	return table
}

// Control reports whether an MMS control came from another client, not from ScadaBench.
func (a *MmsAccess) Control(iedID string, event *pyjson.Object) bool {
	peer, _ := pyv.Get(event, "peer").(string)
	ref := pyv.PyStr(pyv.Get(event, "ref"))
	now := a.clock.Monotonic()
	a.mu.Lock()
	defer a.mu.Unlock()
	key := PointKey{IED: iedID, Ref: ref}
	deadline, found := a.own[key]
	delete(a.own, key)
	own := found && deadline > now
	if own && peer != "" {
		livedom.MarkOwnPeer(a.tableLocked(iedID), peer, now)
	}
	return peer != "" && !own
}

// Update merges one worker summary and publishes the client list (bus only).
func (a *MmsAccess) Update(iedID string, event *pyjson.Object) {
	now := a.clock.Monotonic()
	a.mu.Lock()
	table := a.tableLocked(iedID)
	if err := livedom.MergeSummary(table, pyv.GetList(event, "clients"), now); err != nil {
		a.mu.Unlock() // Python's exception left the event consumer before publishing
		return
	}
	rows := livedom.ClientView(table, now)
	a.mu.Unlock()
	a.bus.Publish("mms_access", pyv.O("ied_id", iedID, "clients", rows))
}

// Snapshot is {ied_id: client rows} for GET /mms/access.
func (a *MmsAccess) Snapshot() *pyjson.Object {
	now := a.clock.Monotonic()
	a.mu.Lock()
	defer a.mu.Unlock()
	result := pyjson.NewObject()
	for _, id := range a.order {
		result.Set(id, livedom.ClientView(a.clients[id], now))
	}
	return result
}
