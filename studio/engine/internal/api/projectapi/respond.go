package projectapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
)

// Flask/Werkzeug texts for request errors these routes can raise.
const (
	badRequestText  = "The browser (or proxy) sent a request that this server could not understand."
	notJSONText     = "Did not attempt to load JSON data because the request Content-Type was not 'application/json'."
	internalErrText = "The server encountered an internal error and was unable to complete your request. " +
		"Either the server is overloaded or there is an error in the application."
)

// httpError is a Werkzeug HTTPException: {"error": description} with its status.
type httpError struct {
	status      int
	description string
}

func (e *httpError) Error() string { return e.description }

// writeJSON answers like flask.jsonify (sorted keys, compact, trailing newline).
func writeJSON(w http.ResponseWriter, status int, value any) { httpapi.WriteJSON(w, status, value) }

// writeError answers with app_factory.py's error handlers. httpapi (stream A) owns the
// shared version; this mirror keeps the routes testable on their own.
func writeError(w http.ResponseWriter, err error) {
	var app *apperr.Error
	var contract *project.ContractError
	var request *httpError
	switch {
	case errors.As(err, &request):
		writeJSON(w, request.status, pyjson.NewObject().Set("error", request.description))
	case errors.As(err, &contract):
		writeJSON(w, 400, pyjson.NewObject().Set("error", "Invalid request").Set("details", contract.Details))
	case errors.As(err, &app) && app.Kind == apperr.Studio:
		status := 400
		if app.Code == "conflict" {
			status = 409
		}
		details := app.Details
		if details == nil {
			details = []any{}
		}
		writeJSON(w, status, pyjson.NewObject().Set("error", app.Message).Set("code", app.Code).Set("details", details))
	case errors.As(err, &app) && app.Kind == apperr.Runtime:
		writeJSON(w, 503, pyjson.NewObject().Set("error", app.Message))
	case errors.As(err, &app):
		writeJSON(w, 400, pyjson.NewObject().Set("error", app.Message))
	default:
		writeJSON(w, 500, pyjson.NewObject().Set("error", internalErrText))
	}
}

// sendFile is flask.send_file(BytesIO(data), mimetype, as_attachment=True, download_name);
// Range requests are answered in full (Python would answer 206; the UI never sends Range).
func sendFile(w http.ResponseWriter, r *http.Request, data []byte, mimetype, downloadName string) {
	if mimetype == "application/xml" {
		mimetype += "; charset=utf-8"
	}
	header := w.Header()
	header.Set("Content-Type", mimetype)
	header.Set("Content-Disposition", contentDisposition(downloadName))
	header.Set("Content-Length", strconv.Itoa(len(data)))
	header.Set("Cache-Control", "no-cache")
	if r.Method == "GET" || r.Method == "HEAD" { // werkzeug make_conditional
		header.Set("Accept-Ranges", "bytes")
	}
	w.WriteHeader(200)
	_, _ = w.Write(data)
}

// contentDisposition follows werkzeug send_file: ASCII names as a token or quoted string,
// other names with an ASCII fallback and an RFC 5987 filename*.
func contentDisposition(name string) string {
	ascii := true
	for _, r := range name {
		if r >= utf8.RuneSelf {
			ascii = false
			break
		}
	}
	if ascii {
		return "attachment; filename=" + quoteHeader(name)
	}
	return "attachment; filename=" + quoteHeader(asciiFallback(name)) + "; filename*=UTF-8''" + rfc5987(name)
}

func quoteHeader(value string) string {
	const token = "!#$%&'*+-.^_`|~"
	plain := value != ""
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(token, r)) {
			plain = false
			break
		}
	}
	if plain {
		return value
	}
	return `"` + strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`) + `"`
}

func rfc5987(value string) string {
	const safe = "!#$&+-.^_`|~"
	var b strings.Builder
	for _, c := range []byte(value) {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.IndexByte(safe, c) >= 0 {
			b.WriteByte(c)
			continue
		}
		b.WriteString("%" + strings.ToUpper(strconv.FormatInt(int64(c)+0x100, 16)[1:]))
	}
	return b.String()
}
