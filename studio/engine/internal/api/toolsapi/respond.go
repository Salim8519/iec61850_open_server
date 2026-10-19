// Package toolsapi serves the Tools place routes (IED Explorer, Find IEDs, tool memory,
// activity, station gateway, SCADA master/converter), the test panel and the CSV trace
// preview with Python's exact paths, JSON bytes (Flask jsonify) and error texts.
//
// Errors are answered here as app_factory.py maps them; when stream A's httpapi offers a
// shared error writer, handle() should delegate to it (see streams/e-tools.md).
package toolsapi

import (
	"errors"
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// internalError is Werkzeug's InternalServerError description (any unexpected exception).
const internalError = "The server encountered an internal error and was unable to complete your request. " +
	"Either the server is overloaded or there is an error in the application."

// httpError is a Werkzeug HTTPException: its status and description, no "code" key.
type httpError struct {
	status int
	text   string
}

func (e *httpError) Error() string { return e.text }

// answer is a body with a status other than 200 (handlers that return (jsonify, 400)).
type answer struct {
	status int
	body   any
}

// download is a flask Response with a text body (CSV downloads).
type download struct {
	body        string
	contentType string
	disposition string
}

// handle wraps a handler that returns a JSON value, an *answer, a *download or an error.
func handle(fn func(*http.Request) (any, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value, err := run(fn, r)
		if err != nil {
			status, body := errorBody(err)
			writeJSON(w, status, body)
			return
		}
		switch v := value.(type) {
		case *download:
			w.Header().Set("Content-Type", v.contentType)
			w.Header().Set("Content-Disposition", v.disposition)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(v.body))
		case *answer:
			writeJSON(w, v.status, v.body)
		default:
			writeJSON(w, http.StatusOK, value)
		}
	})
}

// run turns a panic (Python's unexpected exception) into an internal error.
func run(fn func(*http.Request) (any, error), r *http.Request) (value any, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			value, err = nil, errors.New("internal error")
		}
	}()
	return fn(r)
}

func errorBody(err error) (int, any) {
	var http_ *httpError
	if errors.As(err, &http_) {
		return http_.status, pyjson.NewObject().Set("error", http_.text)
	}
	var studio *apperr.Error
	if errors.As(err, &studio) {
		switch studio.Kind {
		case apperr.Studio:
			status := http.StatusBadRequest
			if studio.Code == "conflict" {
				status = http.StatusConflict
			}
			details := studio.Details
			if details == nil || isEmpty(details) {
				details = []any{}
			}
			return status, pyjson.NewObject().Set("code", studio.Code).Set("details", details).Set("error", studio.Message)
		case apperr.Value, apperr.Key:
			return http.StatusBadRequest, pyjson.NewObject().Set("error", studio.Message)
		case apperr.Runtime:
			return http.StatusServiceUnavailable, pyjson.NewObject().Set("error", studio.Message)
		}
	}
	return http.StatusInternalServerError, pyjson.NewObject().Set("error", internalError)
}

func isEmpty(value any) bool {
	switch v := value.(type) {
	case *pyjson.Object:
		return v.Len() == 0
	case []any:
		return len(v) == 0
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := pyjson.Flask(value)
	if err != nil {
		status = http.StatusInternalServerError
		body, _ = pyjson.Flask(pyjson.NewObject().Set("error", internalError))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(body)
}
