//go:build parity

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"studio-engine/internal/app"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// parityOp is one /api/v1/_parity/ operation (tests/parity/inspect_ops.py); its value is
// encoded as parity/wire tagged JSON.
type parityOp func(e *app.Engine, body *pyjson.Object, r *http.Request) (any, error)

// wireBytes is wire.encode(bytes).
type wireBytes []byte

var parityOps = map[string]parityOp{
	"GET project/raw":      func(e *app.Engine, _ *pyjson.Object, _ *http.Request) (any, error) { return e.Projects.Current(), nil },
	"GET project/snapshot": func(e *app.Engine, _ *pyjson.Object, _ *http.Request) (any, error) { return e.Projects.Snapshot(), nil },
	"GET project/revision": func(e *app.Engine, _ *pyjson.Object, _ *http.Request) (any, error) { return e.Projects.Revision(), nil },
	"GET project/export": func(e *app.Engine, _ *pyjson.Object, _ *http.Request) (any, error) {
		data, err := e.Projects.Export()
		return wireBytes(data), err
	},
	"POST project/add_template": func(e *app.Engine, body *pyjson.Object, _ *http.Request) (any, error) {
		name := "BayController"
		if args := argList(body); len(args) > 0 {
			name, _ = args[0].(string)
		}
		models, err := e.Projects.AddTemplate(name)
		list := make([]any, len(models))
		for i, model := range models {
			list[i] = model
		}
		return list, err
	},
	"POST project/restore": func(e *app.Engine, body *pyjson.Object, _ *http.Request) (any, error) {
		args, persist := argList(body), true
		if len(args) > 1 {
			persist, _ = args[1].(bool)
		}
		if value, found := kwargs(body).Get("persist"); found {
			persist, _ = value.(bool)
		}
		if len(args) == 0 {
			return nil, apperr.ValueError("restore() needs the archive bytes")
		}
		payload, _ := args[0].(wireBytes)
		return nil, e.Projects.Restore(payload, persist)
	},
	"POST project/scenarios": func(e *app.Engine, body *pyjson.Object, _ *http.Request) (any, error) {
		if args := argList(body); len(args) > 0 {
			return nil, e.Projects.Scenarios(args[0])
		}
		return nil, apperr.ValueError("scenarios() needs a value")
	},
	"GET events": func(e *app.Engine, _ *pyjson.Object, r *http.Request) (any, error) {
		cursor, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
		items := []any{}
		for _, event := range e.Bus.After(cursor, 0, nil) {
			decoded, err := pyjson.Decode(event.Data)
			if err != nil {
				return nil, err
			}
			items = append(items, decoded)
		}
		return items, nil
	},
	"GET runtime/state": func(e *app.Engine, _ *pyjson.Object, _ *http.Request) (any, error) { return e.Runtime.State(), nil },
	"POST runtime/close": func(e *app.Engine, _ *pyjson.Object, _ *http.Request) (any, error) {
		e.Runtime.Close()
		return nil, nil
	},
	"GET startup/done": func(e *app.Engine, _ *pyjson.Object, _ *http.Request) (any, error) {
		select {
		case <-e.Recovery.Done():
			return true, nil
		default:
			return false, nil
		}
	},
	"POST startup/wait": func(e *app.Engine, body *pyjson.Object, _ *http.Request) (any, error) {
		timeout := time.Duration(-1)
		if seconds, ok := toFloat(field(body, "timeout")); ok {
			timeout = time.Duration(seconds * float64(time.Second))
		}
		return e.Recovery.Wait(timeout), nil
	},
}

func field(body *pyjson.Object, key string) any {
	if body == nil {
		return nil
	}
	value, _ := body.Get(key)
	return value
}

func argList(body *pyjson.Object) []any { args, _ := field(body, "args").([]any); return args }

func kwargs(body *pyjson.Object) *pyjson.Object {
	if values, ok := field(body, "kwargs").(*pyjson.Object); ok {
		return values
	}
	return pyjson.NewObject()
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case int64:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
}

// decodeWire turns wire tags into Go values (only bytes are needed by the operations).
func decodeWire(value any) any {
	switch v := value.(type) {
	case []any:
		for i, item := range v {
			v[i] = decodeWire(item)
		}
	case *pyjson.Object:
		if raw, tagged := v.Get("__bytes__"); tagged && v.Len() == 1 {
			text, _ := raw.(string)
			data, _ := base64.StdEncoding.DecodeString(text)
			return wireBytes(data)
		}
		for _, key := range v.Keys() {
			item, _ := v.Get(key)
			v.Set(key, decodeWire(item))
		}
	}
	return value
}

// runParityOp answers one operation: {"value": ...} or {"raise": encoded exception} (500).
func runParityOp(w http.ResponseWriter, e *app.Engine, op parityOp, body []byte, r *http.Request) {
	parsed := pyjson.NewObject()
	if len(body) > 0 {
		decoded, err := pyjson.Decode(body)
		if object, ok := decodeWire(decoded).(*pyjson.Object); err == nil && ok {
			parsed = object
		}
	}
	value, err := op(e, parsed, r)
	if err != nil {
		writeRaw(w, http.StatusInternalServerError, `{"raise": `+exceptionJSON(err)+`}`)
		return
	}
	if data, isBytes := value.(wireBytes); isBytes {
		encoded, _ := json.Marshal(base64.StdEncoding.EncodeToString(data))
		writeRaw(w, http.StatusOK, `{"value": {"__bytes__": `+string(encoded)+`}}`)
		return
	}
	encoded, err := pyjson.Dumps(value)
	if err != nil {
		writeRaw(w, http.StatusInternalServerError, `{"raise": `+exceptionJSON(err)+`}`)
		return
	}
	writeRaw(w, http.StatusOK, `{"value": `+string(encoded)+`}`)
}

// exceptionJSON is wire.encode_exception for the Python class an error stands for.
func exceptionJSON(err error) string {
	kind, module := "RuntimeError", "builtins"
	var known *apperr.Error
	message := err.Error()
	if errors.As(err, &known) {
		message = known.Message
		switch known.Kind {
		case apperr.Studio:
			kind, module = "StudioError", "studio_app.contracts.errors"
		case apperr.Value:
			kind = "ValueError"
		case apperr.Key:
			kind = "KeyError"
		}
	}
	args, _ := json.Marshal([]string{message})
	return fmt.Sprintf(`{"type": %q, "module": %q, "args": %s}`, kind, module, args)
}

func writeRaw(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
