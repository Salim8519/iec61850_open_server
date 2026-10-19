package native

import "sync"

// fifo is an unbounded first-in first-out queue: the pipe reader only appends, so a slow
// consumer never stalls the worker's pipe (0111 rule 6), and order is kept (events before
// later replies, processes-1.md §4).
type fifo[T any] struct {
	mu     sync.Mutex
	items  []T
	head   int
	ready  chan struct{}
	closed bool
}

func newFifo[T any]() *fifo[T] { return &fifo[T]{ready: make(chan struct{}, 1)} }

func (q *fifo[T]) push(item T) {
	q.mu.Lock()
	q.items = append(q.items, item)
	q.mu.Unlock()
	select {
	case q.ready <- struct{}{}:
	default:
	}
}

// close lets pop return false once every queued item was taken.
func (q *fifo[T]) close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	select {
	case q.ready <- struct{}{}:
	default:
	}
}

// pop waits for the next item; false means closed and drained.
func (q *fifo[T]) pop() (T, bool) {
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
