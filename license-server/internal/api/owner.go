package api

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"time"

	"substation-license-server/internal/adminauth"
	"substation-license-server/internal/audit"
	"substation-license-server/internal/security"
)

// The owner page (ADR-0072) lives under Config.AdminPath and answers only requests
// that carry the gateway secret, which the Cloudflare Worker on scadabench.com adds.
// Anything else gets the same 404 as an unknown address, so the page cannot be found.
const (
	gatewayHeader  = "X-Admin-Gateway"
	clientIPHeader = "X-Admin-Client-IP"
	confirmHeader  = "X-Confirm-Code"
	csrfHeader     = "X-CSRF-Token"
	ownerCookie    = "__Secure-owner"
)

// ownerState is everything the owner page keeps between requests.
type ownerState struct {
	keeper   *adminauth.Keeper
	sessions *adminauth.Sessions
	lockout  *adminauth.Lockout
	limiter  *security.Limiter
	audit    *audit.Log
	news     *newsJob
	site     *siteWatch
	loginMu  chan struct{}
}

func newOwnerState(dataDir string) *ownerState {
	return &ownerState{keeper: adminauth.NewKeeper(dataDir), sessions: adminauth.NewSessions(),
		lockout: adminauth.NewLockout(10, time.Hour, time.Hour),
		limiter: security.NewLimiter(security.LimiterConfig{GeneralPerMinute: 120, SensitivePerMinute: 6,
			MaxFailures: 5, BanDuration: time.Hour, MaxEntries: 1000}),
		audit: audit.Open(dataDir), news: &newsJob{}, site: &siteWatch{}, loginMu: make(chan struct{}, 1)}
}

// ownerCall is one checked request: who asked and from which signed-in browser.
type ownerCall struct {
	ip      string
	session adminauth.Session
	token   string
}

type ownerHandler func(r *http.Request, call *ownerCall) (any, error)

// Route options for owner API calls.
const (
	public    = 0
	signedIn  = 1
	confirmed = 2 // signed in and a fresh authenticator code in X-Confirm-Code
)

func (s *Server) mountOwner(mux *http.ServeMux) {
	if !s.Config.AdminReady() {
		return
	}
	base := s.Config.AdminPath
	page := http.NewServeMux()
	page.HandleFunc("GET "+base+"/{$}", s.ownerPage)
	page.HandleFunc("GET "+base+"/static/{name}", s.ownerStatic)
	s.ownerRoutes(page, base+"/api/")
	page.HandleFunc(base+"/", func(w http.ResponseWriter, r *http.Request) { ownerNotFound(w) })
	gated := s.ownerGate(page)
	mux.Handle(base+"/", gated)
	mux.Handle(base, s.ownerGate(http.RedirectHandler(base+"/", http.StatusFound)))
}

// ownerGate checks the gateway secret first, then the per-address rate limit, and
// sets the strict headers every owner response carries.
func (s *Server) ownerGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		supplied := r.Header.Get(gatewayHeader)
		if !s.Config.AdminReady() || subtle.ConstantTimeCompare([]byte(supplied), []byte(s.Config.AdminGatewaySecret)) != 1 {
			ownerNotFound(w)
			return
		}
		ip := s.ownerIP(r)
		if !s.owner.limiter.Allow(ip, security.ClassGeneral, s.Now()) {
			http.Error(w, "Too many requests. Wait a minute.", http.StatusTooManyRequests)
			return
		}
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; "+
			"img-src 'self' data:; font-src 'self'; connect-src 'self'; form-action 'self'; "+
			"base-uri 'none'; frame-ancestors 'none'")
		h.Set("Strict-Transport-Security", "max-age=31536000")
		h.Set("X-Robots-Tag", "noindex, nofollow")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func ownerNotFound(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	http.Error(w, "404 page not found", http.StatusNotFound)
}

// ownerIP trusts the Worker's copy of the visitor address only after the gateway
// secret matched (ownerGate runs first).
func (s *Server) ownerIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get(clientIPHeader)); net.ParseIP(forwarded) != nil {
		return forwarded
	}
	return s.clientIP(r)
}

// ownerAPI wraps one owner call with the origin, session, CSRF and confirmation checks.
func (s *Server) ownerAPI(level int, handler ownerHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		call := &ownerCall{ip: s.ownerIP(r)}
		if r.Method != http.MethodGet {
			if r.Header.Get("Origin") != s.Config.AdminOrigin ||
				!strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "Request refused.", "code": "origin"})
				return
			}
		}
		if level >= signedIn && !s.ownerSession(r, call) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Please sign in again.", "code": "signed_out"})
			return
		}
		if level >= signedIn && r.Method != http.MethodGet &&
			subtle.ConstantTimeCompare([]byte(r.Header.Get(csrfHeader)), []byte(call.session.CSRF)) != 1 {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Reload the page and try again.", "code": "csrf"})
			return
		}
		if level >= confirmed && !s.ownerConfirm(r, call) {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": "Type a new 6-digit code from your authenticator app to confirm (each code works once).", "code": "confirm"})
			return
		}
		result, err := handler(r, call)
		if err != nil {
			s.writeError(w, r, call.ip, err)
			return
		}
		if cookie, ok := result.(ownerCookieResult); ok {
			setOwnerCookie(w, cookie)
			result = map[string]any{"ok": true, "csrf": cookie.csrf}
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func (s *Server) ownerSession(r *http.Request, call *ownerCall) bool {
	cookie, err := r.Cookie(ownerCookie)
	if err != nil {
		return false
	}
	account, ready, err := s.owner.keeper.Load()
	if err != nil || !ready {
		return false
	}
	session, ok := s.owner.sessions.Get(cookie.Value, account.Epoch, s.Now())
	call.session, call.token = session, cookie.Value
	return ok
}

// ownerConfirm checks a fresh authenticator code for risky actions; wrong codes
// count toward the address ban like wrong sign-ins.
func (s *Server) ownerConfirm(r *http.Request, call *ownerCall) bool {
	code, ok := r.Header.Get(confirmHeader), false
	_, _, err := s.owner.keeper.Update(func(a *adminauth.Account) bool {
		ok = a.CheckAuthenticator(code, s.Now())
		return ok
	})
	if err == nil && ok {
		return true
	}
	s.owner.limiter.Fail(call.ip, s.Now())
	s.ownerAudit("confirm", call.ip, r.URL.Path, false)
	return false
}

func (s *Server) ownerAudit(action, ip, detail string, ok bool) {
	if err := s.owner.audit.Add(action, ip, detail, ok, s.Now()); err != nil {
		s.Log.Event("audit_error", map[string]any{"error": err.Error()})
	}
}
