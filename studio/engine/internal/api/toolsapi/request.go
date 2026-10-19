package toolsapi

import (
	"bytes"
	"io"
	"math"
	"math/big"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// maxBody is Flask's MAX_CONTENT_LENGTH (100 MiB).
const maxBody = 100 * 1024 * 1024

const badRequest = "The browser (or proxy) sent a request that this server could not understand."

// isJSON is Werkzeug's Request.is_json: application/json or application/*+json.
func isJSON(r *http.Request) bool {
	mimetype, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return mimetype == "application/json" || (strings.HasPrefix(mimetype, "application/") && strings.HasSuffix(mimetype, "+json"))
}

// loadJSON reads and decodes the body like Flask's json provider (UTF-8, a BOM ignored).
func loadJSON(r *http.Request) (any, bool) {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil || len(data) > maxBody {
		return nil, false
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if !utf8.Valid(data) {
		return nil, false
	}
	value, err := pyjson.Decode(data)
	return value, err == nil
}

// silentJSON is request.get_json(silent=True): None for a wrong type or a parse error.
func silentJSON(r *http.Request) any {
	if !isJSON(r) {
		return nil
	}
	value, ok := loadJSON(r)
	if !ok {
		return nil
	}
	return value
}

// strictJSON is request.get_json(): 415 without a JSON type, 400 on a parse error.
func strictJSON(r *http.Request) (any, error) {
	if !isJSON(r) {
		return nil, &httpError{http.StatusUnsupportedMediaType,
			"Did not attempt to load JSON data because the request Content-Type was not 'application/json'."}
	}
	value, ok := loadJSON(r)
	if !ok {
		return nil, &httpError{http.StatusBadRequest, badRequest}
	}
	return value, nil
}

// body is the handlers' body(): a JSON object, else ValueError "Send a JSON object".
func body(r *http.Request) (*pyjson.Object, error) {
	object, ok := silentJSON(r).(*pyjson.Object)
	if !ok {
		return nil, apperr.ValueError("Send a JSON object")
	}
	return object, nil
}

// value is data.get(key, fallback).
func value(data *pyjson.Object, key string, fallback any) any {
	if found, ok := data.Get(key); ok {
		return found
	}
	return fallback
}

// query is request.args.get(name, "") (the first value).
func query(r *http.Request, name string) (string, bool) {
	values, found := r.URL.Query()[name]
	if !found || len(values) == 0 {
		return "", false
	}
	return values[0], true
}

// afterArg is int(request.args.get("after", 0)), clamped to int64 for the comparison.
func afterArg(r *http.Request) (int64, error) {
	text, found := query(r, "after")
	if !found {
		return 0, nil
	}
	parsed, err := pytext.ParseInt(text)
	if err != nil {
		return 0, apperr.ValueError(err.Error())
	}
	if large, ok := parsed.(*big.Int); ok {
		if large.Sign() > 0 {
			return math.MaxInt64, nil
		}
		return math.MinInt64, nil
	}
	return parsed.(int64), nil
}
