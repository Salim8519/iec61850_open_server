package livedom

import (
	"strings"

	"studio-engine/internal/pyjson"
)

// Counts is an insertion-ordered {name: count} dict (reads per logical node, writes per
// control) of one MMS client.
type Counts struct {
	keys   []string
	values map[string]int64
}

func newCounts() *Counts { return &Counts{values: map[string]int64{}} }

// Add is counts[name] = counts.get(name, 0) + count.
func (c *Counts) Add(name string, count int64) {
	if _, found := c.values[name]; !found {
		c.keys = append(c.keys, name)
	}
	c.values[name] += count
}

// Keys returns the names in insertion order.
func (c *Counts) Keys() []string { return append([]string(nil), c.keys...) }

// Get returns the count of name (0 when missing).
func (c *Counts) Get(name string) int64 { return c.values[name] }

// Sum is sum(counts.values()).
func (c *Counts) Sum() int64 {
	var total int64
	for _, value := range c.values {
		total += value
	}
	return total
}

func (c *Counts) object() *pyjson.Object {
	object := pyjson.NewObject()
	for _, key := range c.keys {
		object.Set(key, c.values[key])
	}
	return object
}

// Client is one entry of the per-IED client table: the Python dict
// {"peer", "ip", "first_seen", "last_read", "reads", "writes", "total_reads", "rcbs", "own"}.
type Client struct {
	Peer       string
	IP         string
	FirstSeen  float64
	LastRead   *float64 // nil is Python None: the client never read anything yet
	Reads      *Counts
	Writes     *Counts
	TotalReads int64
	Rcbs       []any
	Own        bool
}

// Clients is the {peer: client} dict of one simulated IED. It keeps Python dict order: a new
// peer goes to the end, a peer that left and came back goes to the end again.
type Clients struct {
	peers  []string
	byPeer map[string]*Client
}

// NewClients returns an empty client table.
func NewClients() *Clients { return &Clients{byPeer: map[string]*Client{}} }

// Len is len(clients).
func (c *Clients) Len() int { return len(c.peers) }

// Get returns the client of peer, or nil.
func (c *Clients) Get(peer string) *Client { return c.byPeer[peer] }

// Peers returns the peers in dict order.
func (c *Clients) Peers() []string { return append([]string(nil), c.peers...) }

// pop is clients.pop(peer, None).
func (c *Clients) pop(peer string) {
	if _, found := c.byPeer[peer]; !found {
		return
	}
	delete(c.byPeer, peer)
	for index, item := range c.peers {
		if item == peer {
			c.peers = append(c.peers[:index], c.peers[index+1:]...)
			return
		}
	}
}

// setdefault returns the client of peer, adding a fresh one first seen at now when missing.
func (c *Clients) setdefault(peer string, now float64, own bool) *Client {
	if entry, found := c.byPeer[peer]; found {
		return entry
	}
	entry := &Client{Peer: peer, IP: peerIP(peer), FirstSeen: now, Reads: newCounts(), Writes: newCounts(),
		Rcbs: []any{}, Own: own}
	c.byPeer[peer] = entry
	c.peers = append(c.peers, peer)
	return entry
}

// peerIP is _ip: the address without the port after the last colon.
func peerIP(peer string) string {
	if index := strings.LastIndex(peer, ":"); index >= 0 {
		return peer[:index]
	}
	return peer
}

// Object renders the table as the Python dict (json.dumps order), for logs and tests.
func (c *Clients) Object() *pyjson.Object {
	result := pyjson.NewObject()
	for _, peer := range c.peers {
		entry := c.byPeer[peer]
		var lastRead any
		if entry.LastRead != nil {
			lastRead = *entry.LastRead
		}
		result.Set(peer, pyjson.NewObject().Set("peer", entry.Peer).Set("ip", entry.IP).
			Set("first_seen", entry.FirstSeen).Set("last_read", lastRead).Set("reads", entry.Reads.object()).
			Set("writes", entry.Writes.object()).Set("total_reads", entry.TotalReads).
			Set("rcbs", append([]any{}, entry.Rcbs...)).Set("own", entry.Own))
	}
	return result
}
