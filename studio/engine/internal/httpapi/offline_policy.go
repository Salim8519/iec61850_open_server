package httpapi

import "net/http"

// OfflinePolicy is the Content-Security-Policy of the ScadaBench page (ADR-0133): the browser
// may load scripts, styles, images, fonts and API answers only from this engine, so the
// interface works without internet and can never fetch anything from it, even by mistake.
// Inline styles stay allowed (React style attributes); links the user clicks still open.
// studio/frontend/vite.config.ts serves the same policy in development, plus Vite's reload.
const OfflinePolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; object-src 'none'; " +
	"frame-src 'none'; worker-src 'self' blob:; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

func setOfflinePolicy(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", OfflinePolicy)
}
