package api

import (
	"net/http"
	"strconv"

	"substation-license-server/internal/release"
	"substation-license-server/internal/security"
)

// LoadRelease signs the saved release record once; every request then only copies bytes.
// No record yet is fine: the endpoint answers 404 "no_release" until the owner publishes one.
func (s *Server) LoadRelease() error {
	info, ok, err := release.Load(s.Config.DataDir)
	if err != nil || !ok {
		s.releaseBody = nil
		return err
	}
	signed, err := release.Sign(info, s.Signer.KeyID, s.Signer.SignRelease)
	if err != nil {
		return err
	}
	s.releaseBody = &signed
	return nil
}

// latestRelease answers "what is the newest ScadaBench" for the Windows app's daily check.
// Abuse protection: per-IP token bucket (general class) and the shared in-flight cap; the
// answer is a fixed signed byte string (no signing, encoding or disk per request); a matching
// If-None-Match costs only a 304; no request body, query or cookie is read; no CORS, so web
// pages cannot make browsers call it. Nothing about the caller is logged or stored.
func (s *Server) latestRelease(w http.ResponseWriter, r *http.Request) {
	if !s.Limiter.Allow(s.clientIP(r), security.ClassGeneral, s.Now()) {
		w.Header().Set("Retry-After", "60")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "Too many update checks from your network. Try again later.", "code": "rate_limited"})
		return
	}
	signed := s.releaseBody
	if signed == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "No release is published yet.", "code": "no_release"})
		return
	}
	header := w.Header()
	header.Set("ETag", signed.ETag)
	// Shared caches (Cloudflare) may keep it for 5 minutes; a new release shows within that.
	header.Set("Cache-Control", "public, max-age=300")
	if r.Header.Get("If-None-Match") == signed.ETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	header.Set("Content-Type", "application/json")
	header.Set("Content-Length", strconv.Itoa(len(signed.Body)))
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		w.Write(signed.Body)
	}
}
