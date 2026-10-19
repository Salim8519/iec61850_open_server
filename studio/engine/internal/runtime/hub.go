package runtime

import (
	"log"
	"sync"
	"sync/atomic"
)

// Change says what changed in the runtime (S6 value hub):
//   - IEDID and Ref: one point's value, quality, timestamp or live_state;
//   - Ref == "": the state of one IED, or every value of it (IED state, goose_state, stale
//     marking, values reset, IED added to or removed from the run's index);
//   - IEDID == "" and Ref == "": the run state, or everything (also sent after a subscriber
//     fell behind and notifications were dropped: re-read all).
//
// The fields match gateway/service.Change, so service.Change(change) converts it.
type Change struct {
	IEDID string
	Ref   string
}

// hubQueue bounds the distinct changes waiting for one subscriber; beyond it the queue is
// dropped and the subscriber gets one Change{} (resync) instead.
const hubQueue = 4096

// hub fans changes out to subscribers. notify never blocks and never calls a subscriber: it
// only takes the subscriber's leaf lock to queue (and coalesce) the change, so it may run
// under any runtime lock. Each subscriber has one goroutine that delivers in queue order.
type hub struct {
	mu   sync.Mutex // Subscribe and cancel only
	subs atomic.Pointer[[]*subscriber]
}

type subscriber struct {
	notify  func(Change)
	mu      sync.Mutex
	queue   []Change
	spare   []Change
	pending map[Change]struct{}
	resync  bool
	wake    chan struct{}
	stop    chan struct{}
	once    sync.Once
	stopped atomic.Bool
}

// Subscribe calls notify after every value, quality, timestamp, live_state, IED-state and
// run-state change until cancel is called. notify runs on the subscriber's own goroutine,
// one change at a time; a slow notify only delays its own subscriber. Repeated changes of
// one point that are still queued are delivered once; after an overflow the subscriber gets
// Change{} (re-read everything). cancel may be called from inside notify.
func (c *Coordinator) Subscribe(notify func(Change)) (cancel func()) {
	return c.hub.subscribe(notify)
}

func (h *hub) subscribe(notify func(Change)) func() {
	s := &subscriber{notify: notify, queue: make([]Change, 0, 64), spare: make([]Change, 0, 64),
		pending: make(map[Change]struct{}, 64), wake: make(chan struct{}, 1), stop: make(chan struct{})}
	h.mu.Lock()
	var next []*subscriber
	if current := h.subs.Load(); current != nil {
		next = append(next, *current...)
	}
	next = append(next, s)
	h.subs.Store(&next)
	h.mu.Unlock()
	go s.run()
	return func() { h.cancel(s) }
}

func (h *hub) cancel(s *subscriber) {
	s.once.Do(func() {
		s.stopped.Store(true)
		h.mu.Lock()
		var next []*subscriber
		if current := h.subs.Load(); current != nil {
			for _, other := range *current {
				if other != s {
					next = append(next, other)
				}
			}
		}
		h.subs.Store(&next)
		h.mu.Unlock()
		close(s.stop)
	})
}

// notify queues change for every subscriber; it allocates nothing once queues are warm.
func (h *hub) notify(change Change) {
	list := h.subs.Load()
	if list == nil {
		return
	}
	for _, s := range *list {
		s.offer(change)
	}
}

func (s *subscriber) offer(change Change) {
	s.mu.Lock()
	if s.resync {
		s.mu.Unlock() // everything is re-read anyway
		return
	}
	if _, queued := s.pending[change]; queued {
		s.mu.Unlock()
		return
	}
	// Only the first change after a drain wakes the goroutine: it takes the whole queue.
	wake := len(s.queue) == 0
	if len(s.queue) >= hubQueue {
		s.resync = true
		s.queue = s.queue[:0]
		clear(s.pending)
	} else {
		s.queue = append(s.queue, change)
		s.pending[change] = struct{}{}
	}
	s.mu.Unlock()
	if !wake {
		return
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *subscriber) run() {
	for {
		select {
		case <-s.stop:
			return
		case <-s.wake:
		}
		s.mu.Lock()
		batch, resync := s.queue, s.resync
		s.queue, s.resync = s.spare[:0], false
		for _, change := range batch {
			delete(s.pending, change)
		}
		s.mu.Unlock()
		if resync {
			s.deliver(Change{})
		} else {
			for _, change := range batch {
				if s.stopped.Load() {
					return
				}
				s.deliver(change)
			}
		}
		s.mu.Lock()
		s.spare = batch[:0]
		s.mu.Unlock()
	}
}

func (s *subscriber) deliver(change Change) {
	if s.stopped.Load() {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("value hub subscriber failed: %v", recovered)
		}
	}()
	s.notify(change)
}
