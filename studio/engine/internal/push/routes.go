package push

import (
	"io"
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/pyjson"
)

// Version is the push protocol version announced by /capabilities.
const Version = 1

// Routes implements api.Area. These routes exist only in the Go engine; the Python engine
// answers /capabilities with its JSON 404 and the frontend keeps polling.
func (a *Area) Routes() []api.Route {
	return []api.Route{
		{Method: http.MethodGet, Pattern: "/api/v1/capabilities", Handler: httpapi.Handle(a.capabilities)},
		{Method: http.MethodGet, Pattern: "/api/v1/push", Handler: httpapi.Handle(a.stream)},
		{Method: http.MethodPut, Pattern: "/api/v1/push/topics", Handler: httpapi.Handle(a.putTopics)},
	}
}

// capabilities is {"epoch": boot id, "paths": {GET path under /api/v1: {"t": topic,
// "params": [query names the topic takes]}}, "push": 1, "topics": [sorted names]}.
func (a *Area) capabilities(w http.ResponseWriter, r *http.Request) error {
	topics := make([]any, len(a.names))
	paths := pyjson.NewObject()
	for index, name := range a.names {
		topics[index] = name
		topic := a.topics[name]
		if topic.Path == "" {
			continue
		}
		params := make([]any, len(topic.Params))
		for at, param := range topic.Params {
			params[at] = param
		}
		paths.Set(topic.Path, pyjson.NewObject().Set("t", name).Set("params", params))
	}
	httpapi.WriteJSON(w, http.StatusOK, pyjson.NewObject().Set("epoch", a.epoch).Set("paths", paths).
		Set("push", int64(Version)).Set("topics", topics))
	return nil
}

var errTopicsBody = apperr.ValueError(`Send {"client": tab id, "topics": [{"t": name, "params": {}}], "resync": [names]}`)

// putTopics is PUT /api/v1/push/topics {client, topics: [{t, params}], resync?: [t]}: it
// replaces the tab's subscriptions and answers {"epoch", "topics": accepted, "unknown"}.
// Unknown names are reported, not refused, so an older server works with a newer page.
func (a *Area) putTopics(w http.ResponseWriter, r *http.Request) error {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		return errTopicsBody
	}
	decoded, err := pyjson.Decode(raw)
	body, isObject := decoded.(*pyjson.Object)
	if err != nil || !isObject {
		return errTopicsBody
	}
	clientValue, _ := body.Get("client")
	clientText, _ := clientValue.(string)
	id, err := clientID(clientText)
	if err != nil {
		return err
	}
	listed, _ := body.Get("topics")
	items, isList := listed.([]any)
	if listed != nil && !isList {
		return errTopicsBody
	}
	var wanted []want
	accepted, unknown := []any{}, []any{}
	for _, item := range items {
		entry, _ := item.(*pyjson.Object)
		name, _ := get(entry, "t").(string)
		topic := a.topics[name]
		if topic == nil {
			unknown = append(unknown, name)
			continue
		}
		params, _ := get(entry, "params").(*pyjson.Object)
		wanted = append(wanted, want{topic: topic, params: params})
		accepted = append(accepted, name)
	}
	var resync []string
	if listedResync, _ := body.Get("resync"); listedResync != nil {
		names, isList := listedResync.([]any)
		if !isList {
			return errTopicsBody
		}
		resync = []string{}
		for _, name := range names {
			if text, isText := name.(string); isText {
				resync = append(resync, text)
			}
		}
	}
	if err := a.subscribe(id, wanted, resync); err != nil {
		return err
	}
	httpapi.WriteJSON(w, http.StatusOK, pyjson.NewObject().Set("epoch", a.epoch).Set("topics", accepted).
		Set("unknown", unknown))
	return nil
}

func get(object *pyjson.Object, key string) any {
	if object == nil {
		return nil
	}
	value, _ := object.Get(key)
	return value
}
