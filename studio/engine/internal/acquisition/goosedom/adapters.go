package goosedom

import (
	"sort"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// kindOrder is KIND_ORDER: how adapter kinds are listed (unknown kinds last, 9).
var kindOrder = map[string]int{"ethernet": 0, "wifi": 1, "virtual_lab": 2, "virtual": 3, "vpn": 4, "bluetooth": 5}

// adapterKind is _kind: what an adapter is, from its description and name.
func adapterKind(item *pyjson.Object) string {
	text := pyLower(strOr(item, "description", "") + " " + strOr(item, "name", ""))
	has := func(words ...string) bool {
		for _, word := range words {
			if strings.Contains(text, word) {
				return true
			}
		}
		return false
	}
	switch {
	case strings.Contains(text, "loopback") && strings.Contains(text, "km-test"):
		return "virtual_lab"
	case has("vmware", "virtualbox", "hyper-v", "vethernet", "km-test"):
		return "virtual"
	case has("wi-fi", "wireless", "wlan", "802.11"):
		return "wifi"
	case has("vpn", "zerotier", "tap-", "tunnel", "wireguard", "openvpn"):
		return "vpn"
	case strings.Contains(text, "bluetooth"):
		return "bluetooth"
	}
	return "ethernet"
}

// NormalizeAdapters turns adapter records (IP Helper list or decoded PowerShell JSON)
// into stable new objects sorted by kind, connection, default route and name, like
// networking/adapters.normalize. A single PowerShell object must be passed as a
// one-item list; listing the adapters (PowerShell, IP Helper) is not part of this port.
func NormalizeAdapters(items []any) []*pyjson.Object {
	adapters := []*pyjson.Object{}
	for _, raw := range items {
		item := pyv.Obj(raw)
		addresses := pyv.Get(item, "addresses")
		var list []any
		if object, isObject := addresses.(*pyjson.Object); isObject {
			list = []any{object}
		} else if pyv.Truthy(addresses) {
			list = pyv.List(addresses)
		}
		guid := pyUpper(strOr(item, "guid", ""))
		if !strings.HasPrefix(guid, "{") {
			guid = "{" + guid + "}"
		}
		views := []any{}
		for _, address := range pyv.Objects(list) {
			views = append(views, pyv.O("ip", pyv.Get(address, "ip"), "prefix", pyv.Get(address, "prefix"),
				"skip_as_source", pyv.Truthy(pyv.Get(address, "skip_as_source")),
				"origin", pyv.GetOr(address, "origin", "")))
		}
		adapters = append(adapters, pyv.O("guid", guid, "name", pyv.GetOr(item, "name", ""),
			"description", pyv.GetOr(item, "description", ""), "status", pyv.GetOr(item, "status", ""),
			"index", pyv.Get(item, "index"), "mac", pyv.GetOr(item, "mac", ""), "speed", pyv.GetOr(item, "speed", ""),
			"kind", adapterKind(item), "default_route", pyv.Truthy(pyv.Get(item, "default_route")),
			"addresses", views))
	}
	sort.SliceStable(adapters, func(i, j int) bool { return adapterLess(adapters[i], adapters[j]) })
	return adapters
}

// adapterLess orders by (KIND_ORDER, status != "Up", default_route, name.lower()):
// False sorts before True, so adapters without the default route come first.
func adapterLess(left, right *pyjson.Object) bool {
	if a, b := kindRank(left), kindRank(right); a != b {
		return a < b
	}
	if a, b := !pyv.Equal(pyv.Get(left, "status"), "Up"), !pyv.Equal(pyv.Get(right, "status"), "Up"); a != b {
		return !a
	}
	if a, b := pyv.Truthy(pyv.Get(left, "default_route")), pyv.Truthy(pyv.Get(right, "default_route")); a != b {
		return !a
	}
	return pyLower(pyv.PyStr(pyv.Get(left, "name"))) < pyLower(pyv.PyStr(pyv.Get(right, "name")))
}

func kindRank(adapter *pyjson.Object) int {
	if rank, found := kindOrder[pyv.GetStr(adapter, "kind")]; found {
		return rank
	}
	return 9
}

// pyUpper is str.upper() for adapter GUIDs. ASCII is exact; other text uses Go's
// simple uppercase mapping (Python's multi-character mappings such as "ß" -> "SS" are
// not reproduced; Windows GUIDs are hexadecimal).
func pyUpper(text string) string {
	return strings.ToUpper(text)
}
