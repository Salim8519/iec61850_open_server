package api

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"net/http"
	"path"
)

//go:embed pages/*
var pageFiles embed.FS

var pageTemplates = template.Must(template.ParseFS(pageFiles, "pages/*.html"))

type pageData struct {
	Product       string
	Price         string
	Currency      string
	ClientID      string
	CheckoutReady bool
	Support       string
	Transfers     int
	TrialDays     int
	Seller        string
	Asset         string
}

// assetVersion changes whenever the embedded CSS or JS changes, so buyers never get
// an old stylesheet from their browser cache after an update.
var assetVersion = func() string {
	hash := sha256.New()
	for _, name := range []string{"pages/style.css", "pages/common.js"} {
		raw, _ := pageFiles.ReadFile(name)
		hash.Write(raw)
	}
	return hex.EncodeToString(hash.Sum(nil))[:10]
}()

// Amount is the price as buyers read it: "$149.00" for USD, otherwise "149.00 EUR".
func (d pageData) Amount() string {
	if d.Currency == "USD" {
		return "$" + d.Price
	}
	return d.Price + " " + d.Currency
}

func (s *Server) data() pageData {
	return pageData{Product: s.Config.ProductName, Price: s.Config.LicensePrice,
		Currency: s.Config.LicenseCurrency, ClientID: s.Config.PayPalClientID,
		CheckoutReady: s.Config.PayPalReady() && s.Payments != nil, Support: s.Config.SupportEmail,
		Transfers: s.Config.MaxTransfersPerYear, TrialDays: s.Config.TrialDays,
		Seller: s.Config.SellerName, Asset: assetVersion}
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, name string) {
	if !s.Limiter.Allow(s.clientIP(r), 0, s.Now()) {
		http.Error(w, "Too many requests. Wait a few minutes.", http.StatusTooManyRequests)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	pageHeaders(w)
	if err := pageTemplates.ExecuteTemplate(w, name, s.data()); err != nil {
		s.Log.Event("template_error", map[string]any{"page": name, "error": err.Error()})
	}
}

func (s *Server) buyPage(w http.ResponseWriter, r *http.Request) { s.render(w, r, "buy.html") }
func (s *Server) activatePage(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "activate.html")
}

// Fonts are Geist (SIL OFL 1.1, see pages/FONTS-OFL.txt), the same faces as scadabench.com.
var staticTypes = map[string]string{".js": "text/javascript; charset=utf-8",
	".css": "text/css; charset=utf-8", ".woff2": "font/woff2"}

func staticHandler() http.Handler {
	files := http.FileServer(http.FS(pageFiles))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		contentType, allowed := staticTypes[path.Ext(name)]
		if !allowed || (name != "common.js" && name != "style.css" && path.Ext(name) != ".woff2") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=604800")
		r.URL.Path = "/pages/" + name
		files.ServeHTTP(w, r)
	})
}

// product tells scadabench.com the live price, so a price change shows everywhere
// without rebuilding the website. Only the site's own origins may read it.
func (s *Server) product(w http.ResponseWriter, r *http.Request) {
	if !s.Limiter.Allow(s.clientIP(r), 0, s.Now()) {
		http.Error(w, "Too many requests.", http.StatusTooManyRequests)
		return
	}
	switch origin := r.Header.Get("Origin"); origin {
	case "https://scadabench.com", "https://www.scadabench.com", "https://scadabench.pages.dev":
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	d := s.data()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	json.NewEncoder(w).Encode(map[string]any{"price": d.Price, "currency": d.Currency, "amount": d.Amount(),
		"trial_days": d.TrialDays, "checkout": d.CheckoutReady})
}
