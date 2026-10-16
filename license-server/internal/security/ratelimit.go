package security

import (
	"container/list"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Bucket classes: general traffic and sensitive license operations.
const (
	ClassGeneral   = 0
	ClassSensitive = 1
)

// LimiterConfig sets per-minute rates, failure ban and the memory ceiling.
type LimiterConfig struct {
	GeneralPerMinute   float64
	SensitivePerMinute float64
	MaxFailures        int
	BanDuration        time.Duration
	MaxEntries         int
}

// DefaultLimiterConfig is the documented production policy.
func DefaultLimiterConfig() LimiterConfig {
	return LimiterConfig{GeneralPerMinute: 10, SensitivePerMinute: 5, MaxFailures: 8,
		BanDuration: 30 * time.Minute, MaxEntries: 10000}
}

type entry struct {
	ip          string
	tokens      [2]float64
	updated     time.Time
	failures    int
	bannedUntil time.Time
}

// Limiter is a bounded LRU of per-IP token buckets; the oldest IP is evicted
// when full, so memory has a fixed ceiling (~60 bytes per entry plus map cost).
type Limiter struct {
	cfg   LimiterConfig
	mu    sync.Mutex
	order *list.List
	items map[string]*list.Element
}

// NewLimiter creates an empty limiter.
func NewLimiter(cfg LimiterConfig) *Limiter {
	return &Limiter{cfg: cfg, order: list.New(), items: map[string]*list.Element{}}
}

func (l *Limiter) get(ip string, now time.Time) *entry {
	if element, ok := l.items[ip]; ok {
		l.order.MoveToFront(element)
		return element.Value.(*entry)
	}
	if l.order.Len() >= l.cfg.MaxEntries {
		oldest := l.order.Back()
		l.order.Remove(oldest)
		delete(l.items, oldest.Value.(*entry).ip)
	}
	e := &entry{ip: ip, updated: now,
		tokens: [2]float64{l.cfg.GeneralPerMinute, l.cfg.SensitivePerMinute}}
	l.items[ip] = l.order.PushFront(e)
	return e
}

// Allow spends one token of the class; banned IPs are always refused.
func (l *Limiter) Allow(ip string, class int, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.get(ip, now)
	if now.Before(e.bannedUntil) {
		return false
	}
	elapsed := now.Sub(e.updated).Minutes()
	e.updated = now
	rates := [2]float64{l.cfg.GeneralPerMinute, l.cfg.SensitivePerMinute}
	for i := range e.tokens {
		e.tokens[i] = minFloat(rates[i], e.tokens[i]+elapsed*rates[i])
	}
	if e.tokens[class] < 1 {
		return false
	}
	e.tokens[class]--
	return true
}

// Fail records a bad key attempt and bans after MaxFailures.
func (l *Limiter) Fail(ip string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.get(ip, now)
	e.failures++
	if e.failures >= l.cfg.MaxFailures {
		e.bannedUntil = now.Add(l.cfg.BanDuration)
		e.failures = 0
	}
}

// Banned reports whether an IP is currently banned.
func (l *Limiter) Banned(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	element, ok := l.items[ip]
	return ok && now.Before(element.Value.(*entry).bannedUntil)
}

// Len returns the number of tracked IPs.
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.order.Len()
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ClientIP trusts CF-Connecting-IP only when the TCP peer is the local cloudflared.
func ClientIP(r *http.Request, trustCloudflare bool) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if trustCloudflare && peer != nil && peer.IsLoopback() {
		if forwarded := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); net.ParseIP(forwarded) != nil {
			return forwarded
		}
	}
	return host
}
