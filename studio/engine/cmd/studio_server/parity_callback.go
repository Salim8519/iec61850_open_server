//go:build parity

package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"studio-engine/internal/pyjson"
)

// callback is a <callback> reference of the seams file ({"url", "key", "id"}): a fake that
// lives in the test process (tests/parity/callbacks.py). call runs it: POST <url>/call
// {"id", "args", "kwargs"} answers {"return": v} | {"raise": exception} | {"original": true}.
type callback struct {
	URL string `json:"url"`
	Key string `json:"key"`
	ID  string `json:"id"`
}

// errOriginal is the {"original": true} answer: the fake asked the engine to run its own code.
var errOriginal = errors.New("parity fake asked for the original")

// wirePath and wireHandle are parity/wire's {"__path__"} and {"__handle__"} tags.
type (
	wirePath   string
	wireHandle string
)

// remoteError is an exception the fake raised (wire.encode_exception).
type remoteError struct {
	Type, Module string
	Args         []any
	HTTPCode     int64 // urllib HTTPError code; 0 for other exceptions
}

func (e *remoteError) Error() string {
	if len(e.Args) == 1 {
		if text, ok := e.Args[0].(string); ok {
			return text
		}
	}
	return e.Type
}

var callbackClient = &http.Client{Timeout: 120 * time.Second, Transport: &http.Transport{Proxy: nil}}

func (c callback) valid() bool { return c.URL != "" && c.ID != "" }

// call runs the fake with these arguments; it returns errOriginal, a *remoteError or the
// decoded return value.
func (c callback) call(args ...any) (any, error) {
	payload := pyjson.NewObject().Set("id", c.ID).Set("args", wireEncode(args)).Set("kwargs", pyjson.NewObject())
	body, err := pyjson.Dumps(payload)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", c.URL+"/call", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Parity-Key", c.Key)
	response, err := callbackClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("parity callback %s answered %d", c.ID, response.StatusCode)
	}
	decoded, err := pyjson.Decode(data)
	if err != nil {
		return nil, err
	}
	answer, _ := decoded.(*pyjson.Object)
	if original, _ := field(answer, "original").(bool); original {
		return nil, errOriginal
	}
	if raised, found := answer.Get("raise"); found {
		return nil, decodeException(raised)
	}
	return wireDecode(field(answer, "return")), nil
}

// wireEncode is wire.encode for the Go values the seams hand to a fake.
func wireEncode(value any) any {
	switch v := value.(type) {
	case wireBytes:
		return pyjson.NewObject().Set("__bytes__", base64.StdEncoding.EncodeToString(v))
	case wirePath:
		return pyjson.NewObject().Set("__path__", string(v))
	case wireHandle:
		return pyjson.NewObject().Set("__handle__", string(v))
	case []string:
		items := make([]any, len(v))
		for i, item := range v {
			items[i] = item
		}
		return items
	case []any:
		items := make([]any, len(v))
		for i, item := range v {
			items[i] = wireEncode(item)
		}
		return items
	case map[string]string:
		object := pyjson.NewObject()
		for key, item := range v {
			object.Set(key, item)
		}
		return object
	case int:
		return int64(v)
	}
	return value
}

// wireDecode is wire.decode: tags become Go values (a set or tuple becomes a list).
func wireDecode(value any) any {
	switch v := value.(type) {
	case []any:
		for i, item := range v {
			v[i] = wireDecode(item)
		}
	case *pyjson.Object:
		if v.Len() == 1 {
			key := v.Keys()[0]
			inner, _ := v.Get(key)
			switch key {
			case "__bytes__":
				text, _ := inner.(string)
				data, _ := base64.StdEncoding.DecodeString(text)
				return wireBytes(data)
			case "__set__", "__tuple__":
				return wireDecode(inner)
			case "__path__":
				text, _ := inner.(string)
				return wirePath(text)
			case "__handle__":
				text, _ := inner.(string)
				return wireHandle(text)
			case "__exc__":
				return decodeException(inner)
			}
		}
		for _, key := range v.Keys() {
			item, _ := v.Get(key)
			v.Set(key, wireDecode(item))
		}
	}
	return value
}

func decodeException(value any) *remoteError {
	record, _ := value.(*pyjson.Object)
	failure := &remoteError{Type: "RuntimeError", Module: "builtins"}
	if record == nil {
		return failure
	}
	failure.Type, _ = field(record, "type").(string)
	failure.Module, _ = field(record, "module").(string)
	failure.Args, _ = wireDecode(field(record, "args")).([]any)
	if code, ok := field(field(record, "http"), "code").(int64); ok {
		failure.HTTPCode = code
	}
	return failure
}
