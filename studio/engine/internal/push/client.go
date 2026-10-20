package push

import "sync"

// client is one browser tab: its subscriptions and its outbox. The outbox holds at most one
// entry per topic; a newer frame for a topic whose last frame was not written yet turns the
// entry into "send the full frame", so a slow tab never builds a queue and never applies a
// delta to the wrong revision.
type client struct {
	id string

	mu       sync.Mutex
	topics   map[string]*instance // topic name -> subscribed instance
	outbox   []outgoing
	index    map[string]int      // topic name -> outbox position
	sent     map[*instance]int64 // newest revision handed to the stream
	wake     chan struct{}       // of the attached stream; nil while detached
	attached int64               // generation of the attached stream; 0 while detached
	streams  int64               // generations handed out
	expiry   Timer               // drops the client after Grace without a stream
}

type outgoing struct {
	in    *instance
	frame []byte // nil: the full frame of the newest revision
	rev   int64
}

func newClient(id string) *client {
	return &client{id: id, topics: map[string]*instance{}, index: map[string]int{}, sent: map[*instance]int64{}}
}

// offer queues frame (revision rev) of in; called with the area lock held, never blocks.
// A revision the stream already wrote (through a full frame resolved meanwhile) is skipped.
func (c *client) offer(in *instance, frame []byte, rev int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.topics[in.topic.Name] != in || c.sent[in] >= rev {
		return
	}
	c.put(in, frame, rev)
}

// put queues a frame (nil: the full frame); the caller holds c.mu.
func (c *client) put(in *instance, frame []byte, rev int64) {
	if at, queued := c.index[in.topic.Name]; queued {
		c.outbox[at] = outgoing{in: in}
	} else {
		c.index[in.topic.Name] = len(c.outbox)
		c.outbox = append(c.outbox, outgoing{in: in, frame: frame, rev: rev})
	}
	c.signal()
}

// wantFull queues the full frame of every subscribed topic (a (re)connect) or of the named
// ones (a resync request); the caller holds c.mu.
func (c *client) wantFull(names []string) {
	if names == nil {
		for _, in := range c.topics {
			c.put(in, nil, 0)
		}
		return
	}
	for _, name := range names {
		if in, found := c.topics[name]; found {
			c.put(in, nil, 0)
		}
	}
}

// forget drops what was sent of a released instance; the caller holds c.mu.
func (c *client) forget(in *instance) { delete(c.sent, in) }

// signal wakes the attached stream; the caller holds c.mu.
func (c *client) signal() {
	if c.wake == nil {
		return
	}
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

// drain appends the outbox frames of stream generation gen; ok is false when another
// stream took over this client. Full frames are resolved here, under the client lock, so a
// frame of the same revision offered meanwhile is recognised as sent.
func (c *client) drain(gen int64, frames [][]byte) ([][]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.attached != gen {
		return frames, false
	}
	for _, item := range c.outbox {
		frame, rev := item.frame, item.rev
		if frame == nil {
			frame, rev = item.in.fullFrame()
		}
		if frame != nil {
			frames = append(frames, frame)
			c.sent[item.in] = max(c.sent[item.in], rev)
		}
	}
	clear(c.outbox)
	c.outbox = c.outbox[:0]
	clear(c.index)
	return frames, true
}
