package netset

import (
	"regexp"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Reaches are where GOOSE goes (ADR-0073): inside the app, or also shared outside.
var Reaches = []string{"inside", "outside"}

// ReachKeys are the network keys only the GOOSE reach setting changes.
var ReachKeys = []string{"goose_reach", "goose_uplink_guid", "goose_share_guid"}

// guid is the Windows adapter GUID pattern; Python's "$" also matches before a final "\n".
var guid = regexp.MustCompile(`^\{[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}\}\n?$`)

// DefaultNetwork returns a fresh DEFAULT_NETWORK.
func DefaultNetwork() *pyjson.Object {
	return pyjson.NewObject().Set("mode", "loopback").Set("adapter_guid", "").Set("adapter_label", "").
		Set("nominal_hz", int64(50)).Set("goose_reach", "inside").Set("goose_uplink_guid", "").
		Set("goose_share_guid", "")
}

// ValidateNetwork is validate_network: the stored network section with defaults filled in.
func ValidateNetwork(raw any) (*pyjson.Object, error) {
	value, err := pyval.Merge(DefaultNetwork(), pyval.Or(raw, pyjson.NewObject()))
	if err != nil {
		return nil, err
	}
	mode, _ := value.Get("mode")
	if member, err := pyval.InStrings(mode, "loopback", "lab"); err != nil {
		return nil, err
	} else if !member {
		return nil, apperr.ValueError("Network mode must be 'loopback' (this PC only) or 'lab' (network adapter)")
	}
	adapter := guidText(value, "adapter_guid")
	if mode == "lab" && !guid.MatchString(adapter) {
		return nil, apperr.ValueError("Choose a network adapter before switching to lab network mode")
	}
	if adapter != "" && !guid.MatchString(adapter) {
		return nil, apperr.ValueError("Adapter identifier is not a Windows adapter GUID")
	}
	hz, _ := value.Get("nominal_hz")
	_, isBool := hz.(bool)
	if err := pyval.CheckHashable(hz); err != nil {
		return nil, err
	}
	if !(pyval.Equal(hz, int64(50)) || pyval.Equal(hz, int64(60))) || isBool {
		return nil, apperr.ValueError("Nominal frequency must be 50 Hz or 60 Hz")
	}
	reach, _ := value.Get("goose_reach")
	if !pyval.InTuple(reach, Reaches...) {
		return nil, apperr.ValueError("GOOSE must stay 'inside' ScadaBench or be shared 'outside' with the network")
	}
	uplink, share := guidText(value, "goose_uplink_guid"), guidText(value, "goose_share_guid")
	for _, text := range []string{uplink, share} {
		if text != "" && !guid.MatchString(text) {
			return nil, apperr.ValueError("GOOSE adapter identifier is not a Windows adapter GUID")
		}
	}
	label := pyval.Prefix(pyval.Str(pyval.Or(pyval.Lookup(value, "adapter_label", nil), "")), 100)
	return pyjson.NewObject().Set("mode", mode).Set("adapter_guid", adapter).Set("adapter_label", label).
		Set("nominal_hz", hz).Set("goose_reach", reach).Set("goose_uplink_guid", uplink).
		Set("goose_share_guid", share), nil
}

// guidText is str(value.get(key) or "").upper().
func guidText(value *pyjson.Object, key string) string {
	return pyval.Upper(pyval.Str(pyval.Or(pyval.Lookup(value, key, nil), "")))
}

// LabNotice is projects/goose_reach.lab_notice: true once for a lab project saved before
// "Where GOOSE goes" existed (ADR-0073).
func LabNotice(rawNetwork any, network *pyjson.Object) bool {
	mode, _ := network.Get("mode")
	if mode != "lab" {
		return false
	}
	stored, isDict := pyval.Or(rawNetwork, pyjson.NewObject()).(*pyjson.Object)
	return !isDict || !pyval.Has(stored, "goose_reach")
}
