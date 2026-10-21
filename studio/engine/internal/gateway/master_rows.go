package gateway

import (
	"math/big"
	"sort"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

var groupNames = map[string]string{
	"single": "Single point", "double": "Double point", "step": "Step position", "bitstring": "Bitstring 32",
	"normalized": "Measured (normalised)", "scaled": "Measured (scaled)", "float": "Measured (float)",
	"counter": "Integrated total", "binary": "Binary input", "double_binary": "Double-bit binary input",
	"analog": "Analog input", "binary_output": "Binary output status", "analog_output": "Analog output status",
	"coil": "Coil", "discrete_input": "Discrete input", "holding_register": "Holding register",
	"input_register": "Input register", "frozen_counter": "Frozen counter"}

// commandFor is what the device can be commanded with for a point of a group.
var commandFor = map[string]map[string]string{
	"iec104": {"single": "single", "double": "double", "normalized": "setpoint_normalized",
		"scaled": "setpoint_scaled", "float": "setpoint_float"},
	"dnp3":   {"binary_output": "binary_output", "analog_output": "analog_output_float"},
	"modbus": {"coil": "coil", "holding_register": "register"}}

var registerWrite = map[string]string{"float32": "registers_float", "int32": "registers_int32", "uint32": "registers_int32"}

type qualityBit struct {
	bit  int64
	word string
}

var (
	iec104Quality = []qualityBit{{0x80, "invalid"}, {0x40, "not topical"}, {0x20, "substituted"},
		{0x10, "blocked"}, {0x01, "overflow"}}
	dnp3Quality = []qualityBit{{0x02, "restart"}, {0x04, "communication lost"}, {0x08, "remote forced"},
		{0x10, "local forced"}}
	// IEEE 1815 flag bits 5 and 6 mean different things per object (change 0089).
	dnp3Bit5 = map[string]string{"binary": "chatter filter", "double_binary": "chatter filter",
		"counter": "rollover", "frozen_counter": "rollover", "analog": "over range", "analog_output": "over range"}
	dnp3Bit6 = map[string]string{"counter": "discontinuity", "frozen_counter": "discontinuity",
		"analog": "reference error", "analog_output": "reference error"}
	doubleText = map[int64]string{0: "intermediate (00)", 1: "open (OFF)", 2: "closed (ON)", 3: "bad state (11)"}
)

const linkLost = "not current (link lost)"

// bitSet is bool(quality & bit) for an int quality (Python's two's complement for negatives).
func bitSet(quality any, bit int64) bool {
	if value, ok := quality.(*big.Int); ok {
		return new(big.Int).And(value, big.NewInt(bit)).Sign() != 0
	}
	number, _ := pytext.Int64(quality)
	return number&bit != 0
}

// QualityWords is quality_words(protocol, quality, group) for an int quality; an empty
// list means good. group "" stands for None.
func QualityWords(protocol string, quality any, group string) []string {
	words := []string{}
	switch protocol {
	case "iec104":
		for _, item := range iec104Quality {
			if bitSet(quality, item.bit) {
				words = append(words, item.word)
			}
		}
	case "dnp3":
		if !bitSet(quality, 0x01) {
			words = append(words, "offline")
		}
		for _, item := range dnp3Quality {
			if bitSet(quality, item.bit) {
				words = append(words, item.word)
			}
		}
		if word, found := dnp3Bit5[group]; found && bitSet(quality, 0x20) {
			words = append(words, word)
		}
		if word, found := dnp3Bit6[group]; found && bitSet(quality, 0x40) {
			words = append(words, word)
		}
	default:
		if pytext.Truthy(quality) {
			words = append(words, "no answer")
		}
	}
	return words
}

// ValueText is value_text(row): the value in plain words. A list or dict value of a double
// point is unhashable in Python's lookup and gives a TypeError.
func ValueText(row *pyjson.Object) (string, error) {
	value := get(row, "value")
	group, err := subscript(row, "group")
	if err != nil {
		return "", err
	}
	switch group {
	case "double", "double_binary":
		if err := hashable(value); err != nil {
			return "", err
		}
		if number, ok := doubleKey(value); ok {
			return doubleText[number], nil
		}
		return pytext.Str(value), nil
	case "single", "binary", "binary_output", "coil", "discrete_input":
		if pytext.Truthy(value) {
			return "ON", nil
		}
		return "OFF", nil
	}
	if f, ok := value.(float64); ok {
		return pytext.FormatG(f, 6), nil
	}
	if value == nil {
		return "", nil
	}
	return pytext.Str(value), nil
}

// doubleKey finds a DOUBLE_TEXT key equal to value (True == 1 == 1.0 in a dict lookup).
func doubleKey(value any) (int64, bool) {
	for number := range doubleText {
		if pytext.IsNumber(value) && pytext.Equal(value, number) {
			return number, true
		}
	}
	return 0, false
}

// GroupName is group_name(protocol, group); "" for a group outside GROUPS.
func GroupName(protocol, group string) string {
	if protocol == "dnp3" && group == "counter" {
		return "Counter" // "Integrated total" is the IEC 104 word
	}
	return groupNames[group]
}

// command is _command(protocol, row): the device command for a point of this group.
func command(protocol string, row *pyjson.Object) (any, error) {
	group, _ := get(row, "group").(string)
	name, found := commandFor[protocol][group]
	if !found {
		return nil, nil
	}
	if name != "register" {
		return name, nil
	}
	write, known, err := lookupName(registerWrite, get(row, "type"))
	if err != nil || !known {
		return name, err
	}
	return registerWrite[write], nil
}

// Rows is rows(table, protocol, unticked, labels, link_open): display rows sorted by group
// then address; values are stale while the link is down. labels nil is an empty dict.
func Rows(table *pyjson.Object, protocol string, unticked []string, labels *pyjson.Object, linkOpen bool) ([]any, error) {
	hidden := map[string]bool{}
	for _, key := range unticked {
		hidden[key] = true
	}
	order := map[string]int{}
	for index, group := range groups {
		order[group] = index
	}
	list := []*pyjson.Object{}
	for _, key := range table.Keys() {
		row, _ := table.Get(key)
		list = append(list, row.(*pyjson.Object))
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := get(list[i], "group").(string), get(list[j], "group").(string)
		if order[a] != order[b] {
			return order[a] < order[b]
		}
		return cmpInt(get(list[i], "address"), get(list[j], "address")) < 0
	})
	result := []any{}
	for _, row := range list {
		quality, err := subscript(row, "quality")
		if err != nil {
			return nil, err
		}
		group, _ := get(row, "group").(string)
		words := QualityWords(protocol, quality, group)
		if !linkOpen { // a last-known value is never presented as good
			words = append([]string{linkLost}, words...)
		}
		key, _ := get(row, "key").(string)
		label := any("")
		if labels != nil {
			label = getOr(labels, key, "")
		}
		text, err := ValueText(row)
		if err != nil {
			return nil, err
		}
		qualityText := "good"
		if len(words) > 0 {
			qualityText = strings.Join(words, ", ")
		}
		device, err := command(protocol, row)
		if err != nil {
			return nil, err
		}
		result = append(result, copyObject(row).Set("group_name", GroupName(protocol, group)).Set("label", label).
			Set("ticked", !hidden[key]).Set("value_text", text).Set("quality_text", qualityText).
			Set("good", len(words) == 0).Set("stale", !linkOpen).Set("command", device))
	}
	return result, nil
}
