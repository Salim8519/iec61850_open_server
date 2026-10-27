package gateway

import (
	"net"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// The network choices of the SCADA master (ADR-0123, "local") and of the station gateway
// (ADR-0124, "listen_address" and "allowed_scada") are Go engine only. Like the master point
// list they are written only when stored, so a project without them normalises to the bytes the
// frozen Python engine produces. "" means the default; any other value must be IPv4 text, and a
// damaged value stops the load with the reason instead of being dropped.
func keepAddresses(stored, result *pyjson.Object, keys ...string) error {
	for _, key := range keys {
		raw, found := stored.Get(key)
		if !found {
			continue
		}
		text, isText := raw.(string)
		if !isText || (text != "" && net.ParseIP(text).To4() == nil) {
			return apperr.ValueError("The saved network address " + key + " must be an IPv4 address or empty")
		}
		result.Set(key, text)
	}
	return nil
}

// keepAddressList keeps a stored list of IPv4 addresses (the gateway's own addresses on the lab
// card, ADR-0125) the same way.
func keepAddressList(stored, result *pyjson.Object, key string) error {
	raw, found := stored.Get(key)
	if !found {
		return nil
	}
	list, isList := raw.([]any)
	if !isList {
		return apperr.ValueError("The saved network addresses " + key + " must be a list of IPv4 addresses")
	}
	kept := []any{}
	for _, item := range list {
		text, isText := item.(string)
		if !isText || net.ParseIP(text).To4() == nil {
			return apperr.ValueError("The saved network addresses " + key + " must be a list of IPv4 addresses")
		}
		kept = append(kept, text)
	}
	result.Set(key, kept)
	return nil
}
