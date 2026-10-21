// Package runtimeapi serves the runtime routes of the Python blueprints runtime.py,
// equipment.py, scenarios.py (runs and start), signals.py (update, source, controls/operate)
// and test_panel.py (pulse, release-override, GET test-panel) over the Go runtime.
package runtimeapi

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Flask/Werkzeug texts the handlers can produce (http-conventions.md §4).
const (
	internalText    = "The server encountered an internal error and was unable to complete your request. Either the server is overloaded or there is an error in the application."
	badRequestText  = "The browser (or proxy) sent a request that this server could not understand."
	unsupportedText = "Did not attempt to load JSON data because the request Content-Type was not 'application/json'."
)

// httpError is a Werkzeug HTTPException: {"error": description} with its status.
type httpError struct {
	status int
	text   string
}

func (e *httpError) Error() string { return e.text }

// handler returns the jsonify-ed body and status, or an error mapped like app_factory.py.
type handler func(r *http.Request) (any, int, error)

// serve adapts a handler. It writes Flask's error bodies itself so the area is complete
// before stream A's httpapi exists; httpapi may wrap the same apperr values the same way.
func serve(next handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, status, err := safely(next, r)
		if err != nil {
			body, status = errorBody(err)
		}
		httpapi.WriteJSON(w, status, body)
	})
}

func safely(next handler, r *http.Request) (body any, status int, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			body, status, err = nil, 0, errors.New("internal error")
		}
	}()
	return next(r)
}

func errorBody(err error) (any, int) {
	var failure *apperr.Error
	if errors.As(err, &failure) {
		switch failure.Kind {
		case apperr.Studio:
			status := http.StatusBadRequest
			if failure.Code == "conflict" {
				status = http.StatusConflict
			}
			details := failure.Details
			if details == nil || isEmpty(details) {
				details = []any{}
			}
			return pyjson.NewObject().Set("error", failure.Message).Set("code", failure.Code).Set("details", details), status
		case apperr.Runtime:
			return pyjson.NewObject().Set("error", failure.Message), http.StatusServiceUnavailable
		default:
			return pyjson.NewObject().Set("error", failure.Message), http.StatusBadRequest
		}
	}
	var web *httpError
	if errors.As(err, &web) {
		return pyjson.NewObject().Set("error", web.text), web.status
	}
	return pyjson.NewObject().Set("error", internalText), http.StatusInternalServerError
}

func isEmpty(value any) bool {
	switch typed := value.(type) {
	case *pyjson.Object:
		return typed.Len() == 0
	case []any:
		return len(typed) == 0
	}
	return false
}

func readBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	return io.ReadAll(r.Body)
}

func isJSON(r *http.Request) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}
	return media == "application/json" || (strings.HasPrefix(media, "application/") && strings.HasSuffix(media, "+json"))
}

// getJSON is request.get_json(): 415 without a JSON content type, 400 for a bad body.
func getJSON(r *http.Request) (any, error) {
	if !isJSON(r) {
		return nil, &httpError{http.StatusUnsupportedMediaType, unsupportedText}
	}
	raw, err := readBody(r)
	if err != nil {
		return nil, &httpError{http.StatusBadRequest, badRequestText}
	}
	value, err := pyjson.Decode(raw)
	if err != nil {
		return nil, &httpError{http.StatusBadRequest, badRequestText}
	}
	return value, nil
}

// getJSONSilent is request.get_json(silent=True) or {}.
func getJSONSilent(r *http.Request) any {
	value, err := getJSON(r)
	if err != nil || !pyval.Truthy(value) {
		return pyjson.NewObject()
	}
	return value
}

// field is data[key]: KeyError (400) when missing; a body that is not an object fails like
// Python's subscript of a list or None (TypeError, Flask's 500).
func field(data any, key string) (any, error) {
	object, ok := data.(*pyjson.Object)
	if !ok {
		return nil, errors.New("TypeError")
	}
	value, present := object.Get(key)
	if !present {
		return nil, apperr.KeyError(key)
	}
	return value, nil
}

// optional is data.get(key, fallback) on a JSON object (AttributeError → 500 otherwise).
func optional(data any, key string, fallback any) (any, error) {
	object, ok := data.(*pyjson.Object)
	if !ok {
		return nil, errors.New("AttributeError")
	}
	if value, present := object.Get(key); present {
		return value, nil
	}
	return fallback, nil
}

// text is a field used as a dict key or IED id: Python compares it with stored strings, so a
// non-string never matches; "\x00" keeps that behaviour.
func text(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return "\x00" + pyval.Repr(value)
}
