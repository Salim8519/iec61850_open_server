// Package masterpoints is the SCADA master's own point list (ADR-0110, Go engine only): what the
// master expects at which protocol address, never linked to an IED. It checks the saved list,
// plans additions (one, a range, pasted or Excel rows, CSV, copied from the gateway list or from
// what the device sent), compares the list with what the device reported, derives the Modbus
// poll ranges and builds one-click commands. It is pure: no HTTP, no files, no workers.
package masterpoints

import (
	"math"
	"strconv"
	"strings"

	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// MaxPoints is the largest master point list per protocol (as the gateway list).
const MaxPoints = 5000

// GatewayProtocols are the protocols a copied point may come from.
var GatewayProtocols = []string{"iec104", "modbus", "dnp3"}

// addressLimit is the highest address of a protocol (IEC 104 IOA, DNP3 index, Modbus address).
var addressLimit = map[string]int64{"iec104": 16777215, "dnp3": 65535, "modbus": 65535}

// lowestAddress: IOA 0 is "no address" in IEC 104; DNP3 and Modbus count from 0.
var lowestAddress = map[string]int64{"iec104": 1, "dnp3": 0, "modbus": 0}

// AddressWord is what each protocol calls an address, for the plain-English problems.
var AddressWord = map[string]string{"iec104": "IOA", "dnp3": "index", "modbus": "address"}

// CheckPoint reads one point of the list and returns it in its stored shape (the canonical key
// order, every default set, the Modbus and command details derived from the type), or the
// plain reason it cannot be stored. The id is kept when it is one of ours.
func CheckPoint(protocol string, raw any) (*pyjson.Object, string) {
	item, ok := raw.(*pyjson.Object)
	if !ok {
		return nil, "a point must be an object"
	}
	typed, _ := field(item, "type").(string)
	entry, known := addpoints.MasterEntry(protocol, typed)
	if !known {
		return nil, "choose one of the " + ProtocolName(protocol) + " types"
	}
	address, problem := checkAddress(protocol, entry, field(item, "address"))
	if problem != "" {
		return nil, problem
	}
	name, _ := field(item, "name").(string)
	unit, _ := field(item, "unit").(string)
	if pytext.Len(pytext.Strip(name)) > 120 || pytext.Len(pytext.Strip(unit)) > 16 {
		return nil, "the name has at most 120 characters and the unit at most 16"
	}
	scale, okScale := number(field(item, "scale"), 1)
	offset, okOffset := number(field(item, "offset"), 0)
	if !okScale || scale == 0 || !okOffset {
		return nil, "the scale is a number other than 0 and the offset a number"
	}
	point := pyjson.NewObject().Set("id", keptID(field(item, "id"))).Set("name", pytext.Strip(name)).
		Set("type", entry.Key).Set("address", address).Set("scale", scale).Set("offset", offset).
		Set("unit", pytext.Strip(unit))
	if protocol == "modbus" {
		order := "big"
		if field(item, "word_order") == "little" && entry.Registers() == 2 {
			order = "little"
		}
		point.Set("table", entry.Table).Set("data_type", entry.DataType).Set("registers", entry.Registers()).
			Set("word_order", order)
	}
	if entry.Command != "" {
		if problem := commandFields(protocol, entry, item, point); problem != "" {
			return nil, problem
		}
	}
	if from, problem := checkFrom(field(item, "from")); problem != "" {
		return nil, problem
	} else if from != nil {
		point.Set("from", from)
	}
	return point, ""
}

func checkAddress(protocol string, entry addpoints.MasterType, raw any) (int64, string) {
	low, high := lowestAddress[protocol], addressLimit[protocol]-(entry.Registers()-1)
	if !pytext.IsInt(raw) || !pytext.IntIn(raw, low, high) {
		return 0, "the " + AddressWord[protocol] + " is a whole number from " + strconv.FormatInt(low, 10) +
			" to " + strconv.FormatInt(high, 10)
	}
	address, _ := pytext.Int64(raw)
	return address, ""
}

// commandFields stores how the point is commanded: select before operate and the value (or
// step) a Send uses, plus the DNP3 control code.
func commandFields(protocol string, entry addpoints.MasterType, item, point *pyjson.Object) string {
	selected := protocol != "modbus"
	if flag, isBool := field(item, "select").(bool); isBool {
		selected = flag && protocol != "modbus"
	}
	value, ok := number(field(item, "value"), DefaultValue(entry))
	if !ok {
		return "the command value is a number"
	}
	if choices := valueChoices(entry); choices != nil && !containsNumber(choices, value) {
		return "the command value of a " + entry.Name + " is " + joinNumbers(choices)
	}
	point.Set("command", entry.Command).Set("select", selected).Set("value", whole(value))
	if entry.Code != "" {
		point.Set("code", entry.Code)
	}
	return ""
}

// DefaultValue is the value a new command point sends: ON / close, Raise, 0 for set points.
func DefaultValue(entry addpoints.MasterType) float64 {
	switch entry.Command {
	case "double", "regulating":
		return 2
	case "single", "coil", "binary_output":
		return 1
	}
	return 0
}

func valueChoices(entry addpoints.MasterType) []float64 {
	switch entry.Command {
	case "double", "regulating":
		return []float64{1, 2}
	case "single", "coil", "binary_output":
		return []float64{0, 1}
	}
	return nil
}

// checkFrom reads the source of a copied point: {protocol, address, kind} of the gateway point.
func checkFrom(raw any) (*pyjson.Object, string) {
	if raw == nil {
		return nil, ""
	}
	from, ok := raw.(*pyjson.Object)
	protocol, _ := field(from, "protocol").(string)
	kind, _ := field(from, "kind").(string)
	if !ok || !contains(GatewayProtocols, protocol) || kind == "" || !pytext.IsInt(field(from, "address")) {
		return nil, "the copied-from part names the gateway protocol, address and kind"
	}
	address, _ := pytext.Int64(field(from, "address"))
	return pyjson.NewObject().Set("protocol", protocol).Set("address", address).Set("kind", kind), ""
}

// keptID keeps an id of the form "m<number>"; other ids are replaced when the list is checked.
func keptID(raw any) string {
	id, _ := raw.(string)
	if len(id) > 1 && len(id) <= 12 && id[0] == 'm' && strings.Trim(id[1:], "0123456789") == "" {
		return id
	}
	return ""
}

func number(raw any, fallback float64) (float64, bool) {
	if raw == nil {
		return fallback, true
	}
	if _, isBool := raw.(bool); isBool || !pytext.IsNumber(raw) {
		return 0, false
	}
	value, ok := pytext.Float(raw)
	return value, ok && !math.IsInf(value, 0) && !math.IsNaN(value) && math.Abs(value) <= 1e12
}

// whole keeps a whole number an int (so 2, not 2.0, reaches the worker and the page).
func whole(value float64) any {
	if value == math.Trunc(value) && math.Abs(value) < 1e15 {
		return int64(value)
	}
	return value
}

func field(item *pyjson.Object, key string) any {
	if item == nil {
		return nil
	}
	value, _ := item.Get(key)
	return value
}

func text(item *pyjson.Object, key string) string {
	value, _ := field(item, key).(string)
	return value
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func containsNumber(items []float64, value float64) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func joinNumbers(items []float64) string {
	parts := []string{}
	for _, item := range items {
		parts = append(parts, strconv.FormatFloat(item, 'f', -1, 64))
	}
	return strings.Join(parts, " or ")
}

// ProtocolName is the protocol in words.
func ProtocolName(protocol string) string {
	return map[string]string{"iec104": "IEC 104", "dnp3": "DNP3", "modbus": "Modbus"}[protocol]
}
