// Package security holds the stateless proof-of-work puzzle and the IP limiter.
package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/bits"
	"strconv"
	"sync"
	"time"
)

// ChallengeWindow is how long a solved puzzle stays acceptable.
const ChallengeWindow = 5 * time.Minute

// Challenge is sent to clients; the MAC lets the server verify it without storage.
type Challenge struct {
	Nonce      string `json:"nonce"`
	Timestamp  int64  `json:"timestamp"`
	Difficulty int    `json:"difficulty"`
	MAC        string `json:"mac"`
}

// Solution is what a client returns alongside its request.
type Solution struct {
	Nonce      string `json:"pow_nonce"`
	Timestamp  int64  `json:"pow_ts"`
	Difficulty int    `json:"pow_difficulty"`
	MAC        string `json:"pow_mac"`
	Counter    uint64 `json:"pow_counter"`
}

func challengeMAC(secret, nonce string, timestamp int64, difficulty int) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%s|%d|%d", nonce, timestamp, difficulty)
	return hex.EncodeToString(mac.Sum(nil))
}

// IssueChallenge creates a fresh signed puzzle.
func IssueChallenge(secret string, difficulty int, now time.Time) Challenge {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic("system random source failed: " + err.Error())
	}
	nonce := hex.EncodeToString(buf)
	ts := now.Unix()
	return Challenge{Nonce: nonce, Timestamp: ts, Difficulty: difficulty,
		MAC: challengeMAC(secret, nonce, ts, difficulty)}
}

// LeadingZeroBits counts zero bits at the start of a digest.
func LeadingZeroBits(digest []byte) int {
	total := 0
	for _, b := range digest {
		if b == 0 {
			total += 8
			continue
		}
		return total + bits.LeadingZeros8(b)
	}
	return total
}

// Digest is SHA256(nonce + ":" + counter), shared with the Python and JS solvers.
func Digest(nonce string, counter uint64) [32]byte {
	return sha256.Sum256([]byte(nonce + ":" + strconv.FormatUint(counter, 10)))
}

// Solve brute-forces a counter (used by tests and the CLI).
func Solve(c Challenge) uint64 {
	for counter := uint64(0); ; counter++ {
		digest := Digest(c.Nonce, counter)
		if LeadingZeroBits(digest[:]) >= c.Difficulty {
			return counter
		}
	}
}

// Verifier checks solutions and refuses to accept the same nonce twice.
type Verifier struct {
	secret     string
	minimum    int
	mu         sync.Mutex
	used       map[string]int64
	lastPurged int64
}

// NewVerifier requires at least minimum difficulty from every solution.
func NewVerifier(secret string, minimum int) *Verifier {
	return &Verifier{secret: secret, minimum: minimum, used: map[string]int64{}}
}

// Verify validates MAC, time window, difficulty, work and single use.
func (v *Verifier) Verify(s Solution, now time.Time) error {
	if s.Difficulty < v.minimum {
		return errors.New("proof of work is too easy")
	}
	expected := challengeMAC(v.secret, s.Nonce, s.Timestamp, s.Difficulty)
	if !hmac.Equal([]byte(expected), []byte(s.MAC)) {
		return errors.New("proof of work challenge is not ours")
	}
	age := now.Unix() - s.Timestamp
	if age < -30 || age > int64(ChallengeWindow.Seconds()) {
		return errors.New("proof of work challenge expired; try again")
	}
	digest := Digest(s.Nonce, s.Counter)
	if LeadingZeroBits(digest[:]) < s.Difficulty {
		return errors.New("proof of work is not solved")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.purge(now.Unix())
	if _, seen := v.used[s.Nonce]; seen {
		return errors.New("proof of work was already used")
	}
	if len(v.used) >= 100000 {
		return errors.New("server is busy; try again in a minute")
	}
	v.used[s.Nonce] = s.Timestamp
	return nil
}

func (v *Verifier) purge(now int64) {
	if now-v.lastPurged < 30 {
		return
	}
	v.lastPurged = now
	limit := now - int64(ChallengeWindow.Seconds()) - 60
	for nonce, ts := range v.used {
		if ts < limit {
			delete(v.used, nonce)
		}
	}
}
