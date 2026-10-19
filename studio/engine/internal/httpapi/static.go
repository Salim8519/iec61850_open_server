package httpapi

import (
	"hash/adler32"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
)

// Frontend serves the browser UI from frontend.pak (memory) or from a dist folder, as
// launcher/frontend_assets.py FrontendAssets does.
type Frontend struct {
	folder string
	files  map[string][]byte
}

// LoadFrontend opens location: a .pak file, or a folder holding frontend.pak or the dist
// files (index.html and assets/).
func LoadFrontend(location string) (*Frontend, error) {
	pack := location
	if !strings.EqualFold(filepath.Ext(location), ".pak") {
		pack = filepath.Join(location, "frontend.pak")
	}
	if info, err := os.Stat(pack); err == nil && info.Mode().IsRegular() {
		blob, err := os.ReadFile(pack)
		if err != nil {
			return nil, err
		}
		files, err := unpackFrontend(blob)
		if err != nil {
			return nil, err
		}
		return &Frontend{folder: location, files: files}, nil
	}
	return &Frontend{folder: location}, nil
}

func (f *Frontend) available() bool {
	if f == nil {
		return false
	}
	if f.files != nil {
		_, found := f.files["index.html"]
		return found
	}
	_, err := os.Stat(filepath.Join(f.folder, "index.html"))
	return err == nil
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	front := s.config.Frontend
	if !front.available() {
		WriteBody(w, http.StatusServiceUnavailable, "text/html; charset=utf-8",
			[]byte("ScadaBench frontend build is not available yet."))
		return
	}
	if front.files != nil {
		front.memory(w, "index.html")
		return
	}
	front.send(w, r, front.folder, "index.html")
}

func (s *Server) asset(w http.ResponseWriter, r *http.Request) {
	front, name := s.config.Frontend, r.PathValue("filename")
	switch {
	case front == nil:
		WriteError(w, ErrNotFound)
	case front.files != nil:
		front.memory(w, "assets/"+name)
	default:
		front.send(w, r, filepath.Join(front.folder, "assets"), name)
	}
}

// memory answers a packed file: index.html no-cache, the rest immutable for a year.
func (f *Frontend) memory(w http.ResponseWriter, name string) {
	content, found := f.files[name]
	if !found {
		WriteBody(w, http.StatusNotFound, "text/plain; charset=utf-8", []byte("Not found"))
		return
	}
	cache := "public, max-age=31536000, immutable"
	if name == "index.html" {
		cache = "no-cache"
	}
	w.Header().Set("Cache-Control", cache)
	WriteBody(w, http.StatusOK, contentType(packedMime(name)), content)
}

// send is Flask's send_from_directory: safe join (no "..", no absolute or drive parts),
// a regular file or 404, then a conditional response with Werkzeug's ETag.
func (f *Frontend) send(w http.ResponseWriter, r *http.Request, directory, name string) {
	full, ok := safeJoin(directory, name)
	if !ok {
		WriteError(w, ErrNotFound)
		return
	}
	file, err := os.Open(full)
	if err != nil {
		WriteError(w, ErrNotFound)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		WriteError(w, ErrNotFound)
		return
	}
	// os.stat's st_mtime float: whole seconds plus nanoseconds * 1e-9 (posixmodule fill_time).
	modified := info.ModTime()
	mtime := float64(modified.Unix()) + float64(modified.Nanosecond())*1e-9
	etag := pyjson.Repr(mtime) + "-" + strconv.FormatInt(info.Size(), 10) + "-" +
		strconv.FormatUint(uint64(adler32.Checksum([]byte(full))), 10)
	header := w.Header()
	header.Set("Content-Type", contentType(guessMime(name)))
	header.Set("Cache-Control", "no-cache")
	header.Set("ETag", `"`+etag+`"`)
	header.Set("Content-Disposition", ContentDisposition("inline", path.Base(full)))
	http.ServeContent(w, r, "", info.ModTime(), file)
}

// safeJoin is werkzeug.security.safe_join with posixpath.join, as Flask builds the path.
func safeJoin(directory, name string) (string, bool) {
	if name == "" {
		return "", false
	}
	cleaned := path.Clean(name)
	if strings.ContainsAny(cleaned, `\:`) || path.IsAbs(cleaned) || cleaned == ".." ||
		strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return directory + "/" + cleaned, true
}

func packedMime(name string) string {
	if strings.HasSuffix(name, ".js") {
		return "text/javascript"
	}
	return guessMime(name)
}

// guessMime is mimetypes.guess_type: Go and CPython both read the Windows registry's
// "Content Type" values over their built-in tables.
func guessMime(name string) string {
	guessed, _, _ := strings.Cut(mime.TypeByExtension(path.Ext(name)), ";")
	if guessed == "" {
		return "application/octet-stream"
	}
	return strings.TrimSpace(guessed)
}

// contentType adds "; charset=utf-8" as Werkzeug's get_content_type does.
func contentType(mimetype string) string {
	switch {
	case strings.HasPrefix(mimetype, "text/"), strings.HasSuffix(mimetype, "+xml"),
		mimetype == "application/ecmascript", mimetype == "application/javascript",
		mimetype == "application/sql", mimetype == "application/xml",
		mimetype == "application/xml-dtd", mimetype == "application/xml-external-parsed-entity":
		return mimetype + "; charset=utf-8"
	}
	return mimetype
}
