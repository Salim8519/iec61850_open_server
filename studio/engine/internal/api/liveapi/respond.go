package liveapi

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/pyjson"
)

// internalError is Werkzeug's InternalServerError description (http-conventions §4).
const internalError = "The server encountered an internal error and was unable to complete your request. " +
	"Either the server is overloaded or there is an error in the application."

// errNotObject stands for Python's AttributeError when a JSON body is a truthy non-object
// and the handler calls .get on it (500).
var errNotObject = errors.New("'list' object has no attribute 'get'")

// writeJSON answers like Flask's jsonify (httpapi's shared writer).
func writeJSON(w http.ResponseWriter, status int, value any) { httpapi.WriteJSON(w, status, value) }

// writeDownload answers a text attachment (Response(text, mimetype=...) with the header).
func writeDownload(w http.ResponseWriter, mimetype, disposition, text string) {
	w.Header().Set("Content-Disposition", disposition)
	httpapi.WriteBody(w, http.StatusOK, mimetype+"; charset=utf-8", []byte(text))
}

// writeError maps a Python-class error to Flask's status and body (app_factory.py:76-97).
// The integrating session may route this through httpapi's shared error writer instead;
// the bodies are the same.
func writeError(w http.ResponseWriter, err error) {
	var studio *apperr.Error
	if !errors.As(err, &studio) {
		writeJSON(w, http.StatusInternalServerError, pyv.O("error", internalError))
		return
	}
	switch studio.Kind {
	case apperr.Studio:
		details := studio.Details
		if !pyv.Truthy(details) {
			details = []any{}
		}
		status := http.StatusBadRequest
		if studio.Code == "conflict" {
			status = http.StatusConflict
		}
		writeJSON(w, status, pyv.O("code", studio.Code, "details", details, "error", studio.Message))
	case apperr.Runtime:
		writeJSON(w, http.StatusServiceUnavailable, pyv.O("error", studio.Message))
	default:
		writeJSON(w, http.StatusBadRequest, pyv.O("error", studio.Message))
	}
}

// silentJSON is request.get_json(silent=True): the decoded body when the request says it is
// JSON and parses, otherwise nil.
func silentJSON(r *http.Request) any {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType != "application/json" && !(strings.HasPrefix(mediaType, "application/") &&
		strings.HasSuffix(mediaType, "+json")) {
		return nil
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil
	}
	value, err := pyjson.Decode(data)
	if err != nil {
		return nil
	}
	return value
}

// body is `request.get_json(silent=True) or {}` used with .get: an object, or the 500 Python
// raised for a truthy non-object.
func body(r *http.Request) (*pyjson.Object, error) {
	value := silentJSON(r)
	if !pyv.Truthy(value) {
		return pyjson.NewObject(), nil
	}
	if object, ok := value.(*pyjson.Object); ok {
		return object, nil
	}
	return nil, errNotObject
}
