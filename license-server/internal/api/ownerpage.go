package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"path"

	"substation-license-server/internal/security"
)

// ownerAssets are the only files the owner page serves; they come from the binary.
var ownerAssets = map[string]string{
	"admin.css": "text/css; charset=utf-8", "admin-core.js": "text/javascript; charset=utf-8",
	"admin-views.js": "text/javascript; charset=utf-8", "admin-ops.js": "text/javascript; charset=utf-8",
	"geist.woff2": "font/woff2", "geist-mono.woff2": "font/woff2",
}

// ownerAssetVersion changes with the owner page files, so a browser never runs an
// old script against a new server after a deploy.
var ownerAssetVersion = func() string {
	hash := sha256.New()
	for _, name := range []string{"admin.css", "admin-core.js", "admin-views.js", "admin-ops.js"} {
		raw, _ := pageFiles.ReadFile("pages/" + name)
		hash.Write(raw)
	}
	return hex.EncodeToString(hash.Sum(nil))[:10]
}()

type ownerPageData struct {
	Base  string
	Asset string
}

func (s *Server) ownerPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTemplates.ExecuteTemplate(w, "admin.html", ownerPageData{Base: s.Config.AdminPath, Asset: ownerAssetVersion}); err != nil {
		s.Log.Event("template_error", map[string]any{"page": "admin", "error": err.Error()})
	}
}

func (s *Server) ownerStatic(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	contentType, ok := ownerAssets[name]
	if !ok {
		ownerNotFound(w)
		return
	}
	raw, err := pageFiles.ReadFile(path.Join("pages", name))
	if err != nil {
		ownerNotFound(w)
		return
	}
	// "private" keeps Cloudflare from caching anything behind the gateway secret.
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("Content-Type", contentType)
	w.Write(raw)
}

type unsubscribeData struct {
	pageData
	Token string
	Done  bool
}

// unsubscribePage confirms with a button (GET never changes anything, because mail
// scanners open links); the POST also serves one-click unsubscribe from mail apps.
func (s *Server) unsubscribePage(w http.ResponseWriter, r *http.Request) {
	if !s.Limiter.Allow(s.clientIP(r), security.ClassGeneral, s.Now()) {
		http.Error(w, "Too many requests. Wait a few minutes.", http.StatusTooManyRequests)
		return
	}
	data := unsubscribeData{pageData: s.data(), Token: r.URL.Query().Get("t")}
	if r.Method == http.MethodPost {
		if _, err := s.Store.Unsubscribe(data.Token, s.Now()); err != nil {
			s.Log.Event("internal_error", map[string]any{"path": "/unsubscribe", "error": err.Error()})
			http.Error(w, "Please try again in a minute.", http.StatusInternalServerError)
			return
		}
		data.Done = true
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	pageHeaders(w)
	if err := pageTemplates.ExecuteTemplate(w, "unsubscribe.html", data); err != nil {
		s.Log.Event("template_error", map[string]any{"page": "unsubscribe", "error": err.Error()})
	}
}
