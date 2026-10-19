package store

import (
	"time"

	bolt "go.etcd.io/bbolt"
)

// ResetTrial forgets a machine's trial (and its parts) so it can start one again. A
// machine ID that was refused or asked for an email has those parts forgotten too.
func (s *Store) ResetTrial(hwid string) (bool, error) {
	found := false
	err := s.db.Update(func(tx *bolt.Tx) error {
		var trial Trial
		var err error
		if found, err = get(tx, bucketTrials, hwid, &trial); err != nil {
			return err
		}
		if err := unindex(tx, hwid, trial.Parts); err != nil {
			return err
		}
		var refused TrialParts
		hadRefusal, err := get(tx, bucketTrialRefusals, hwid, &refused)
		if err != nil {
			return err
		}
		for _, part := range refused.entries() {
			if err := tx.Bucket(bucketTrialParts).Delete([]byte(part.key)); err != nil {
				return err
			}
		}
		found = found || hadRefusal
		for _, bucket := range [][]byte{bucketTrialRefusals, bucketTrialEmailCode, bucketTrials} {
			if err := tx.Bucket(bucket).Delete([]byte(hwid)); err != nil {
				return err
			}
		}
		return nil
	})
	return found, err
}

// ResetTrialParts forgets these parts entirely, so no earlier trial matches them
// (owner override). It returns how many parts were known.
func (s *Store) ResetTrialParts(parts TrialParts) (int, error) {
	known := 0
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketTrialParts)
		for _, part := range parts.Clean().entries() {
			if bucket.Get([]byte(part.key)) != nil {
				known++
			}
			if err := bucket.Delete([]byte(part.key)); err != nil {
				return err
			}
		}
		return nil
	})
	return known, err
}

// ResetTrialIP clears the recent new-trial count of one network address.
func (s *Store) ResetTrialIP(ip string) (bool, error) {
	return s.deleteKey(bucketTrialIPs, ip)
}

// AllowEmail lets a (hashed, normalised) address start one more trial.
func (s *Store) AllowEmail(hash string) (bool, error) {
	return s.deleteKey(bucketTrialEmails, hash)
}

func (s *Store) deleteKey(name []byte, key string) (bool, error) {
	found := false
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(name)
		found = bucket.Get([]byte(key)) != nil
		return bucket.Delete([]byte(key))
	})
	return found, err
}

// BusyNetworks counts addresses with more than after new trials within window, for
// the owner's list (a signal to look at, never a block).
func (s *Store) BusyNetworks(after int, window time.Duration, now time.Time) (count int, err error) {
	if after <= 0 {
		return 0, nil
	}
	err = s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketTrialIPs).ForEach(func(ip, _ []byte) error {
			recent, err := networkTrials(tx, string(ip), window, now)
			if err == nil && len(recent) > after {
				count++
			}
			return err
		})
	})
	return count, err
}

func unindex(tx *bolt.Tx, hwid string, parts TrialParts) error {
	for _, part := range parts.entries() {
		var machines []string
		if _, err := get(tx, bucketTrialParts, part.key, &machines); err != nil {
			return err
		}
		kept := []string{}
		for _, machine := range machines {
			if machine != hwid {
				kept = append(kept, machine)
			}
		}
		if err := put(tx, bucketTrialParts, part.key, kept); err != nil {
			return err
		}
	}
	return nil
}
