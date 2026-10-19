package settings

import (
	"math/big"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/pytext"
)

// Limits of tool_memory.py.
const (
	maxHosts = 50
	maxWatch = 200
)

// text is _text(value, limit): a str of at most limit code points, else None (nil).
func text(value any, limit int) (string, bool) {
	s, ok := value.(string)
	if !ok || pytext.Len(s) > limit {
		return "", false
	}
	return s, true
}

// truthyText is `_text(value, limit)` used as a condition or with `or ""`: an empty
// string is falsy in Python.
func truthyText(value any, limit int) string {
	s, _ := text(value, limit)
	return s
}

// isInt is isinstance(value, int): bool counts as int in Python.
func isInt(value any) bool {
	switch value.(type) {
	case bool, int, int64, *big.Int:
		return true
	}
	return false
}

func dict(value any) *pyjson.Object {
	if object, ok := value.(*pyjson.Object); ok {
		return object
	}
	return pyjson.NewObject()
}

func lookup(object *pyjson.Object, key string) any {
	value, _ := object.Get(key)
	return value
}

// cleanWatch is _watch: the first 200 dict items whose ref (<= 200) and fc (<= 4) are
// non-empty text, as {"ref", "fc", "type": str(item.get("type", ""))[:32]}.
func cleanWatch(items any) []any {
	list, ok := items.([]any)
	clean := []any{}
	if !ok {
		return clean
	}
	if len(list) > maxWatch {
		list = list[:maxWatch]
	}
	for _, raw := range list {
		item, isDict := raw.(*pyjson.Object)
		if !isDict || truthyText(lookup(item, "ref"), 200) == "" || truthyText(lookup(item, "fc"), 4) == "" {
			continue
		}
		kind, found := item.Get("type")
		if !found {
			kind = ""
		}
		clean = append(clean, pyjson.NewObject().Set("ref", lookup(item, "ref")).
			Set("fc", lookup(item, "fc")).Set("type", pytext.Slice(pytext.Str(kind), 32)))
	}
	return clean
}

// clean is _clean(raw): the remembered tool settings with their fixed key order.
func clean(raw any) *pyjson.Object {
	root := dict(raw)
	explorer := dict(lookup(root, "explorer"))
	find := dict(lookup(root, "find"))
	watches := dict(lookup(explorer, "watch_by_host"))
	port := lookup(explorer, "port")
	if !isInt(port) {
		port = int64(102)
	}
	interval := lookup(explorer, "interval_ms")
	if !intervalOK(interval) {
		interval = int64(1000)
	}
	byHost := pyjson.NewObject()
	hosts := watches.Keys()
	if len(hosts) > maxHosts {
		hosts = hosts[len(hosts)-maxHosts:]
	}
	for _, host := range hosts {
		if truthyText(host, 64) != "" {
			byHost.Set(host, cleanWatch(lookup(watches, host)))
		}
	}
	findPort := lookup(find, "port")
	if !isInt(findPort) {
		findPort = int64(102)
	}
	return pyjson.NewObject().
		Set("explorer", pyjson.NewObject().Set("host", truthyText(lookup(explorer, "host"), 64)).
			Set("port", port).Set("interval_ms", interval).Set("watch_by_host", byHost)).
		Set("find", pyjson.NewObject().Set("range", truthyText(lookup(find, "range"), 64)).
			Set("port", findPort).Set("choice", truthyText(lookup(find, "choice"), 200)))
}

// intervalOK is `isinstance(interval, int) and 500 <= interval <= 60000`.
func intervalOK(value any) bool {
	switch v := value.(type) {
	case int64:
		return v >= 500 && v <= 60000
	case int:
		return v >= 500 && v <= 60000
	}
	return false // bool is 0/1, a big int is out of range
}
