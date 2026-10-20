package netcfg

import (
	"bytes"
	"errors"
	"regexp"
	"unicode/utf8"

	"studio-engine/internal/pyjson"
)

// Patterns of helper/validation.py. Windows names the rules of its "allow access?" question
// like "TCP Query User{GUID}c:\path\program.exe"; no quotes, so a name cannot leave its
// PowerShell string. \w is Unicode in Python, hence \p{L}\p{N}_.
var (
	guidPattern        = regexp.MustCompile(`^\{[0-9A-F]{8}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{4}-[0-9A-F]{12}\}$`)
	ruleNamePattern    = regexp.MustCompile(`^` + regexp.QuoteMeta(RulePrefix) + `[0-9a-f]{12}$`)
	gatewayRulePattern = regexp.MustCompile(`^` + regexp.QuoteMeta(GatewayPrefix) + `[0-9a-f]{12}$`)
	windowsRulePattern = regexp.MustCompile(`^[\p{L}\p{N}_ {}\-.:\\()]{1,400}$`)
)

// HelperRefused is validation.HelperRefused: the request was refused before any Windows
// change was made. Its text goes to the result file as "reason".
type HelperRefused struct{ Reason string }

func (e *HelperRefused) Error() string { return e.Reason }

func refuse(reason string) error { return &HelperRefused{Reason: reason} }

// errHelperCrash stands for a Python exception the helper did not catch (it exits 1 and
// writes no result file).
var errHelperCrash = errors.New("network helper request could not be read")

func labAddress(value any) error {
	address, ok := ipv4Address(value)
	if !ok {
		return refuse("Not an IPv4 address: " + pyRepr(value))
	}
	if isLoopback(address) || isMulticast(address) || isLinkLocal(address) || isUnspecified(address) {
		return refuse("Address " + formatIPv4(address) + " cannot be used for the lab network")
	}
	return nil
}

func get(item *pyjson.Object, key string) any {
	value, _ := item.Get(key)
	return value
}

func textOr(item *pyjson.Object, key string) string {
	value, found := item.Get(key)
	if !found {
		return ""
	}
	return pyStr(value)
}

func validateAlias(item *pyjson.Object) error {
	if !pyMatch(guidPattern, textOr(item, "adapter_guid")) {
		return refuse("Alias item has no valid adapter GUID")
	}
	if index, ok := pyInt(get(item, "adapter_index")); !ok || index.Sign() <= 0 {
		return refuse("Alias item has no valid adapter index")
	}
	if !pyIntIn(get(item, "prefix"), 8, 30) {
		return refuse("Alias prefix must be between 8 and 30")
	}
	return labAddress(get(item, "ip"))
}

func validateFirewall(item *pyjson.Object, program string) error {
	if !pyMatch(ruleNamePattern, textOr(item, "name")) || get(item, "group") != Group {
		return refuse("Firewall item is not a ScadaBench rule")
	}
	ports, ok := get(item, "ports").([]any)
	if !ok || len(ports) == 0 {
		return refuse("Firewall ports are invalid")
	}
	for _, port := range ports {
		if !pyIntIn(port, 1, 65535) {
			return refuse("Firewall ports are invalid")
		}
	}
	addresses, ok := get(item, "addresses").([]any)
	if !ok || len(addresses) == 0 {
		return refuse("Firewall rule needs at least one address")
	}
	for _, address := range addresses {
		if err := labAddress(address); err != nil {
			return err
		}
	}
	if !samePath(textOr(item, "program"), program) {
		return refuse("Firewall rule must name the ScadaBench IED worker")
	}
	return nil
}

func validateGatewayProgram(item *pyjson.Object, program string) error {
	if !samePath(textOr(item, "program"), gatewayWorker(program)) {
		return refuse("Gateway firewall changes must name the ScadaBench gateway worker")
	}
	return nil
}

func validateItem(item *pyjson.Object, program string) error {
	switch get(item, "kind") {
	case "alias":
		return validateAlias(item)
	case "firewall":
		return validateFirewall(item, program)
	case "snmp_firewall":
		return validateSNMPFirewall(item, program)
	case "gateway_firewall":
		if !pyMatch(gatewayRulePattern, textOr(item, "name")) || get(item, "group") != GatewayGroup {
			return refuse("Gateway firewall item is not a ScadaBench rule")
		}
		return validateGatewayProgram(item, program)
	case "unblock":
		if !pyMatch(windowsRulePattern, textOr(item, "name")) {
			return refuse("Not a Windows Firewall rule name")
		}
		return validateGatewayProgram(item, program) // the command re-checks the live rule first
	}
	return refuse("Unknown change kind " + pyRepr(get(item, "kind")))
}

// decodeRequest is json.loads(bytes): UTF-8 with or without BOM.
func decodeRequest(payload []byte) (any, error) {
	payload = bytes.TrimPrefix(payload, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(payload) {
		return nil, errors.New("invalid UTF-8")
	}
	return pyjson.Decode(payload)
}

// ValidateRequest is validation.validate_request: the parsed request if it is untampered and
// fully allowed, else *HelperRefused (or errHelperCrash where Python raised another error).
func ValidateRequest(payload []byte, expectedHash, program string) (*pyjson.Object, error) {
	if sha256Hex(payload) != expectedHash {
		return nil, refuse("Request file changed after ScadaBench prepared it")
	}
	value, err := decodeRequest(payload)
	if err != nil {
		return nil, refuse("Request file is not valid JSON")
	}
	request, ok := value.(*pyjson.Object)
	if !ok {
		return nil, errHelperCrash
	}
	action := get(request, "action")
	switch action.(type) {
	case []any, *pyjson.Object:
		return nil, errHelperCrash // unhashable: "in ACTIONS" raised TypeError
	}
	items, isList := get(request, "items").([]any)
	if (action != "apply" && action != "remove") || !isList {
		return nil, refuse("Unknown helper action")
	}
	if len(items) == 0 || len(items) > 64 {
		return nil, refuse("Request must contain between 1 and 64 changes")
	}
	for _, raw := range items {
		item, ok := raw.(*pyjson.Object)
		if !ok || !truthy(get(item, "owned")) {
			return nil, refuse("Only changes owned by ScadaBench can be applied or removed")
		}
		if err := validateItem(item, program); err != nil {
			return nil, err
		}
	}
	return request, nil
}
