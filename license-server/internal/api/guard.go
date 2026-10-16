package api

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"sync"
	"time"
)

// maxInFlight bounds memory and CPU on a small VPS: beyond it the server answers
// 503 at once instead of queueing work. Normal traffic is a few requests a minute.
const maxInFlight = 48

func (s *Server) capacity(next http.Handler) http.Handler {
	slots := make(chan struct{}, maxInFlight)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Retry-After", "10")
			http.Error(w, "The license server is busy. Try again in a few seconds.", http.StatusServiceUnavailable)
		}
	})
}

// dataUsage caches the data folder size for a minute so the check costs nothing.
type dataUsage struct {
	mu       sync.Mutex
	bytes    int64
	checked  time.Time
	reported time.Time
}

func (u *dataUsage) size(dir string, now time.Time) int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	if now.Sub(u.checked) < time.Minute && !u.checked.IsZero() {
		return u.bytes
	}
	u.bytes, u.checked = FolderBytes(dir), now
	return u.bytes
}

// FolderBytes adds up every file under dir (database, logs, sales record, keys).
func FolderBytes(dir string) int64 {
	var total int64
	filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			if info, infoErr := entry.Info(); infoErr == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

// dataFull pauses new free trials (the only records strangers can create) when the
// data folder reaches its budget. Purchases, activations and checks keep working.
func (s *Server) dataFull() bool {
	if s.Config.DataDir == "" {
		return false
	}
	now := s.Now()
	full := s.usage.size(s.Config.DataDir, now) >= s.Config.DataBudgetBytes()
	s.usage.mu.Lock()
	report := full && now.Sub(s.usage.reported) > time.Hour
	if report {
		s.usage.reported = now
	}
	s.usage.mu.Unlock()
	if report {
		s.Log.Event("data_budget_reached", map[string]any{"limit_mb": s.Config.DataBudgetBytes() >> 20})
	}
	return full
}

var errTrialsPaused = apiError{http.StatusServiceUnavailable, "trials_paused",
	"New free trials are paused for a moment. Try again later or contact support."}

// pageHeaders lets the checkout pages load only this server and PayPal; HSTS keeps
// buyers on HTTPS. PayPal's documented domains are allowed for its buttons and card form.
func pageHeaders(w http.ResponseWriter) {
	paypal := "https://*.paypal.com https://*.paypalobjects.com https://*.venmo.com"
	w.Header().Set("Content-Security-Policy", "default-src 'self'; "+
		"script-src 'self' 'unsafe-inline' "+paypal+"; "+
		"connect-src 'self' "+paypal+" https://*.braintreegateway.com; "+
		"frame-src "+paypal+" https://*.braintreegateway.com https://*.cardinalcommerce.com; "+
		"img-src 'self' data: https://scadabench.com "+paypal+"; "+
		"style-src 'self' 'unsafe-inline'; font-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
}
