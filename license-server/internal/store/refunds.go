package store

import (
	"errors"

	bolt "go.etcd.io/bbolt"
)

// bucketRefunded remembers refunds that arrived before the order they refund, keyed
// by order id or capture id, so the late order (or capture) issues nothing.
var bucketRefunded = []byte("refunded_orders")

// ErrOrderRefunded means a refund for this order arrived before the order itself.
var ErrOrderRefunded = errors.New("this order was refunded before it was fulfilled")

// MarkRefunded records a refund for an order id or capture id the server has not seen.
// Repeating it changes nothing.
func (s *Store) MarkRefunded(id string, now int64) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketRefunded)
		if id == "" || bucket.Get([]byte(id)) != nil {
			return nil
		}
		return put(tx, bucketRefunded, id, map[string]int64{"at": now})
	})
}

func refundedBefore(tx *bolt.Tx, order Order) bool {
	bucket := tx.Bucket(bucketRefunded)
	for _, id := range []string{order.OrderID, order.CaptureID} {
		if id != "" && bucket.Get([]byte(id)) != nil {
			return true
		}
	}
	return false
}
