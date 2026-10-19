package livedom

import (
	"math/big"
	"sort"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// ReadingS is READING_S: a client that read within this many seconds is "reading".
const ReadingS = 5

// ShownNodes is SHOWN_NODES: the most-read logical nodes listed per client.
const ShownNodes = 12

// MergeSummary updates clients in place from one `mms_access` summary (merge_summary) and
// drops clients that left. On malformed input it stops at the same item as Python, keeping
// the updates made before it, and returns the Python exception as a *PyError.
func MergeSummary(clients *Clients, summary []any, now float64) error {
	for _, raw := range summary {
		item, ok := raw.(*pyjson.Object)
		if !ok {
			return &PyError{"AttributeError", "'" + TypeName(raw) + "' object has no attribute 'get'"}
		}
		peerValue := pyv.Get(item, "peer")
		if !pyv.Truthy(peerValue) {
			continue
		}
		peer, isText := peerValue.(string)
		if !pyv.Truthy(pyv.GetOr(item, "connected", true)) {
			if isText {
				clients.pop(peer)
			}
			continue
		}
		if !isText {
			return &PyError{"TypeError", "argument of type '" + TypeName(peerValue) + "' is not iterable"}
		}
		if err := mergeItem(clients.setdefault(peer, now, false), item, now); err != nil {
			return err
		}
	}
	return nil
}

func mergeItem(entry *Client, item *pyjson.Object, now float64) error {
	for _, field := range []struct {
		key    string
		counts *Counts
	}{{"reads", entry.Reads}, {"writes", entry.Writes}} {
		if err := addCounts(field.counts, pyv.Get(item, field.key)); err != nil {
			return err
		}
	}
	if pyv.Truthy(pyv.Get(item, "reads")) {
		at := now
		entry.LastRead = &at
	}
	total, err := PyInt(pyv.GetOr(item, "total_reads", entry.TotalReads))
	if err != nil {
		return err
	}
	entry.TotalReads = total
	rcbs, err := pySorted(pyv.Get(item, "rcbs"))
	if err != nil {
		return err
	}
	entry.Rcbs = rcbs
	return nil
}

// addCounts is `for name, count in (value or {}).items(): counts[name] += int(count)`.
func addCounts(counts *Counts, value any) error {
	if !pyv.Truthy(value) {
		return nil
	}
	object, ok := value.(*pyjson.Object)
	if !ok {
		return &PyError{"AttributeError", "'" + TypeName(value) + "' object has no attribute 'items'"}
	}
	for _, name := range object.Keys() {
		raw, _ := object.Get(name)
		count, err := PyInt(raw)
		if err != nil {
			return err
		}
		counts.Add(name, count)
	}
	return nil
}

// pySorted is sorted(value or []) for a list, a str (its characters) or a dict (its keys).
// Lists of only strings or only numbers are supported; a mixed list fails with a TypeError
// (Python's message names the first pair its sort compares, which may differ).
func pySorted(value any) ([]any, error) {
	var items []any
	switch v := value.(type) {
	case []any:
		items = append(items, v...)
	case string:
		for _, r := range v {
			items = append(items, string(r))
		}
	case *pyjson.Object:
		items = pyv.Strings(v.Keys())
	default:
		if pyv.Truthy(value) {
			return nil, &PyError{"TypeError", "'" + TypeName(value) + "' object is not iterable"}
		}
	}
	texts, numbers := 0, 0
	for _, item := range items {
		if _, ok := item.(string); ok {
			texts++
		} else if _, ok := asNumber(item); ok {
			numbers++
		}
	}
	if texts != len(items) && numbers != len(items) {
		return nil, &PyError{"TypeError", "'<' not supported between instances of mixed types"}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if texts == len(items) {
			return items[i].(string) < items[j].(string)
		}
		left, _ := asNumber(items[i])
		right, _ := asNumber(items[j])
		return left.Cmp(right) < 0
	})
	if items == nil {
		items = []any{}
	}
	return items, nil
}

func asNumber(value any) (*big.Float, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return big.NewFloat(1), true
		}
		return big.NewFloat(0), true
	case float64:
		if v != v {
			return nil, false
		}
		return big.NewFloat(v), true
	case *big.Int:
		return new(big.Float).SetInt(v), true
	}
	if number, ok := pyv.Int64(value); ok {
		return new(big.Float).SetInt64(number), true
	}
	return nil, false
}

// MarkOwnPeer shows ScadaBench's own command client (studio_probe) as ScadaBench, never as
// an external reader (mark_own_peer).
func MarkOwnPeer(clients *Clients, peer string, now float64) {
	clients.setdefault(peer, now, true).Own = true
}

// ClientView returns the display rows, most active first (client_view).
func ClientView(clients *Clients, now float64) []any {
	type row struct {
		object *pyjson.Object
		own    bool
		total  int64
		peer   string
	}
	rows := make([]row, 0, clients.Len())
	for _, peer := range clients.peers {
		entry := clients.byPeer[peer]
		nodes := entry.Reads.Keys()
		sort.SliceStable(nodes, func(i, j int) bool {
			left, right := entry.Reads.Get(nodes[i]), entry.Reads.Get(nodes[j])
			if left != right {
				return left > right
			}
			return nodes[i] < nodes[j]
		})
		reads := []any{}
		for index, node := range nodes {
			if index == ShownNodes {
				break
			}
			reads = append(reads, pyjson.NewObject().Set("node", node).Set("count", entry.Reads.Get(node)))
		}
		reading := entry.LastRead != nil && now-*entry.LastRead <= ReadingS
		object := pyjson.NewObject().Set("peer", entry.Peer).Set("ip", entry.IP).Set("own", entry.Own).
			Set("this_pc", strings.HasPrefix(entry.IP, "127.")).Set("reading", reading).Set("reads", reads).
			Set("more_nodes", int64(max(0, len(nodes)-ShownNodes))).Set("total_reads", entry.TotalReads).
			Set("writes", entry.Writes.Sum()).Set("rcbs", append([]any{}, entry.Rcbs...)).
			Set("connected_s", pyv.RoundHalfEven(now-entry.FirstSeen))
		rows = append(rows, row{object, entry.Own, entry.TotalReads, entry.Peer})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.own != b.own {
			return !a.own
		}
		if a.total != b.total {
			return a.total > b.total
		}
		return a.peer < b.peer
	})
	result := make([]any, len(rows))
	for index, item := range rows {
		result[index] = item.object
	}
	return result
}
