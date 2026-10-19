package platformapi

import (
	"errors"
	"math/big"
	"net/http"

	"studio-engine/internal/httpapi"
	"studio-engine/internal/pyjson"
)

// silentJSON is request.get_json(silent=True): nil (None) when the body is not JSON; only a
// body above Flask's size limit is an error (413).
func silentJSON(r *http.Request) (any, error) {
	return httpapi.ReadJSON(r, true)
}

// bodyOrEmpty is api/network.body(): request.get_json(silent=True) or {}.
func bodyOrEmpty(r *http.Request) (any, error) {
	value, err := silentJSON(r)
	if err != nil || truthy(value) {
		return value, err
	}
	return pyjson.NewObject(), nil
}

// dictBody is api/license._body(): the JSON object, or {} for anything else.
func dictBody(r *http.Request) (*pyjson.Object, error) {
	value, err := silentJSON(r)
	if err != nil {
		return nil, err
	}
	if object, ok := value.(*pyjson.Object); ok {
		return object, nil
	}
	return pyjson.NewObject(), nil
}

// errNoGet is the AttributeError of calling .get on a decoded body that is not a dict;
// Flask answered it with its generic 500.
var errNoGet = errors.New("AttributeError: object has no attribute 'get'")

// get is value.get(key, fallback) on a decoded request body.
func get(value any, key string, fallback any) (any, error) {
	object, ok := value.(*pyjson.Object)
	if !ok {
		return nil, errNoGet
	}
	if item, found := object.Get(key); found {
		return item, nil
	}
	return fallback, nil
}

// pair reads two keys of a body with .get (None when missing), in Python's order.
func pair(value any, first, second string) (any, any, error) {
	a, err := get(value, first, nil)
	if err != nil {
		return nil, nil, err
	}
	b, err := get(value, second, nil)
	return a, b, err
}

// truthy is Python's bool() of a decoded JSON value.
func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case *big.Int:
		return v.Sign() != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case *pyjson.Object:
		return v != nil && v.Len() > 0
	}
	return true
}
