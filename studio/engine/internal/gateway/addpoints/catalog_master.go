package addpoints

import "strings"

// MasterType is one choice of the SCADA master's type list (ADR-0110): the other direction of
// the same catalog. Group is the point group the master expects the device to report (the
// native master's group names, "" for a command only); Command is the master command kind it
// sends ("" for a monitored point). Modbus types carry their table and data type; a DNP3 CROB
// carries its control code; StatusGroup is a status a device may report for a command point.
type MasterType struct {
	Key         string `json:"kind"`
	Name        string `json:"name"`
	TypeID      string `json:"type_id"`
	MostUsed    bool   `json:"most_used"`
	Group       string `json:"group"`
	Command     string `json:"command"`
	Table       string `json:"table,omitempty"`
	DataType    string `json:"data_type,omitempty"`
	Code        string `json:"code,omitempty"`
	StatusGroup string `json:"status_group,omitempty"`
}

// Registers is how many Modbus registers (or bits) one value of the type takes.
func (t MasterType) Registers() int64 {
	if t.DataType == "int32" || t.DataType == "float32" {
		return 2
	}
	return 1
}

func mt(key, name, typeID string, most bool, group, command string) MasterType {
	return MasterType{Key: key, Name: name, TypeID: typeID, MostUsed: most, Group: group, Command: command}
}

func reg(key, name, typeID string, most bool, table, dataType, command string) MasterType {
	group := strings.TrimSuffix(table, "s")
	return MasterType{Key: key, Name: name, TypeID: typeID, MostUsed: most, Group: group, Command: command,
		Table: table, DataType: dataType}
}

func crob(key, name, typeID string, most bool, code string) MasterType {
	return MasterType{Key: key, Name: name, TypeID: typeID, MostUsed: most, Command: "binary_output", Code: code,
		StatusGroup: "binary_output"}
}

// MasterCatalog is the master's type list per protocol, most used first. The keys of the
// gateway catalog's Master field are entries here, so Copy from gateway maps every gateway
// kind. Type ids follow the native master (iec104_decode.cpp, dnp3_values.cpp, modbus_reads.cpp).
var MasterCatalog = map[string][]MasterType{
	"iec104": {
		mt("single", "Single point", "M_SP 1/30", true, "single", ""),
		mt("double", "Double point", "M_DP 3/31", true, "double", ""),
		mt("float", "Measured value, float", "M_ME_NC 13/36", true, "float", ""),
		mt("scaled", "Measured value, scaled", "M_ME_NB 11/35", true, "scaled", ""),
		mt("command_double", "Double command", "C_DC 46/59", true, "", "double"),
		mt("command_single", "Single command", "C_SC 45/58", true, "", "single"),
		mt("normalized", "Measured value, normalised", "M_ME_NA 9/34", false, "normalized", ""),
		mt("step", "Step position", "M_ST 5/32", false, "step", ""),
		mt("bitstring", "Bitstring of 32 bits", "M_BO 7/33", false, "bitstring", ""),
		mt("counter", "Integrated total (counter)", "M_IT 15/37", false, "counter", ""),
		mt("regulating", "Regulating step command", "C_RC 47/60", false, "", "regulating"),
		mt("setpoint_float", "Set point command, float", "C_SE_NC 50/63", false, "", "setpoint_float"),
		mt("setpoint_scaled", "Set point command, scaled", "C_SE_NB 49/62", false, "", "setpoint_scaled"),
		mt("setpoint_normalized", "Set point command, normalised", "C_SE_NA 48/61", false, "", "setpoint_normalized"),
	},
	"dnp3": {
		mt("binary", "Binary input", "g1v2 / g2v2", true, "binary", ""),
		mt("double_binary", "Double-bit binary input", "g3v2 / g4v2", true, "double_binary", ""),
		mt("analog", "Analog input", "g30 / g32", true, "analog", ""),
		crob("crob", "Binary output command (CROB)", "g12v1 latch", true, "latch"),
		crob("crob_trip_close", "CROB close / trip", "g12v1 close/trip", false, "trip_close"),
		mt("counter", "Counter", "g20 / g22", false, "counter", ""),
		mt("frozen_counter", "Frozen counter", "g21 / g23", false, "frozen_counter", ""),
		mt("binary_output", "Binary output status", "g10v2", false, "binary_output", ""),
		mt("analog_output", "Analog output status", "g40", false, "analog_output", ""),
		{Key: "ao_float", Name: "Analog output command, float", TypeID: "g41v3", Command: "analog_output_float",
			StatusGroup: "analog_output"},
		{Key: "ao_int32", Name: "Analog output command, 32-bit", TypeID: "g41v1", Command: "analog_output_int",
			StatusGroup: "analog_output"},
		{Key: "ao_int16", Name: "Analog output command, 16-bit", TypeID: "g41v2", Command: "analog_output_int16",
			StatusGroup: "analog_output"},
	},
	"modbus": {
		reg("coil", "Coil", "FC1 read, FC5 write", true, "coils", "bool", "coil"),
		reg("discrete_input", "Discrete input", "FC2", true, "discrete_inputs", "bool", ""),
		reg("holding_float32", "Holding register, float32", "FC3, 2 registers", true, "holding_registers", "float32", "registers_float"),
		reg("holding_int16", "Holding register, int16", "FC3, 1 register", true, "holding_registers", "int16", "register"),
		reg("input_float32", "Input register, float32", "FC4, 2 registers", true, "input_registers", "float32", ""),
		reg("input_int16", "Input register, int16", "FC4, 1 register", true, "input_registers", "int16", ""),
		reg("holding_uint16", "Holding register, uint16", "FC3, 1 register", false, "holding_registers", "uint16", "register"),
		reg("holding_int32", "Holding register, int32", "FC3, 2 registers", false, "holding_registers", "int32", "registers_int32"),
		reg("input_uint16", "Input register, uint16", "FC4, 1 register", false, "input_registers", "uint16", ""),
		reg("input_int32", "Input register, int32", "FC4, 2 registers", false, "input_registers", "int32", ""),
	},
}

// MasterTypes is the master type list of a protocol (nil for an unknown protocol).
func MasterTypes(protocol string) []MasterType { return MasterCatalog[protocol] }

// MasterEntry is the master type with key in a protocol.
func MasterEntry(protocol, key string) (MasterType, bool) {
	for _, entry := range MasterCatalog[protocol] {
		if entry.Key == key {
			return entry, true
		}
	}
	return MasterType{}, false
}

// MasterLabel is how a master type is written in the Excel template's Type list.
func MasterLabel(entry MasterType) string { return entry.Name + " (" + entry.TypeID + ")" }

// ResolveMasterType reads a master type as a person writes it: the key, the name, the template
// label or (IEC 104) the start of the type id such as "M_DP" or "C_DC_NA_1".
func ResolveMasterType(protocol, typed string) (MasterType, bool) {
	want := fold(typed)
	if want == "" {
		return MasterType{}, false
	}
	for _, entry := range MasterCatalog[protocol] {
		if want == fold(entry.Key) || want == fold(entry.Name) || want == fold(MasterLabel(entry)) {
			return entry, true
		}
	}
	if protocol != "iec104" {
		return MasterType{}, false
	}
	for _, entry := range MasterCatalog[protocol] {
		if id := fold(strings.Fields(entry.TypeID)[0]); len(id) >= 4 && strings.HasPrefix(want, id) {
			return entry, true
		}
	}
	return MasterType{}, false
}

// MasterOfGateway is the master type a master sees for a gateway point kind of a protocol.
func MasterOfGateway(protocol, kind string) (MasterType, bool) {
	entry, found := Entry(protocol, kind)
	if !found {
		return MasterType{}, false
	}
	return MasterEntry(protocol, entry.Master)
}

// setpointMasters is the master type of a gateway set point per format (ADR-0107 formats), so
// a copied set point is sent with the value type the gateway expects.
var setpointMasters = map[string]map[string]string{
	"iec104": {"float": "setpoint_float", "scaled": "setpoint_scaled", "normalized": "setpoint_normalized"},
	"dnp3":   {"float": "ao_float", "int16": "ao_int16", "int32": "ao_int32"},
	"modbus": {"float": "holding_float32", "int16": "holding_int16", "uint16": "holding_uint16", "int32": "holding_int32"},
}

// MasterOfGatewayPoint is MasterOfGateway for one gateway point: a set point follows its format.
func MasterOfGatewayPoint(protocol, kind, format string) (MasterType, bool) {
	if key := setpointMasters[protocol][format]; key != "" && kind == "command_setpoint" {
		return MasterEntry(protocol, key)
	}
	return MasterOfGateway(protocol, kind)
}
