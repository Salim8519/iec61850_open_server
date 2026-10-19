// Package store keeps trials, licenses and orders in one BoltDB file.
package store

import (
	"encoding/json"
	"errors"
	"io"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketTrials   = []byte("trials")
	bucketLicenses = []byte("licenses")
	bucketOrders   = []byte("orders")

	// ErrTrialUsed means this machine already had its one trial.
	ErrTrialUsed = errors.New("this computer has already used its free trial")
	// ErrUnknownKey means the key was never issued.
	ErrUnknownKey = errors.New("license key not found; check for typing mistakes")
	// ErrRevoked means the key was cancelled (refund, chargeback or admin).
	ErrRevoked = errors.New("this license key was cancelled; contact support")
	// ErrBoundElsewhere means the key is locked to a different computer.
	ErrBoundElsewhere = errors.New("this license key is already activated on another computer")
	// ErrTransferLimit means the yearly move allowance is used up.
	ErrTransferLimit = errors.New("this key was moved too often this year; contact support")
	// ErrDatabaseBusy means another process (usually the running server) holds the file.
	ErrDatabaseBusy = errors.New("database is in use by the running server")
)

// Trial is one machine's single free trial.
type Trial struct {
	HWID      string `json:"hwid"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
	IP        string `json:"ip"`
	// Parts are salted hashes of the machine parts (empty for old clients).
	Parts TrialParts `json:"parts,omitempty"`
	// Email is true when the trial needed (and passed) an email confirmation.
	Email bool `json:"email,omitempty"`
}

// License is one sold or admin-issued key.
type License struct {
	Key         string  `json:"key"`
	Email       string  `json:"email"`
	Type        string  `json:"type"`
	Days        int     `json:"days,omitempty"`
	HWID        string  `json:"hwid"`
	ActivatedAt int64   `json:"activated_at"`
	CreatedAt   int64   `json:"created_at"`
	Revoked     bool    `json:"revoked"`
	Reason      string  `json:"reason,omitempty"`
	OrderID     string  `json:"order_id,omitempty"`
	Transfers   []int64 `json:"transfers,omitempty"`
}

// Order is one completed PayPal purchase.
type Order struct {
	OrderID    string `json:"order_id"`
	CaptureID  string `json:"capture_id"`
	Email      string `json:"email"`
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	LicenseKey string `json:"license_key"`
	CreatedAt  int64  `json:"created_at"`
	Emailed    bool   `json:"emailed"`
}

// Store wraps the BoltDB handle.
type Store struct{ db *bolt.DB }

// Open opens (or creates) the database; it fails fast if the server holds it.
func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: time.Second})
	if errors.Is(err, bolt.ErrTimeout) {
		return nil, ErrDatabaseBusy
	}
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range [][]byte{bucketTrials, bucketLicenses, bucketOrders, bucketTrialParts, bucketTrialIPs,
			bucketTrialRefusals, bucketTrialEmails, bucketTrialEmailCode} {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Backup streams a consistent hot copy of the database.
func (s *Store) Backup(w io.Writer) error {
	return s.db.View(func(tx *bolt.Tx) error { _, err := tx.WriteTo(w); return err })
}

func get(tx *bolt.Tx, bucket []byte, key string, target any) (bool, error) {
	raw := tx.Bucket(bucket).Get([]byte(key))
	if raw == nil {
		return false, nil
	}
	return true, json.Unmarshal(raw, target)
}

func put(tx *bolt.Tx, bucket []byte, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return tx.Bucket(bucket).Put([]byte(key), raw)
}

// TrialCount returns how many machines started a trial.
func (s *Store) TrialCount() (count int) {
	s.db.View(func(tx *bolt.Tx) error { count = tx.Bucket(bucketTrials).Stats().KeyN; return nil })
	return count
}

// CreateLicense stores a new key.
func (s *Store) CreateLicense(license License) error {
	return s.db.Update(func(tx *bolt.Tx) error { return put(tx, bucketLicenses, license.Key, license) })
}

// GetLicense reads one key.
func (s *Store) GetLicense(key string) (License, error) {
	var license License
	err := s.db.View(func(tx *bolt.Tx) error {
		found, err := get(tx, bucketLicenses, key, &license)
		if err == nil && !found {
			err = ErrUnknownKey
		}
		return err
	})
	return license, err
}

// UpdateLicense applies change to a key inside one transaction.
func (s *Store) UpdateLicense(key string, change func(*License) error) (License, error) {
	var license License
	err := s.db.Update(func(tx *bolt.Tx) error {
		found, err := get(tx, bucketLicenses, key, &license)
		if err != nil {
			return err
		}
		if !found {
			return ErrUnknownKey
		}
		if err := change(&license); err != nil {
			return err
		}
		return put(tx, bucketLicenses, key, license)
	})
	return license, err
}

// ListLicenses returns every key (admin view).
func (s *Store) ListLicenses() ([]License, error) {
	licenses := []License{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketLicenses).ForEach(func(_, raw []byte) error {
			var license License
			if err := json.Unmarshal(raw, &license); err != nil {
				return err
			}
			licenses = append(licenses, license)
			return nil
		})
	})
	return licenses, err
}
