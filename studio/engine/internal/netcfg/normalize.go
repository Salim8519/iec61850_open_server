package netcfg

import (
	"sort"
	"strings"

	"studio-engine/internal/pyjson"
)

var kindOrder = map[string]int{"ethernet": 0, "wifi": 1, "virtual_lab": 2, "virtual": 3, "vpn": 4, "bluetooth": 5}

func containsAny(text string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

// adapterKind is adapters._kind: keywords in description and name.
func adapterKind(description, name string) string {
	text := strings.ToLower(description + " " + name)
	switch {
	case strings.Contains(text, "loopback") && strings.Contains(text, "km-test"):
		return "virtual_lab"
	case containsAny(text, "vmware", "virtualbox", "hyper-v", "vethernet", "km-test"):
		return "virtual"
	case containsAny(text, "wi-fi", "wireless", "wlan", "802.11"):
		return "wifi"
	case containsAny(text, "vpn", "zerotier", "tap-", "tunnel", "wireguard", "openvpn"):
		return "vpn"
	case strings.Contains(text, "bluetooth"):
		return "bluetooth"
	}
	return "ethernet"
}

// field is item.get(key, fallback) turned into text (str() for non-strings).
func field(item *pyjson.Object, key, fallback string) string {
	value, found := item.Get(key)
	if !found {
		return fallback
	}
	return pyStr(value)
}

func objects(value any) []*pyjson.Object {
	switch v := value.(type) {
	case *pyjson.Object:
		return []*pyjson.Object{v}
	case []any:
		found := make([]*pyjson.Object, 0, len(v))
		for _, item := range v {
			if object, ok := item.(*pyjson.Object); ok {
				found = append(found, object)
			}
		}
		return found
	}
	return nil
}

func intField(item *pyjson.Object, key string) (int64, bool) {
	value, _ := item.Get(key)
	number, ok := pyInt(value)
	if !ok || !number.IsInt64() {
		return 0, false
	}
	return number.Int64(), true
}

func normalizeOne(item *pyjson.Object) Adapter {
	guid := strings.ToUpper(field(item, "guid", ""))
	if !strings.HasPrefix(guid, "{") {
		guid = "{" + guid + "}"
	}
	adapter := Adapter{GUID: guid, Name: field(item, "name", ""), Description: field(item, "description", ""),
		Status: field(item, "status", ""), MAC: field(item, "mac", ""), Speed: field(item, "speed", "")}
	if index, ok := intField(item, "index"); ok {
		adapter.Index = &index
	}
	adapter.Kind = adapterKind(adapter.Description, adapter.Name)
	route, _ := item.Get("default_route")
	adapter.DefaultRoute = truthy(route)
	raw, _ := item.Get("addresses")
	for _, address := range objects(raw) {
		prefix, _ := intField(address, "prefix")
		skip, _ := address.Get("skip_as_source")
		adapter.Addresses = append(adapter.Addresses, Address{IP: field(address, "ip", ""), Prefix: int(prefix),
			SkipAsSource: truthy(skip), Origin: field(address, "origin", "")})
	}
	return adapter
}

// Normalize is adapters.normalize for decoded records (IP Helper list or PowerShell JSON):
// GUIDs upper-case in braces, a kind per adapter, sorted Ethernet first, then up before down,
// then without default route first, then by name.
func Normalize(raw any) []Adapter {
	adapters := []Adapter{}
	for _, item := range objects(raw) {
		adapters = append(adapters, normalizeOne(item))
	}
	order := func(kind string) int {
		if rank, found := kindOrder[kind]; found {
			return rank
		}
		return 9
	}
	sort.SliceStable(adapters, func(i, j int) bool {
		a, b := adapters[i], adapters[j]
		if order(a.Kind) != order(b.Kind) {
			return order(a.Kind) < order(b.Kind)
		}
		if (a.Status != "Up") != (b.Status != "Up") {
			return a.Status == "Up"
		}
		if a.DefaultRoute != b.DefaultRoute {
			return !a.DefaultRoute
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	return adapters
}

// NormalizeJSON is adapters.normalize for PowerShell's JSON text ("" means no adapters).
func NormalizeJSON(text string) ([]Adapter, error) {
	if text == "" {
		text = "[]"
	}
	value, err := pyjson.Decode([]byte(text))
	if err != nil {
		return nil, err
	}
	return Normalize(value), nil
}
