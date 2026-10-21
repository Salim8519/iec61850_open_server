package native

import "sync"

// Fifo is an unbounded first-in first-out queue: the pipe reader only appends, so a slow
// consumer never stalls the worker's pipe (0111 rule 6), and order is kept (events before
// later replies, processes-1.md §4).
type Fifo[T any] struct {
	mu     sync.Mutex
	items  []T
	head   int
	ready  chan struct{}
	closed bool
}

// NewFifo returns an empty queue (also used by the runtime's native control feedback).
func NewFifo[T any]() *Fifo[T] { return &Fifo[T]{ready: make(chan struct{}, 1)} }

// Push appends item; it never blocks on the consumer.
func (q *Fifo[T]) Push(item T) {
	q.mu.Lock()
	q.items = append(q.items, item)
	q.mu.Unlock()
	select {
	case q.ready <- struct{}{}:
	default:
	}
}

// Close lets Pop return false once every queued item was taken.
func (q *Fifo[T]) Close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	select {
	case q.ready <- struct{}{}:
	default:
	}
}

// Pop waits for the next item; false means closed and drained.
func (q *Fifo[T]) Pop() (T, bool) {
	for {
		q.mu.Lock()
		if q.head < len(q.items) {
			item := q.items[q.head]
			var zero T
			q.items[q.head] = zero
			q.head++
			if q.head == len(q.items) {
				q.items, q.head = q.items[:0], 0
			}
			q.mu.Unlock()
			return item, true
		}
		closed := q.closed
		q.mu.Unlock()
		if closed {
			var zero T
			return zero, false
		}
		<-q.ready
	}
}
