package netcfg

import (
	"sort"
	"strings"

	"studio-engine/internal/pyjson"
)

// Names that identify ScadaBench's Windows changes (change_plan.py). GROUP keeps the old
// product name: rules already created under it must still be found and removed.
const (
	Group         = "Substation Studio"
	RulePrefix    = "SubstationStudio-MMS-"
	GatewayGroup  = "ScadaBench"
	GatewayPrefix = "ScadaBench-Gateway-"
)

// Plan is an exact, fingerprinted list of Windows changes. Items keep Python's key order.
type Plan struct {
	Items       []*pyjson.Object
	Fingerprint string
}

func contains(list []string, wanted string) bool {
	for _, item := range list {
		if item == wanted {
			return true
		}
	}
	return false
}

func keepOrAdd(present bool) string {
	if present {
		return "keep"
	}
	return "add"
}

func stringsAny(values []string) []any {
	items := make([]any, len(values))
	for index, value := range values {
		items[index] = value
	}
	return items
}

func intsAny(values []int) []any {
	items := make([]any, len(values))
	for index, value := range values {
		items[index] = value
	}
	return items
}

// MMSRuleName is RULE_PREFIX + sha256(json.dumps([sorted(addresses), ports, str(program)],
// sort_keys=True))[:12]; ports must already be sorted and unique.
func MMSRuleName(addresses []string, ports []int, program string) (string, error) {
	sorted := append([]string(nil), addresses...)
	sort.Strings(sorted)
	key, err := dumpsSorted([]any{stringsAny(sorted), intsAny(ports), program})
	if err != nil {
		return "", err
	}
	return RulePrefix + sha256Hex(key)[:12], nil
}

// GatewayRuleName is GATEWAY_PREFIX + sha256(str(program).lower())[:12].
func GatewayRuleName(program string) string {
	return GatewayPrefix + sha256Hex([]byte(strings.ToLower(program)))[:12]
}

func uniqueSortedPorts(ports []int) []int {
	seen := map[int]bool{}
	unique := []int{}
	for _, port := range ports {
		if !seen[port] {
			seen[port] = true
			unique = append(unique, port)
		}
	}
	sort.Ints(unique)
	return unique
}

// BuildPlan is change_plan.build_plan: one alias per usable row and one narrow MMS firewall
// rule; things already present are listed but not owned.
func BuildPlan(adapter Adapter, rows []Row, ports []int, program string, existingRules []string) (Plan, error) {
	own := adapter.ips()
	items, addresses := []*pyjson.Object{}, []string{}
	for _, row := range rows {
		if row.Proposed == "" || (row.Status != "ok" && row.Status != "already_on_adapter") {
			continue
		}
		addresses = append(addresses, row.Proposed)
		present := own[row.Proposed]
		items = append(items, pyjson.NewObject().Set("kind", "alias").Set("adapter_guid", adapter.GUID).
			Set("adapter_index", adapter.indexValue()).Set("adapter_name", adapter.Name).Set("ip", row.Proposed).
			Set("prefix", row.prefixValue()).Set("for", row.Name).Set("owned", !present).
			Set("action", keepOrAdd(present)))
	}
	if len(addresses) > 0 {
		sortedPorts := uniqueSortedPorts(ports)
		name, err := MMSRuleName(addresses, sortedPorts, program)
		if err != nil {
			return Plan{}, err
		}
		sorted := append([]string(nil), addresses...)
		sort.Strings(sorted)
		present := contains(existingRules, name)
		items = append(items, pyjson.NewObject().Set("kind", "firewall").Set("name", name).Set("group", Group).
			Set("addresses", stringsAny(sorted)).Set("ports", intsAny(sortedPorts)).Set("program", program).
			Set("owned", !present).Set("action", keepOrAdd(present)))
	}
	fingerprint, err := sortedFingerprint(pyjson.NewObject().Set("adapter", adapter.GUID).Set("items", itemsAny(items)))
	return Plan{Items: items, Fingerprint: fingerprint}, err
}

func itemsAny(items []*pyjson.Object) []any {
	values := make([]any, len(items))
	for index, item := range items {
		values[index] = item
	}
	return values
}

// GatewayPlan is change_plan.gateway_plan (ADR-0062): one allow rule for the gateway worker on
// TCP, and the Windows Block rules for that worker switched off (never deleted).
func GatewayPlan(program string, existingRules []string, blocking []BlockingRule) (Plan, error) {
	name := GatewayRuleName(program)
	present := contains(existingRules, name)
	items := []*pyjson.Object{pyjson.NewObject().Set("kind", "gateway_firewall").Set("name", name).
		Set("group", GatewayGroup).Set("program", program).Set("owned", !present).Set("action", keepOrAdd(present))}
	for _, rule := range blocking {
		items = append(items, pyjson.NewObject().Set("kind", "unblock").Set("name", rule.Name).
			Set("display", rule.Display).Set("program", program).Set("owned", true).Set("action", "add"))
	}
	fingerprint, err := sortedFingerprint(pyjson.NewObject().Set("gateway", itemsAny(items)))
	return Plan{Items: items, Fingerprint: fingerprint}, err
}
