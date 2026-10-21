package store

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Activate binds an unbound key to hwid, or confirms the same machine again. A key that
// is bound to another machine is never rebound: each license works on one computer only
// (ADR-0097), so there is no transfer and no reset in the store.
func (s *Store) Activate(key, hwid string, now time.Time) (License, error) {
	return s.ActivateWithHostKey(key, hwid, "", now)
}

// ActivateWithHostKey is Activate that also remembers the PC's public VM host key. The newest key
// wins, so a Windows reinstall (a new key pair) can activate again; an empty hostKey changes nothing.
func (s *Store) ActivateWithHostKey(key, hwid, hostKey string, now time.Time) (License, error) {
	return s.UpdateLicense(key, func(license *License) error {
		if license.Revoked {
			return ErrRevoked
		}
		if license.HWID != "" && license.HWID != hwid {
			return ErrBoundElsewhere
		}
		if license.HWID == "" {
			license.HWID = hwid
			license.ActivatedAt = now.Unix()
		}
		if hostKey != "" && license.Type != "trial" {
			license.HostKey = hostKey
		}
		return nil
	})
}

// Revoke cancels a key with a reason (refund, chargeback, admin).
func (s *Store) Revoke(key, reason string) (License, error) {
	return s.UpdateLicense(key, func(license *License) error {
		license.Revoked, license.Reason = true, reason
		return nil
	})
}

// DeleteLicense removes a key completely (owner testing; use revoke for customers).
func (s *Store) DeleteLicense(key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketLicenses)
		if bucket.Get([]byte(key)) == nil {
			return ErrUnknownKey
		}
		return bucket.Delete([]byte(key))
	})
}

// IssueForOrder is idempotent: the first call creates the key and order, later
// calls (browser capture, webhook, retries) return the same key.
func (s *Store) IssueForOrder(order Order, newKey func() string, now time.Time) (Order, bool, error) {
	created := false
	err := s.db.Update(func(tx *bolt.Tx) error {
		var existing Order
		found, err := get(tx, bucketOrders, order.OrderID, &existing)
		if err != nil {
			return err
		}
		if found {
			order = existing
			return nil
		}
		order.LicenseKey = newKey()
		order.CreatedAt = now.Unix()
		license := License{Key: order.LicenseKey, Email: order.Email, Type: "lifetime",
			CreatedAt: now.Unix(), OrderID: order.OrderID}
		if err := put(tx, bucketLicenses, license.Key, license); err != nil {
			return err
		}
		created = true
		return put(tx, bucketOrders, order.OrderID, order)
	})
	return order, created, err
}

// GetOrder reads an order by PayPal order id.
func (s *Store) GetOrder(orderID string) (Order, bool, error) {
	var order Order
	var found bool
	err := s.db.View(func(tx *bolt.Tx) error {
		var err error
		found, err = get(tx, bucketOrders, orderID, &order)
		return err
	})
	return order, found, err
}

// MarkEmailed records that the delivery email went out.
func (s *Store) MarkEmailed(orderID string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		var order Order
		found, err := get(tx, bucketOrders, orderID, &order)
		if err != nil || !found {
			return err
		}
		order.Emailed = true
		return put(tx, bucketOrders, orderID, order)
	})
}

// FindOrderByCapture finds the order a refund/reversal refers to.
func (s *Store) FindOrderByCapture(captureID string) (Order, bool, error) {
	var result Order
	found := false
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketOrders).ForEach(func(_, raw []byte) error {
			var order Order
			if err := json.Unmarshal(raw, &order); err == nil && order.CaptureID == captureID {
				result, found = order, true
			}
			return nil
		})
	})
	return result, found, err
}
