package httpapi

import (
	"net/http"
	"strconv"

	"studio-engine/internal/pyjson"
)

// WriteJSON answers like `return jsonify(value), status`: Flask's sorted compact body with
// its final newline, Content-Type application/json and Content-Length.
func WriteJSON(w http.ResponseWriter, status int, value any) {
	body, err := pyjson.Flask(value)
	if err != nil {
		writeInternalError(w)
		return
	}
	WriteBody(w, status, "application/json", body)
}

// WriteBody writes a complete response with its Content-Length (also for HEAD, as Flask).
func WriteBody(w http.ResponseWriter, status int, contentType string, body []byte) {
	header := w.Header()
	header.Set("Content-Type", contentType)
	header.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// errorBody is jsonify(error=message).
func errorBody(message string) *pyjson.Object {
	return pyjson.NewObject().Set("error", message)
}

// writeError writes jsonify(error=message), status.
func writeMessage(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, errorBody(message))
}
