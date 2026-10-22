// Package addpoints is the Go-only "Add points" planner of the station gateway (ADR-0105):
// the type catalog per protocol, the IED signals that fit a type, points planned from an IED,
// pasted or Excel rows checked before anything changes, the Excel template and the smart
// "add only new points" proposal. It is pure: no HTTP, no files, no workers.
package addpoints

import "strings"

// TypeEntry is one choice of the type list. Kind is the gateway point kind the worker serves;
// TypeID is the protocol's own type name shown as small secondary text.
type TypeEntry struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	TypeID   string `json:"type_id"`
	MostUsed bool   `json:"most_used"`
}

// Catalog is the one list of point types per protocol. Most used come first, in this order;
// the others show under "More types". A new gateway kind is added here (one line per protocol)
// and in gateway.Kinds or gateway.TypedKinds; nothing else lists types. Type ids follow the native worker
// (iec104_objects.cpp, dnp3_outstation.cpp, modbus_pdu.cpp).
var Catalog = map[string][]TypeEntry{
	"iec104": {
		{Kind: "single", Name: "Single point", TypeID: "M_SP 1/30", MostUsed: true},
		{Kind: "double", Name: "Double point", TypeID: "M_DP 3/31", MostUsed: true},
		{Kind: "float", Name: "Measured value, float", TypeID: "M_ME_NC 13/36", MostUsed: true},
		{Kind: "integer", Name: "Measured value, scaled", TypeID: "M_ME_NB 11/35", MostUsed: true},
		{Kind: "command_single", Name: "Single command", TypeID: "C_SC 45/58", MostUsed: true},
		{Kind: "command_double", Name: "Double command", TypeID: "C_DC 46/59", MostUsed: true},
		{Kind: "command_setpoint", Name: "Set point command", TypeID: "C_SE 48-50/61-63", MostUsed: true},
		{Kind: "command_step", Name: "Regulating step command", TypeID: "C_RC 47/60"},
		{Kind: "step_position", Name: "Step position (tap)", TypeID: "M_ST 5/32"},
	},
	"dnp3": {
		{Kind: "single", Name: "Binary input", TypeID: "g1v2 / g2v2", MostUsed: true},
		{Kind: "double", Name: "Double-bit binary input", TypeID: "g3v2 / g4v2", MostUsed: true},
		{Kind: "float", Name: "Analog input, float", TypeID: "g30v5 / g32v7", MostUsed: true},
		{Kind: "command_single", Name: "Binary output command (CROB)", TypeID: "g12v1", MostUsed: true},
		{Kind: "integer", Name: "Analog input, 32-bit", TypeID: "g30v1 / g32v3"},
		{Kind: "command_double", Name: "CROB close / trip", TypeID: "g12v1 close/trip"},
		{Kind: "command_setpoint", Name: "Analog output (set point)", TypeID: "g41v1-3"},
		{Kind: "command_step", Name: "CROB raise / lower (tap)", TypeID: "g12v1 close=raise"},
		{Kind: "step_position", Name: "Analog input, tap position", TypeID: "g30 / g32"},
	},
	"modbus": {
		{Kind: "command_single", Name: "Coil", TypeID: "FC1 read, FC5/FC15 write", MostUsed: true},
		{Kind: "single", Name: "Discrete input", TypeID: "FC2", MostUsed: true},
		{Kind: "float", Name: "Holding / input register, float", TypeID: "FC3/FC4, 2 registers", MostUsed: true},
		{Kind: "integer", Name: "Holding / input register, int32", TypeID: "FC3/FC4, 2 registers", MostUsed: true},
		{Kind: "double", Name: "Holding / input register, position", TypeID: "FC3/FC4, 1 register"},
		{Kind: "command_double", Name: "Coil, ON = close", TypeID: "FC5/FC15"},
		{Kind: "command_setpoint", Name: "Holding register set point", TypeID: "FC6/FC16"},
		{Kind: "step_position", Name: "Input register, tap position", TypeID: "FC4, int16"},
	},
}

// Types is the type list of a protocol (nil for an unknown protocol).
func Types(protocol string) []TypeEntry { return Catalog[protocol] }

// Entry is the catalog entry of a kind in a protocol.
func Entry(protocol, kind string) (TypeEntry, bool) {
	for _, entry := range Catalog[protocol] {
		if entry.Kind == kind {
			return entry, true
		}
	}
	return TypeEntry{}, false
}

// ResolveType reads a type as a person writes it in a sheet: the kind ("double"), the name
// ("Double point"), the name with its type id as the template writes it ("Double point
// (M_DP 3/31)") or the start of the type id ("M_DP", "C_SC"). Case and spaces do not matter.
func ResolveType(protocol, text string) (string, bool) {
	want := fold(text)
	if want == "" {
		return "", false
	}
	for _, entry := range Catalog[protocol] {
		if want == fold(entry.Kind) || want == fold(entry.Name) || want == fold(Label(entry)) {
			return entry.Kind, true
		}
	}
	for _, entry := range Catalog[protocol] { // a type id prefix, e.g. "M_SP" or "M_SP_NA_1"
		id := fold(strings.Fields(entry.TypeID)[0])
		if strings.HasPrefix(want, id) && len(id) >= 4 && protocol == "iec104" {
			return entry.Kind, true
		}
	}
	return "", false
}

// Label is how a type is written in the Excel template's Type list.
func Label(entry TypeEntry) string { return entry.Name + " (" + entry.TypeID + ")" }

func fold(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.ReplaceAll(text, "_", " ")), " "))
}
