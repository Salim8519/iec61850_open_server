package scl

import (
	"strings"

	"studio-engine/internal/scl/xmldoc"
)

// stationAccess is the access point whose server a SCADA reaches over MMS (ADR-0117). An IED
// with one server keeps exactly what it had (name "" filters nothing). NR/NARI relays put the
// station bus on S1 (IP address, reports) and the process bus on G1 (GOOSE, LD PIGO) and M1
// (Sampled Values, LD PISV): ScadaBench simulates S1 and names the others in the preview.
type stationAccess struct {
	server  *xmldoc.Element // nil: no server, or several that cannot be told apart
	name    string          // the chosen AccessPoint; "" when the IED has one server
	ip      string
	others  []*xmldoc.Element // the other access points with a server
	refusal string            // why several servers could not be told apart
}

func stationOf(root, ied *xmldoc.Element) stationAccess {
	var points []*xmldoc.Element
	for _, point := range ied.FindAll("AccessPoint") {
		if point.Find("Server") != nil {
			points = append(points, point)
		}
	}
	switch {
	case len(points) == 0:
		return stationAccess{}
	case len(points) == 1:
		return stationAccess{server: points[0].Find("Server")}
	}
	name := ied.GetOr("name", "")
	names := make([]string, len(points))
	for index, point := range points {
		names[index] = point.GetOr("name", "")
	}
	if !adr0114() {
		return stationAccess{refusal: "IED " + name + " has multiple servers; select an access point first"}
	}
	addresses := accessAddresses(root, name)
	chosen := only(points, func(point *xmldoc.Element) bool { return addresses[point.GetOr("name", "")] != "" })
	if chosen == nil {
		chosen = only(points, func(point *xmldoc.Element) bool { return holds(point.Find("Server"), "ReportControl") })
	}
	if chosen == nil {
		return stationAccess{refusal: "IED " + name + " has several access points with a server (" + strings.Join(names, ", ") +
			") and none is clearly the station bus (the one with the IP address or the reports), so ScadaBench cannot tell which one a SCADA uses"}
	}
	station := stationAccess{server: chosen.Find("Server"), name: chosen.GetOr("name", ""), ip: addresses[chosen.GetOr("name", "")]}
	for _, point := range points {
		if point != chosen {
			station.others = append(station.others, point)
		}
	}
	return station
}

// only is the single access point that matches, or nil when none or several do.
func only(points []*xmldoc.Element, match func(*xmldoc.Element) bool) *xmldoc.Element {
	var found *xmldoc.Element
	for _, point := range points {
		if match(point) {
			if found != nil {
				return nil
			}
			found = point
		}
	}
	return found
}

// accessAddresses are the valid IPv4 addresses of the IED's connected access points by name.
func accessAddresses(root *xmldoc.Element, iedName string) map[string]string {
	addresses := map[string]string{}
	for _, subnetwork := range root.FindAll("Communication/SubNetwork") {
		for _, access := range subnetwork.FindAll("ConnectedAP") {
			if name, ok := access.Get("iedName"); !ok || name != iedName {
				continue
			}
			raw, _ := lookup(readParameters(access.Find("Address")), "IP", "")
			if ip := ipv4(raw); ip != "" && addresses[access.GetOr("apName", "")] == "" {
				addresses[access.GetOr("apName", "")] = ip
			}
		}
	}
	return addresses
}

// stationDevices are the logical devices a SCADA sees: the chosen server's, or, when no server
// could be chosen, every server's as before.
func stationDevices(root, ied *xmldoc.Element) []*xmldoc.Element {
	if station := stationOf(root, ied); station.server != nil {
		return station.server.FindAll("LDevice")
	}
	return ied.FindAll("AccessPoint/Server/LDevice")
}

// note tells the user what the other access points carry; "" for an IED with one server.
func (s stationAccess) note() string {
	if s.name == "" || len(s.others) == 0 {
		return ""
	}
	parts := []string{}
	for _, point := range s.others {
		server := point.Find("Server")
		devices := []string{}
		for _, device := range server.FindAll("LDevice") {
			devices = append(devices, device.GetOr("inst", ""))
		}
		kind := "other data"
		switch {
		case holds(server, "SampledValueControl") || strings.Contains(strings.Join(devices, " "), "SV"):
			kind = "process bus, Sampled Values"
		case holds(server, "GSEControl") || strings.Contains(strings.Join(devices, " "), "GO"):
			kind = "process bus, GOOSE"
		}
		parts = append(parts, point.GetOr("name", "")+" ("+kind+": "+strings.Join(devices, ", ")+")")
	}
	where := s.name
	if s.ip != "" {
		where += " (station bus, " + s.ip + ")"
	}
	return "Simulates " + where + ", the access point a SCADA uses. Not simulated: " + strings.Join(parts, "; ") + "."
}

// holds is true when an element below e has the tag.
func holds(e *xmldoc.Element, tag string) bool {
	for _, child := range e.Children {
		if child.Tag == tag || holds(child, tag) {
			return true
		}
	}
	return false
}

// stationName is the chosen access point of the named IED ("" for one server or none chosen).
func stationName(root *xmldoc.Element, iedName string) string {
	for _, ied := range root.FindAll("IED") {
		if ied.GetOr("name", "") == iedName {
			return stationOf(root, ied).name
		}
	}
	return ""
}
