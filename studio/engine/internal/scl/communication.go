package scl

import (
	"math"
	"math/big"
	"regexp"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

var macPattern = regexp.MustCompile(`^([0-9A-Fa-f]{2}-){5}[0-9A-Fa-f]{2}$`)

// communication is read_communication's result for one IED.
type communication struct {
	address  any // nil or the address object
	gse, smv []*pyjson.Object
}

// readParameters is _parameters: {P type: stripped text}, later duplicates win.
func readParameters(address *xmldoc.Element) map[string]*string {
	result := map[string]*string{}
	if address == nil {
		return result
	}
	for _, item := range address.FindAll("P") {
		key, present := item.Get("type")
		if !present {
			key = "\x00None"
		}
		text := pyrules.Strip(item.Text)
		result[key] = &text
	}
	return result
}

func lookup(values map[string]*string, key, fallback string) (string, bool) {
	if value := values[key]; value != nil {
		return *value, true
	}
	return fallback, false
}

// ipv4 is _ip: str(ipaddress.IPv4Address(value)) or "".
func ipv4(value string) string {
	parts := strings.Split(value, ".")
	if value == "" || len(parts) != 4 {
		return ""
	}
	for _, part := range parts {
		if part == "" || len(part) > 3 || len(part) > 1 && part[0] == '0' {
			return ""
		}
		number := 0
		for _, c := range []byte(part) {
			if c < '0' || c > '9' {
				return ""
			}
			number = number*10 + int(c-'0')
		}
		if number > 255 {
			return ""
		}
	}
	return value
}

// milliseconds is _milliseconds: a non-negative SCL time rounded half-even, else nil.
func milliseconds(e *xmldoc.Element) any {
	if e == nil || pyrules.Strip(e.Text) == "" {
		return nil
	}
	value, err := pyrules.Float(pyrules.Strip(e.Text))
	if err != nil || !(value >= 0) {
		return nil
	}
	if math.IsInf(value, 1) {
		return nil // Python raises OverflowError here; never seen in valid SCL
	}
	rounded, _ := new(big.Float).SetFloat64(pyrules.Round(value)).Int(nil)
	return intValue(rounded)
}

func hexInt(text string) (*big.Int, bool) {
	value, err := pyrules.IntBig(text, 16)
	return value, err == nil
}

func gseAddress(e *xmldoc.Element, warnings *[]string, iedName, kind string) *pyjson.Object {
	values := readParameters(e.Find("Address"))
	ld, hasLD := e.Get("ldInst")
	cb, hasCB := e.Get("cbName")
	where := kind + " address " + iedName + "/" + noneOr(ld, hasLD) + "/" + noneOr(cb, hasCB)
	mac, _ := lookup(values, "MAC-Address", "")
	appText, _ := lookup(values, "APPID", "")
	vlanText, _ := lookup(values, "VLAN-ID", "0")
	priorityText, _ := lookup(values, "VLAN-PRIORITY", "4")
	if vlanText == "" {
		vlanText = "0"
	}
	if priorityText == "" {
		priorityText = "4"
	}
	appID, ok1 := hexInt(appText)
	vlanID, ok2 := hexInt(vlanText)
	priority, err := pyrules.IntBig(priorityText, 10)
	if !ok1 || !ok2 || err != nil {
		*warnings = append(*warnings, where+" has an unreadable APPID, VLAN-ID or VLAN-PRIORITY")
		return nil
	}
	if !macPattern.MatchString(mac) {
		*warnings = append(*warnings, where+" has no valid MAC-Address")
		return nil
	}
	inRange := func(v *big.Int, max int64) bool { return v.Sign() >= 0 && v.Cmp(big.NewInt(max)) <= 0 }
	if !inRange(appID, 0xffff) || !inRange(vlanID, 0xfff) || !inRange(priority, 7) {
		*warnings = append(*warnings, where+" has APPID, VLAN-ID or VLAN-PRIORITY out of range")
		return nil
	}
	return pyjson.NewObject().Set("ld_inst", e.GetOr("ldInst", "")).Set("cb_name", e.GetOr("cbName", "")).
		Set("mac", strings.ToUpper(mac)).Set("app_id", appID.Int64()).Set("vlan_id", vlanID.Int64()).
		Set("vlan_priority", priority.Int64()).Set("min_time", milliseconds(e.Find("MinTime"))).
		Set("max_time", milliseconds(e.Find("MaxTime")))
}

// readCommunication is read_communication(root, ied_name).
func readCommunication(root *xmldoc.Element, iedName string) (communication, []string) {
	warnings := []string{}
	var result communication
	for _, subnetwork := range root.FindAll("Communication/SubNetwork") {
		for _, access := range subnetwork.FindAll("ConnectedAP") {
			if name, ok := access.Get("iedName"); !ok || name != iedName {
				continue
			}
			values := readParameters(access.Find("Address"))
			raw, hasIP := lookup(values, "IP", "")
			ip := ipv4(raw)
			switch {
			case ip != "" && result.address == nil:
				subnet, _ := lookup(values, "IP-SUBNET", "")
				gateway, _ := lookup(values, "IP-GATEWAY", "")
				result.address = pyjson.NewObject().Set("ip", ip).Set("subnet", ipv4(subnet)).
					Set("gateway", ipv4(gateway)).Set("access_point", access.GetOr("apName", "")).
					Set("subnetwork", subnetwork.GetOr("name", ""))
			case hasIP && raw != "" && ip == "":
				warnings = append(warnings, "SCD address for "+iedName+" is not a valid IPv4 address")
			}
			for _, element := range access.FindAll("GSE") {
				if parsed := gseAddress(element, &warnings, iedName, "GOOSE"); parsed != nil {
					result.gse = append(result.gse, parsed)
				}
			}
			for _, element := range access.FindAll("SMV") {
				if parsed := gseAddress(element, &warnings, iedName, "Sampled Values"); parsed != nil {
					result.smv = append(result.smv, parsed)
				}
			}
		}
	}
	return result, warnings
}
