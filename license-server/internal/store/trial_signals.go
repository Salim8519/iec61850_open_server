package store

import (
	"errors"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketTrialIPs = []byte("trial_ips")
	// bucketTrialRefusals remembers the parts of a refused machine ID, so the owner can
	// clear exactly that hardware with reset-trial --hwid (the customer's Machine ID).
	bucketTrialRefusals  = []byte("trial_refusals")
	bucketTrialEmails    = []byte("trial_emails")
	bucketTrialEmailCode = []byte("trial_email_codes")

	// ErrBadCode means the email code is wrong, expired or for another address.
	ErrBadCode = errors.New("that code is not right or has expired; ask for a new code")
)

const maxCodeAttempts = 5

// EmailCode is one pending email confirmation for a machine ID. Waived is set when
// the mail could not be sent: the address is still unique, but no code is needed.
type EmailCode struct {
	EmailHash string `json:"email_hash"`
	MAC       string `json:"mac"`
	Expires   int64  `json:"expires"`
	Attempts  int    `json:"attempts"`
	Waived    bool   `json:"waived,omitempty"`
}

type emailRecord struct {
	HWID string `json:"hwid"`
	At   int64  `json:"at"`
}

// busyNetwork reports more than after new trials from ip within window.
func busyNetwork(tx *bolt.Tx, ip string, after int, window time.Duration, now time.Time) (bool, error) {
	if after <= 0 || ip == "" {
		return false, nil
	}
	recent, err := networkTrials(tx, ip, window, now)
	return len(recent) >= after, err
}

func networkTrials(tx *bolt.Tx, ip string, window time.Duration, now time.Time) ([]int64, error) {
	var times []int64
	if _, err := get(tx, bucketTrialIPs, ip, &times); err != nil {
		return nil, err
	}
	recent := []int64{}
	for _, at := range times {
		if window <= 0 || at > now.Add(-window).Unix() {
			recent = append(recent, at)
		}
	}
	return recent, nil
}

func recordNetworkTrial(tx *bolt.Tx, ip string, window time.Duration, now time.Time) error {
	if ip == "" {
		return nil
	}
	recent, err := networkTrials(tx, ip, window, now)
	if err != nil {
		return err
	}
	if len(recent) >= 200 {
		recent = recent[len(recent)-199:]
	}
	return put(tx, bucketTrialIPs, ip, append(recent, now.Unix()))
}

func emailUsed(tx *bolt.Tx, hash string) (bool, error) {
	var record emailRecord
	return get(tx, bucketTrialEmails, hash, &record)
}

func recordEmail(tx *bolt.Tx, hash, hwid string, now time.Time) error {
	return put(tx, bucketTrialEmails, hash, emailRecord{HWID: hwid, At: now.Unix()})
}

// EmailUsed reports whether a (hashed, normalised) address already had a trial.
func (s *Store) EmailUsed(hash string) (used bool, err error) {
	err = s.db.View(func(tx *bolt.Tx) error { used, err = emailUsed(tx, hash); return err })
	return used, err
}

// SaveEmailCode replaces the pending confirmation of a machine ID.
func (s *Store) SaveEmailCode(hwid string, code EmailCode) error {
	return s.db.Update(func(tx *bolt.Tx) error { return put(tx, bucketTrialEmailCode, hwid, code) })
}

// ConsumeEmailCode accepts the pending confirmation once: same address, not expired,
// and either waived (mail outage) or matching (check compares MACs). Wrong codes count;
// after five the code is dropped and a new one must be requested.
func (s *Store) ConsumeEmailCode(hwid, emailHash string, check func(mac string) bool, now time.Time) error {
	accepted := false
	err := s.db.Update(func(tx *bolt.Tx) error {
		var code EmailCode
		found, err := get(tx, bucketTrialEmailCode, hwid, &code)
		if err != nil || !found || now.Unix() >= code.Expires || code.EmailHash != emailHash {
			return err
		}
		bucket := tx.Bucket(bucketTrialEmailCode)
		if code.Waived || check(code.MAC) {
			accepted = true
			return bucket.Delete([]byte(hwid))
		}
		if code.Attempts++; code.Attempts >= maxCodeAttempts {
			return bucket.Delete([]byte(hwid))
		}
		return put(tx, bucketTrialEmailCode, hwid, code)
	})
	if err == nil && !accepted {
		err = ErrBadCode
	}
	return err
}
