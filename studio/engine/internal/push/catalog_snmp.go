package push

// SNMP devices tool change names (snmp/service, ADR-0092); they are also topic triggers.
const (
	ChangeSNMPStatus   = "snmp.status"
	ChangeSNMPRequests = "snmp.requests"
	ChangeSNMPTraps    = "snmp.traps"
	ChangeSNMPValues   = "snmp.values"
)

// snmpSpecs are the Go-only SNMP tool topics: its status, the reader's table and the
// request and trap logs. Device point tables are polled (their path names the device).
func snmpSpecs() []Spec {
	whole := func(name, pattern string, triggers ...string) Spec {
		return Spec{Pattern: "/api/v1" + pattern, Topic: Topic{Name: name, Triggers: triggers}}
	}
	keyed := func(spec Spec, list, key string) Spec {
		spec.Topic.Kind, spec.Topic.List, spec.Topic.Key = Keyed, list, key
		return spec
	}
	return []Spec{
		whole("snmp.status", "/tools/snmp", ChangeSNMPStatus),
		keyed(whole("snmp.values", "/tools/snmp/read/values", ChangeSNMPValues, ChangeSNMPStatus), "values", "oid"),
		keyed(whole("snmp.requests", "/tools/snmp/requests", ChangeSNMPRequests), "entries", "seq"),
		keyed(whole("snmp.traps", "/tools/snmp/traps", ChangeSNMPTraps), "entries", "seq"),
	}
}
