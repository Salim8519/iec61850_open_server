package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Subscriber is one person who agreed to receive product news. People only get
// here by ticking the box at checkout or by the owner confirming they agreed.
type Subscriber struct {
	Email          string `json:"email"`
	Source         string `json:"source"`
	ConsentAt      int64  `json:"consent_at"`
	Token          string `json:"token"`
	UnsubscribedAt int64  `json:"unsubscribed_at,omitempty"`
}

// Active reports whether news may be sent to this person.
func (s Subscriber) Active() bool { return s.UnsubscribedAt == 0 }

// ErrUnsubscribed means the person said no; only their own new consent at
// checkout may add them again, never the owner.
var ErrUnsubscribed = errors.New("this person unsubscribed; only they can sign up again (at checkout)")

// ErrBadEmail means the address cannot be a real mailbox.
var ErrBadEmail = errors.New("enter a valid email address")

func newsKey(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// ValidEmail is a light check: one @, a dot in the domain, no spaces, sane length.
func ValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	local, domain, found := strings.Cut(email, "@")
	return found && local != "" && strings.Contains(domain, ".") && !strings.ContainsAny(email, " \t\r\n<>,;\"") &&
		!strings.Contains(domain, "@") && len(email) <= 254
}

// Subscribe adds or confirms a subscriber. source is "checkout" or "owner".
func (s *Store) Subscribe(email, source string, now time.Time) (Subscriber, error) {
	key := newsKey(email)
	if !ValidEmail(key) {
		return Subscriber{}, ErrBadEmail
	}
	var subscriber Subscriber
	err := s.db.Update(func(tx *bolt.Tx) error {
		found, err := get(tx, bucketNews, key, &subscriber)
		if err != nil {
			return err
		}
		if found && subscriber.Active() {
			return nil
		}
		if found && source != "checkout" {
			return ErrUnsubscribed
		}
		token := subscriber.Token
		if token == "" {
			raw := make([]byte, 16)
			if _, err := rand.Read(raw); err != nil {
				return err
			}
			token = hex.EncodeToString(raw)
		}
		subscriber = Subscriber{Email: key, Source: source, ConsentAt: now.Unix(), Token: token}
		return put(tx, bucketNews, key, subscriber)
	})
	return subscriber, err
}

// Unsubscribe marks the person behind token as unsubscribed. Unknown tokens
// return found=false so the public page can answer the same way either way.
func (s *Store) Unsubscribe(token string, now time.Time) (bool, error) {
	if len(token) != 32 {
		return false, nil
	}
	found := false
	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketNews)
		return bucket.ForEach(func(key, raw []byte) error {
			var subscriber Subscriber
			if json.Unmarshal(raw, &subscriber) != nil || subscriber.Token != token {
				return nil
			}
			found = true
			if subscriber.Active() {
				subscriber.UnsubscribedAt = now.Unix()
				return put(tx, bucketNews, string(key), subscriber)
			}
			return nil
		})
	})
	return found, err
}

// RemoveSubscriber deletes the record completely (a person's request to be forgotten).
func (s *Store) RemoveSubscriber(email string) error {
	return s.db.Update(func(tx *bolt.Tx) error { return tx.Bucket(bucketNews).Delete([]byte(newsKey(email))) })
}

// ListSubscribers returns everyone, active or not, in address order.
func (s *Store) ListSubscribers() ([]Subscriber, error) {
	subscribers := []Subscriber{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketNews).ForEach(func(_, raw []byte) error {
			var subscriber Subscriber
			if err := json.Unmarshal(raw, &subscriber); err != nil {
				return err
			}
			subscribers = append(subscribers, subscriber)
			return nil
		})
	})
	return subscribers, err
}
