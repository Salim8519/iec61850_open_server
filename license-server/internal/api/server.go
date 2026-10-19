// Package api serves the license, checkout and admin HTTP endpoints.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/config"
	"substation-license-server/internal/logging"
	"substation-license-server/internal/paypal"
	"substation-license-server/internal/sales"
	"substation-license-server/internal/security"
	"substation-license-server/internal/store"
)

// Payments is the PayPal surface the server uses (faked in tests).
type Payments interface {
	CreateOrder(ctx context.Context, order paypal.OrderRequest) (string, error)
	CaptureOrder(ctx context.Context, orderID string) (paypal.Capture, error)
	GetOrder(ctx context.Context, orderID string) (paypal.Capture, error)
	VerifyWebhook(ctx context.Context, webhookID string, header http.Header, body []byte) error
}

// Mailer sends one plain-text email (faked in tests).
type Mailer interface {
	Send(ctx context.Context, to, subject, text string) error
}

// Server wires configuration, storage, signing and abuse protection together.
type Server struct {
	Config   config.Config
	Store    *store.Store
	Signer   *certs.Signer
	Limiter  *security.Limiter
	Pow      *security.Verifier
	Log      *logging.Capped
	Payments Payments
	Mailer   Mailer
	Sales    *sales.Book
	Now      func() time.Time
	usage    dataUsage
}

// New builds a server with the documented production limits.
func New(cfg config.Config, db *store.Store, signer *certs.Signer, log *logging.Capped) *Server {
	limits := security.DefaultLimiterConfig()
	if cfg.RateGeneral > 0 {
		limits.GeneralPerMinute = float64(cfg.RateGeneral)
	}
	if cfg.RateSensitive > 0 {
		limits.SensitivePerMinute = float64(cfg.RateSensitive)
	}
	return &Server{Config: cfg, Store: db, Signer: signer, Log: log,
		Limiter: security.NewLimiter(limits), Sales: sales.Open(cfg.DataDir),
		Pow: security.NewVerifier(cfg.PowSecret, cfg.PowDifficulty), Now: time.Now}
}

type apiError struct {
	status int
	code   string
	text   string
}

func (e apiError) Error() string { return e.text }

func badRequest(text string) error { return apiError{http.StatusBadRequest, "bad_request", text} }

// Handler returns the complete route table.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.limited(security.ClassGeneral, s.health))
	mux.HandleFunc("GET /api/v1/challenge", s.limited(security.ClassGeneral, s.challenge))
	mux.HandleFunc("GET /api/v1/product", s.product)
	mux.HandleFunc("POST /api/v1/trial", s.limited(security.ClassSensitive, s.trial))
	mux.HandleFunc("POST /api/v1/trial/email-code", s.limited(security.ClassSensitive, s.trialEmailCode))
	mux.HandleFunc("POST /api/v1/activate", s.limited(security.ClassSensitive, s.activate))
	mux.HandleFunc("POST /api/v1/transfer", s.limited(security.ClassSensitive, s.transfer))
	mux.HandleFunc("POST /api/v1/verify", s.limited(security.ClassGeneral, s.verify))
	mux.HandleFunc("POST /api/v1/checkout/create-order", s.limited(security.ClassSensitive, s.createOrder))
	mux.HandleFunc("POST /api/v1/checkout/capture-order", s.limited(security.ClassSensitive, s.captureOrder))
	mux.HandleFunc("POST /api/v1/webhooks/paypal", s.paypalWebhook)
	mux.HandleFunc("POST /api/v1/admin/{action}", s.admin)
	mux.HandleFunc("GET /buy", s.buyPage)
	mux.HandleFunc("GET /activate", s.activatePage)
	mux.Handle("GET /static/{name}", staticHandler())
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/buy", http.StatusFound)
	})
	return s.capacity(securityHeaders(mux))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) clientIP(r *http.Request) string {
	return security.ClientIP(r, s.Config.TrustCloudflare)
}

type jsonHandler func(r *http.Request, ip string) (any, error)

func (s *Server) limited(class int, handler jsonHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := s.clientIP(r)
		if !s.Limiter.Allow(ip, class, s.Now()) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "Too many requests from your network. Wait a few minutes and try again.",
				"code":  "rate_limited"})
			return
		}
		result, err := handler(r, ip)
		if err != nil {
			s.writeError(w, r, ip, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, ip string, err error) {
	var known apiError
	switch {
	case errors.As(err, &known):
	case errors.Is(err, store.ErrUnknownKey):
		known = apiError{http.StatusNotFound, "unknown_key", err.Error()}
		s.Limiter.Fail(ip, s.Now())
	case errors.Is(err, store.ErrRevoked):
		known = apiError{http.StatusForbidden, "revoked", err.Error()}
	case errors.Is(err, store.ErrBoundElsewhere):
		known = apiError{http.StatusConflict, "bound_elsewhere", err.Error()}
	case errors.Is(err, store.ErrTransferLimit):
		known = apiError{http.StatusConflict, "transfer_limit", err.Error()}
	case errors.Is(err, store.ErrTrialUsed):
		known = apiError{http.StatusConflict, "trial_used", err.Error()}
	case errors.Is(err, store.ErrEmailRequired):
		known = apiError{http.StatusConflict, "email_required", err.Error()}
	case errors.Is(err, store.ErrEmailUsed):
		known = apiError{http.StatusConflict, "email_used", err.Error()}
	case errors.Is(err, store.ErrBadCode):
		known = apiError{http.StatusBadRequest, "bad_code", err.Error()}
		s.Limiter.Fail(ip, s.Now())
	default:
		s.Log.Event("internal_error", map[string]any{"path": r.URL.Path, "error": err.Error()})
		known = apiError{http.StatusInternalServerError, "server_error", "The license server had a problem. Try again later."}
	}
	writeJSON(w, known.status, map[string]string{"error": known.text, "code": known.code})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func decode(r *http.Request, target any) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		return badRequest("request body must be JSON")
	}
	return nil
}

func (s *Server) health(r *http.Request, _ string) (any, error) {
	return map[string]any{"ok": true, "kid": s.Signer.KeyID, "time": s.Now().Unix(),
		"checkout": s.Config.PayPalReady()}, nil
}

func (s *Server) challenge(r *http.Request, _ string) (any, error) {
	return security.IssueChallenge(s.Config.PowSecret, s.Config.PowDifficulty, s.Now()), nil
}

func (s *Server) checkPow(solution security.Solution) error {
	if err := s.Pow.Verify(solution, s.Now()); err != nil {
		return apiError{http.StatusBadRequest, "pow_invalid", err.Error()}
	}
	return nil
}
