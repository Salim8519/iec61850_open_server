package runtime

import (
	"math"
	"sort"
	"strings"
	"sync"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// MMS access per simulated IED (simulation/mms_access.py, domain/mms_access.py, ADR-0066).
const (
	readingSeconds = 5
	shownNodes     = 12
)

type accessClient struct {
	peer, ip      string
	firstSeen     float64
	lastRead      *float64
	reads, writes map[string]int64
	readOrder     []string
	totalReads    int64
	rcbs          []any
	own           bool
}

type mmsAccess struct {
	c       *Coordinator
	mu      sync.Mutex
	clients map[string]map[string]*accessClient
	order   map[string][]string
	ownered map[bindingKey]float64
}

func newMmsAccess(c *Coordinator) *mmsAccess {
	a := &mmsAccess{c: c}
	a.reset()
	return a
}

func (a *mmsAccess) reset() {
	a.mu.Lock()
	a.clients, a.order, a.ownered = map[string]map[string]*accessClient{}, map[string][]string{}, map[bindingKey]float64{}
	a.mu.Unlock()
}

func (a *mmsAccess) forget(ids []string) {
	a.mu.Lock()
	for _, id := range ids {
		delete(a.clients, id)
		delete(a.order, id)
		for key := range a.ownered {
			if key.ied == id {
				delete(a.ownered, key)
			}
		}
	}
	a.mu.Unlock()
	for _, id := range ids {
		a.c.publish("mms_access", pyval.Obj("ied_id", id, "clients", []any{}))
	}
}

// markOwn records ScadaBench's own command so its control_applied is not "external".
func (a *mmsAccess) markOwn(iedID, controlRef string) {
	a.mu.Lock()
	a.ownered[bindingKey{iedID, controlRef}] = a.c.now() + ownCommandSeconds
	a.mu.Unlock()
}

func ipOf(peer string) string {
	if i := strings.LastIndex(peer, ":"); i >= 0 {
		return peer[:i]
	}
	return peer
}

func (a *mmsAccess) entry(iedID, peer string, now float64, own bool) *accessClient {
	if a.clients[iedID] == nil {
		a.clients[iedID] = map[string]*accessClient{}
	}
	client := a.clients[iedID][peer]
	if client == nil {
		client = &accessClient{peer: peer, ip: ipOf(peer), firstSeen: now, reads: map[string]int64{},
			writes: map[string]int64{}, rcbs: []any{}, own: own}
		a.clients[iedID][peer] = client
		a.order[iedID] = append(a.order[iedID], peer)
	}
	return client
}

// control is MmsAccess.control: true when another client sent the control.
func (a *mmsAccess) control(iedID string, event *pyjson.Object) bool {
	peer, _ := pyval.Get(event, "peer").(string)
	ref, _ := pyval.Get(event, "ref").(string)
	now := a.c.now()
	a.mu.Lock()
	defer a.mu.Unlock()
	key := bindingKey{iedID, ref}
	until, marked := a.ownered[key]
	delete(a.ownered, key)
	own := marked && until > now
	if own && peer != "" {
		a.entry(iedID, peer, now, true).own = true
	}
	return peer != "" && !own
}

func (a *mmsAccess) drop(iedID, peer string) {
	delete(a.clients[iedID], peer)
	order := a.order[iedID][:0]
	for _, item := range a.order[iedID] {
		if item != peer {
			order = append(order, item)
		}
	}
	a.order[iedID] = order
}

// update merges one worker summary and publishes the client list (bus only).
func (a *mmsAccess) update(iedID string, event *pyjson.Object) {
	now := a.c.now()
	a.mu.Lock()
	if a.clients[iedID] == nil {
		a.clients[iedID] = map[string]*accessClient{}
	}
	for _, raw := range pyval.List(event, "clients") {
		summary, _ := raw.(*pyjson.Object)
		peer, _ := pyval.Get(summary, "peer").(string)
		if peer == "" {
			continue
		}
		if connected, present := summary.Get("connected"); present && !pyval.Truthy(connected) {
			a.drop(iedID, peer)
			continue
		}
		client := a.entry(iedID, peer, now, false)
		reads := pyval.Map(summary, "reads")
		for _, node := range keysOf(reads) {
			if _, seen := client.reads[node]; !seen {
				client.readOrder = append(client.readOrder, node)
			}
			count, _ := pyval.ToInt(pyval.Get(reads, node))
			client.reads[node] += asInt(count)
		}
		writes := pyval.Map(summary, "writes")
		for _, name := range keysOf(writes) {
			count, _ := pyval.ToInt(pyval.Get(writes, name))
			client.writes[name] += asInt(count)
		}
		if reads != nil && reads.Len() > 0 {
			stamp := now
			client.lastRead = &stamp
		}
		if total, present := summary.Get("total_reads"); present {
			value, _ := pyval.ToInt(total)
			client.totalReads = asInt(value)
		}
		rcbs := append([]any(nil), pyval.List(summary, "rcbs")...)
		sort.SliceStable(rcbs, func(i, j int) bool { return pyval.Repr(rcbs[i]) < pyval.Repr(rcbs[j]) })
		if rcbs == nil {
			rcbs = []any{}
		}
		client.rcbs = rcbs
	}
	rows := a.view(iedID, now)
	a.mu.Unlock()
	a.c.publish("mms_access", pyval.Obj("ied_id", iedID, "clients", rows))
}

func keysOf(object *pyjson.Object) []string {
	if object == nil {
		return nil
	}
	return object.Keys()
}

func asInt(value any) int64 { number, _ := value.(int64); return number }

// view is client_view: most active first.
func (a *mmsAccess) view(iedID string, now float64) []any {
	type row struct {
		object *pyjson.Object
		own    bool
		total  int64
		peer   string
	}
	var rows []row
	for _, peer := range a.order[iedID] {
		client := a.clients[iedID][peer]
		nodes := append([]string(nil), client.readOrder...)
		sort.SliceStable(nodes, func(i, j int) bool {
			if client.reads[nodes[i]] != client.reads[nodes[j]] {
				return client.reads[nodes[i]] > client.reads[nodes[j]]
			}
			return nodes[i] < nodes[j]
		})
		shown := []any{}
		for i, node := range nodes {
			if i < shownNodes {
				shown = append(shown, pyval.Obj("node", node, "count", client.reads[node]))
			}
		}
		writes := int64(0)
		for _, count := range client.writes {
			writes += count
		}
		reading := client.lastRead != nil && now-*client.lastRead <= readingSeconds
		more := int64(len(nodes) - shownNodes)
		if more < 0 {
			more = 0
		}
		object := pyval.Obj("peer", client.peer, "ip", client.ip, "own", client.own,
			"this_pc", strings.HasPrefix(client.ip, "127."), "reading", reading, "reads", shown, "more_nodes", more,
			"total_reads", client.totalReads, "writes", writes, "rcbs", client.rcbs,
			"connected_s", int64(math.RoundToEven(now-client.firstSeen)))
		rows = append(rows, row{object, client.own, client.totalReads, client.peer})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].own != rows[j].own {
			return !rows[i].own
		}
		if rows[i].total != rows[j].total {
			return rows[i].total > rows[j].total
		}
		return rows[i].peer < rows[j].peer
	})
	items := make([]any, len(rows))
	for i, entry := range rows {
		items[i] = entry.object
	}
	return items
}

// MmsAccess is the client list of every simulated IED ({ied_id: rows}) for GET /mms/access.
func (c *Coordinator) MmsAccess() *pyjson.Object {
	now := c.now()
	a := c.access
	a.mu.Lock()
	defer a.mu.Unlock()
	result := pyjson.NewObject()
	for _, id := range sortedKeys(a.clients) {
		result.Set(id, a.view(id, now))
	}
	return result
}
