package signalindex

import "strings"

// Point kinds (ADR-0095, completed by ADR-0113), the same rules as the frontend's
// points/pointKind.ts kindOf61850: functional constraint + basic type, refined by the data
// object's class (cdc) where the type alone cannot tell (commands, counters, histograms,
// settings and description objects). Both implementations are checked against one shared
// table, studio/tests/fixtures/point_kinds/iec61850.json. A request without the Go-only
// parameters is answered exactly as Flask does.

// kindNames is the display order of the kinds; a kind's code is its position here.
var kindNames = [...]string{"SP", "DP", "IS", "MV", "CT", "SC", "DC", "SE", "CM", "CF", "TX", "AT", "UN"}

var kindCodes = func() map[string]uint8 {
	codes := make(map[string]uint8, len(kindNames))
	for code, name := range kindNames {
		codes[name] = uint8(code)
	}
	return codes
}()

const kindUN = uint8(len(kindNames) - 1)

var (
	// numericTypes are the IEC 61850-7-2 integer and floating point types (INT24 and the
	// unsigned forms as SCL writes them; INT64U/INT128U were accepted before and still are).
	numericTypes = setOf("INT8", "INT16", "INT24", "INT32", "INT64", "INT128", "INT8U", "INT16U", "INT24U",
		"INT32U", "INT64U", "INT128U", "FLOAT32", "FLOAT64")
	// otherTextTypes are standard types that are not a process value: object references,
	// currencies, bit strings of control blocks and checks, entry ids and network addresses.
	otherTextTypes = setOf("ObjRef", "Currency", "Check", "PhyComAddr", "TrgOps", "OptFlds", "SvOptFlds",
		"LogOptFlds", "EntryID")
	helperTypes  = setOf("Quality", "Timestamp", "EntryTime")
	settingFCs   = setOf("SP", "SG", "SE", "CF")
	describingFC = setOf("DC", "EX")
	// serviceFCs are service tracking and the control block attributes (7-2 FCs SR, BR, RP, LG,
	// GO, MS, US): configuration of a service, never a process value.
	serviceFCs  = setOf("SR", "BR", "RP", "LG", "GO", "MS", "US")
	commandKind = map[string]string{"SPC": "SC", "DPC": "DC", "APC": "SE", "BAC": "SE", "INC": "SE",
		"ENC": "SE", "BSC": "SE", "ISC": "SE"}
	settingClasses    = setOf("SPG", "ING", "ENG", "ORG", "TSG", "CUG", "VSG", "ASG", "CURVE", "CSG")
	describingClasses = setOf("DPL", "LPL", "CSD")
)

func setOf(items ...string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

// isTextType is a visible, Unicode or octet string or another non-value standard type.
func isTextType(basic string) bool {
	return strings.HasPrefix(basic, "VisString") || strings.HasPrefix(basic, "Unicode") ||
		strings.HasPrefix(basic, "Octet") || otherTextTypes[basic]
}

// pointKind is kindOf61850(fc, basicType, dataClass); the rules apply in this order.
func pointKind(fc, basic, class string) string {
	switch {
	case fc == "SV" || fc == "BL": // substitution and blocking are set by an operator
		return "CF"
	case helperTypes[basic]:
		return "AT"
	case fc == "CO":
		if kind, ok := commandKind[class]; ok {
			return kind
		}
		if numericTypes[basic] {
			return "SE"
		}
		return "CM"
	case settingFCs[fc]:
		return "CF"
	case describingFC[fc] || serviceFCs[fc]:
		return "TX"
	case fc != "ST" && fc != "MX" && fc != "OR": // not a 7-2 functional constraint
		return "UN"
	case settingClasses[class]:
		return "CF"
	case describingClasses[class] || isTextType(basic):
		return "TX"
	case basic == "Dbpos":
		return "DP"
	case basic == "BOOLEAN":
		return "SP"
	case numericTypes[basic]:
		return numericKind(fc, class)
	case basic == "Enum" || basic == "Tcmd":
		return "IS"
	}
	return "UN"
}

// numericKind is the kind of a number in ST, MX or OR: measured in MX; in status a counter
// (BCR, SEC) or a histogram value (HST) by its class, otherwise an integer status.
func numericKind(fc, class string) string {
	switch {
	case fc == "MX" || class == "HST":
		return "MV"
	case class == "BCR" || class == "SEC":
		return "CT"
	}
	return "IS"
}

// dataClassOf is the class of the longest object reference that equals ref or is followed by
// "." in it (points/dataClasses.ts dataClassOf).
func dataClassOf(classes map[string]string, ref string) string {
	for at := len(ref); at > 0; at = strings.LastIndex(ref[:at], ".") {
		if class, ok := classes[ref[:at]]; ok {
			return class
		}
	}
	return ""
}

// kindCode is the kind code of one indexed signal for these classes.
func kindCode(item *signal, classes map[string]string) uint8 {
	if !item.fcOK || !item.kindOK {
		return kindUN
	}
	return kindCodes[pointKind(item.fc, item.kind, dataClassOf(classes, item.ref))]
}
