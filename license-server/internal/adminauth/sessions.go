package adminauth

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// Session limits: signed out after 30 idle minutes or 12 hours in total.
const (
	SessionIdle   = 30 * time.Minute
	SessionMax    = 12 * time.Hour
	maxSessions   = 20
	sessionLength = 32
)

// Session is one signed-in browser. Only a hash of the cookie value is kept, so a
// memory dump does not reveal a usable cookie.
type Session struct {
	CSRF    string
	Epoch   int64
	IP      string
	Created time.Time
	Seen    time.Time
}

// Sessions lives in memory only: a restart signs everyone out, which is safe.
type Sessions struct {
	mu    sync.Mutex
	items map[string]*Session
}

// NewSessions returns an empty session table.
func NewSessions() *Sessions { return &Sessions{items: map[string]*Session{}} }

func tokenID(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Create starts a session and returns the cookie value. When the table is full
// the oldest session is dropped.
func (s *Sessions) Create(epoch int64, ip string, now time.Time) (string, Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) >= maxSessions {
		oldest, oldestAt := "", now
		for id, session := range s.items {
			if !session.Created.After(oldestAt) {
				oldest, oldestAt = id, session.Created
			}
		}
		delete(s.items, oldest)
	}
	token := RandomToken(sessionLength)
	session := &Session{CSRF: RandomToken(sessionLength), Epoch: epoch, IP: ip, Created: now, Seen: now}
	s.items[tokenID(token)] = session
	return token, *session
}

// Get returns a live session and refreshes its idle timer. Expired sessions and
// sessions from before the last reset are removed.
func (s *Sessions) Get(token string, epoch int64, now time.Time) (Session, bool) {
	if len(token) != sessionLength*2 {
		return Session{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := tokenID(token)
	session, ok := s.items[id]
	if !ok {
		return Session{}, false
	}
	if session.Epoch != epoch || now.Sub(session.Seen) > SessionIdle || now.Sub(session.Created) > SessionMax {
		delete(s.items, id)
		return Session{}, false
	}
	session.Seen = now
	return *session, true
}

// Delete signs one browser out.
func (s *Sessions) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, tokenID(token))
}

// DeleteAll signs every browser out.
func (s *Sessions) DeleteAll() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := len(s.items)
	s.items = map[string]*Session{}
	return count
}

// Count returns how many sessions are live (expired ones may still be counted
// until their next lookup).
func (s *Sessions) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}
