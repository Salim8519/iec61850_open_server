package push

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// want is one requested subscription.
type want struct {
	topic  *Topic
	params *pyjson.Object
}

var errTooMany = apperr.RuntimeError("Too many app windows are open; close one and reload")

// clientLocked returns the client id, creating a detached one (with its grace timer) when
// create is set; the caller holds a.mu.
func (a *Area) clientLocked(id string) (*client, error) {
	if c, found := a.byID[id]; found {
		return c, nil
	}
	if len(a.byID) >= a.clients {
		return nil, errTooMany
	}
	c := newClient(id)
	a.byID[id] = c
	c.expiry = a.clock.AfterFunc(a.grace, func() { a.expire(c, 0) })
	return c, nil
}

// attach makes a new stream the client's only writer and queues the full frame of every
// subscribed topic. An older stream of the same tab is woken and ends.
func (a *Area) attach(id string) (*client, int64, chan struct{}, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	c, err := a.clientLocked(id)
	if err != nil {
		return nil, 0, nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.signal() // the previous stream, if any, sees the new generation and ends
	if c.expiry != nil {
		c.expiry.Stop()
		c.expiry = nil
	}
	c.streams++
	c.attached = c.streams
	c.wake = make(chan struct{}, 1)
	c.wantFull(nil)
	for _, in := range c.topics {
		in.throttle.mark() // fresh values for a returning tab; unchanged bytes are not sent
	}
	return c, c.attached, c.wake, nil
}

// detach ends stream gen; the subscriptions stay for Grace.
func (a *Area) detach(c *client, gen int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.attached != gen {
		return
	}
	c.attached, c.wake = 0, nil
	if a.byID[c.id] == c {
		streams := c.streams
		c.expiry = a.clock.AfterFunc(a.grace, func() { a.expire(c, streams) })
	}
}

// expire drops a client whose stream stayed away for Grace since stream number streams
// ended (a later stream makes the timer stale).
func (a *Area) expire(c *client, streams int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	c.mu.Lock()
	idle := c.attached == 0 && c.streams == streams
	c.mu.Unlock()
	if idle && a.byID[c.id] == c {
		a.dropLocked(c)
	}
}

// dropLocked releases every subscription of c and forgets it; the caller holds a.mu.
func (a *Area) dropLocked(c *client) {
	c.mu.Lock()
	if c.expiry != nil {
		c.expiry.Stop()
		c.expiry = nil
	}
	c.signal()
	c.attached, c.wake = 0, nil
	for name, in := range c.topics {
		delete(c.topics, name)
		a.releaseLocked(in, c)
	}
	c.mu.Unlock()
	delete(a.byID, c.id)
}

// subscribe replaces the subscriptions of client id. New or changed topics and the topics
// named in resync get a full frame; topics left out are released (an empty list stops all
// work for a hidden tab).
func (a *Area) subscribe(id string, wanted []want, resync []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	c, err := a.clientLocked(id)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	keep := map[string]*instance{}
	for _, item := range wanted {
		query := item.topic.query(item.params)
		current := c.topics[item.topic.Name]
		if current != nil && current.query == query {
			keep[item.topic.Name] = current
			continue
		}
		in := a.acquireLocked(item.topic, query, c)
		keep[item.topic.Name] = in
		if current != nil {
			a.releaseLocked(current, c)
		}
		c.topics[item.topic.Name] = in
		c.put(in, nil, 0)
	}
	for name, in := range c.topics {
		if keep[name] == nil {
			delete(c.topics, name)
			a.releaseLocked(in, c)
			if at, queued := c.index[name]; queued {
				c.outbox[at].frame, c.outbox[at].in = nil, nil
			}
		}
	}
	c.outbox = compact(c.outbox, c.index)
	if len(resync) > 0 {
		c.wantFull(resync)
	}
	for _, name := range resync {
		if in := c.topics[name]; in != nil {
			in.throttle.mark()
		}
	}
	return nil
}

// compact removes the entries of released topics and re-indexes the outbox.
func compact(outbox []outgoing, index map[string]int) []outgoing {
	kept := outbox[:0]
	clear(index)
	for _, item := range outbox {
		if item.in != nil {
			index[item.in.topic.Name] = len(kept)
			kept = append(kept, item)
		}
	}
	clear(outbox[len(kept):])
	return kept
}

// acquireLocked returns the shared instance of topic+query with c subscribed; a new
// instance builds at once, an existing one re-reads (coalesced).
func (a *Area) acquireLocked(topic *Topic, query string, c *client) *instance {
	key := topic.Name + "?" + query
	in := a.instances[key]
	if in == nil {
		in = newInstance(a, topic, query)
		a.instances[key] = in
		if topic.Values != nil {
			a.valueUsers++
			if a.valueUsers == 1 && a.values != nil {
				a.cancelValues = a.values(a.valueChanged)
			}
		}
	}
	in.subs[c] = struct{}{}
	in.throttle.mark()
	return in
}

// releaseLocked unsubscribes c; the last subscriber closes the instance (no work remains).
func (a *Area) releaseLocked(in *instance, c *client) {
	delete(in.subs, c)
	c.forget(in) // the caller holds c.mu too
	if len(in.subs) > 0 || a.instances[in.key] != in {
		return
	}
	delete(a.instances, in.key)
	in.close()
	if in.topic.Values != nil {
		a.valueUsers--
		if a.valueUsers == 0 && a.cancelValues != nil {
			a.cancelValues()
			a.cancelValues = nil
		}
	}
}
