// Package memory is tools/tool_memory.py: what the user last did in each tool, kept in
// <data>/tool-settings.json byte-identical to the Python file (indent=2, CRLF, no final newline).
package memory

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Limits of the remembered lists (tool_memory.py).
const (
	MaxHosts = 50
	MaxWatch = 200
)

func text(value any, limit int) (string, bool) {
	s, ok := value.(string)
	return s, ok && pytext.Len(s) <= limit
}

// textOr is `_text(value, limit) or ""`.
func textOr(value any, limit int) string {
	s, ok := text(value, limit)
	if !ok {
		return ""
	}
	return s
}

func object(value any) *pyjson.Object {
	if found, ok := value.(*pyjson.Object); ok {
		return found
	}
	return pyjson.NewObject()
}

func get(o *pyjson.Object, key string) any {
	value, _ := o.Get(key)
	return value
}

func watch(items any) []any {
	list, ok := items.([]any)
	clean := []any{}
	if !ok {
		return clean
	}
	if len(list) > MaxWatch {
		list = list[:MaxWatch]
	}
	for _, raw := range list {
		item, isObject := raw.(*pyjson.Object)
		if !isObject {
			continue
		}
		ref, refOK := text(get(item, "ref"), 200)
		fc, fcOK := text(get(item, "fc"), 4)
		if !refOK || ref == "" || !fcOK || fc == "" {
			continue
		}
		kind := ""
		if value, found := item.Get("type"); found {
			kind = pytext.Str(value)
		}
		clean = append(clean, pyjson.NewObject().Set("ref", ref).Set("fc", fc).Set("type", pytext.Head(kind, 32)))
	}
	return clean
}

// Clean is _clean: any decoded document -> the checked memory with defaults.
func Clean(raw any) *pyjson.Object {
	document := object(raw)
	explorer := object(get(document, "explorer"))
	find := object(get(document, "find"))
	watches := object(get(explorer, "watch_by_host"))
	port := get(explorer, "port")
	if !pytext.IsIntOrBool(port) {
		port = int64(102)
	}
	interval := get(explorer, "interval_ms")
	if !pytext.IsIntOrBool(interval) || !between(interval, 500, 60000) {
		interval = int64(1000)
	}
	hosts := pyjson.NewObject()
	keys := watches.Keys()
	if len(keys) > MaxHosts {
		keys = keys[len(keys)-MaxHosts:]
	}
	for _, key := range keys {
		if name, ok := text(key, 64); ok && name != "" {
			hosts.Set(key, watch(get(watches, key)))
		}
	}
	findPort := get(find, "port")
	if !pytext.IsIntOrBool(findPort) {
		findPort = int64(102)
	}
	return pyjson.NewObject().
		Set("explorer", pyjson.NewObject().Set("host", textOr(get(explorer, "host"), 64)).Set("port", port).
			Set("interval_ms", interval).Set("watch_by_host", hosts)).
		Set("find", pyjson.NewObject().Set("range", textOr(get(find, "range"), 64)).Set("port", findPort).
			Set("choice", textOr(get(find, "choice"), 200)))
}

// between compares an int or bool with Python's chained comparison.
func between(value any, low, high int64) bool {
	number, ok := pytext.Int64(value)
	if !ok {
		return false // an int beyond int64 is outside 500..60000
	}
	return number >= low && number <= high
}
