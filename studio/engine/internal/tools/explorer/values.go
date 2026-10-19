package explorer

import (
	"strconv"
	"strings"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// mmsNames is MMS_NAMES of domain/live_value.py.
var mmsNames = map[int64]string{0: "array", 1: "structure", 2: "boolean", 3: "bit string", 4: "integer",
	5: "unsigned", 6: "float", 7: "octet string", 8: "visible string", 9: "generalized time", 10: "binary time",
	11: "BCD", 12: "object id", 13: "unicode string", 14: "UTC time", 15: "access error"}

var validity = map[int64]string{0: "good", 1: "reserved", 2: "invalid", 3: "questionable"}

var qualityFlags = []struct {
	mask int64
	name string
}{{4, "overflow"}, {8, "out of range"}, {16, "bad reference"}, {32, "oscillatory"}, {64, "failure"},
	{128, "old data"}, {256, "inconsistent"}, {512, "inaccurate"}, {1024, "substituted"}, {2048, "test"},
	{4096, "operator blocked"}, {8192, "derived"}}

var dbpos = map[int64]string{0: "intermediate", 1: "off (open)", 2: "on (closed)", 3: "bad state"}

// TypeName is type_name: the MMS type word, or "type <n>". Keys match like a Python dict
// (2.0 and True find 2 and 1).
func TypeName(mmsType any) string {
	if key, ok := dictKey(mmsType); ok {
		if name, found := mmsNames[key]; found {
			return name
		}
	}
	return "type " + pytext.Str(mmsType)
}

func dictKey(value any) (int64, bool) {
	if f, isFloat := value.(float64); isFloat {
		if f == float64(int64(f)) {
			return int64(f), true
		}
		return 0, false
	}
	return pytext.Int64(value)
}

// QualityText is quality_text: validity, then the set flags.
func QualityText(bits int64) string {
	text := validity[bits&3]
	var flags []string
	for _, flag := range qualityFlags {
		if bits&flag.mask != 0 {
			flags = append(flags, flag.name)
		}
	}
	if len(flags) == 0 {
		return text
	}
	return text + ": " + strings.Join(flags, ", ")
}

// TimeText is time_text for a millisecond UTC timestamp (int, bools included as Python does).
func TimeText(milliseconds any) string {
	value, ok := pytext.Int64(milliseconds)
	if !ok || !pytext.IsIntOrBool(milliseconds) || value <= 0 {
		return "not set"
	}
	moment := time.UnixMilli(value).UTC()
	return moment.Format("2006-01-02 15:04:05.") + leftPad(strconv.FormatInt(value%1000, 10), 3) + " UTC"
}

func leftPad(text string, width int) string {
	for len(text) < width {
		text = "0" + text
	}
	return text
}

// Describe is describe(ref, raw): {"text", "status"} with status ok, warning, problem or error.
func Describe(ref string, raw any) *pyjson.Object {
	attribute := ref[strings.LastIndexByte(ref, '.')+1:]
	result := func(text, status string) *pyjson.Object {
		return pyjson.NewObject().Set("text", text).Set("status", status)
	}
	switch value := raw.(type) {
	case *pyjson.Object:
		if code, found := value.Get("access_error"); found {
			return result("device refused the read (access error "+pytext.Str(code)+")", "error")
		}
		if bits, found := value.Get("bits"); found {
			width, _ := pytext.Int64(bits)
			if attribute == "q" && width >= 2 {
				le, _ := pytext.Int64(get(value, "le"))
				text := QualityText(le)
				status := "warning"
				if text == "good" {
					status = "ok"
				} else if strings.HasPrefix(text, "invalid") {
					status = "problem"
				}
				return result(text, status)
			}
			be, _ := pytext.Int64(get(value, "be"))
			if attribute == "stVal" && width == 2 {
				status := "warning"
				if be == 1 || be == 2 {
					status = "ok"
				}
				return result(dbpos[be], status)
			}
			return result("bits "+binary(be, width), "ok")
		}
		if octets, found := value.Get("octets"); found {
			if !pytext.Truthy(octets) {
				return result("(empty)", "ok")
			}
			return result(pytext.Str(octets), "ok")
		}
		return result("value type not shown", "warning")
	case bool:
		if value {
			return result("true", "ok")
		}
		return result("false", "ok")
	case float64:
		return result(pytext.FormatG(value, 6), "ok")
	case nil:
		return result("no value", "warning")
	}
	if (attribute == "t" || attribute == "T") && pytext.IsInt(raw) {
		return result(TimeText(raw), "ok")
	}
	return result(pytext.Str(raw), "ok")
}

// binary is format(value, "0<width>b").
func binary(value, width int64) string {
	sign := ""
	if value < 0 {
		sign, value = "-", -value
	}
	digits := strconv.FormatInt(value, 2)
	for int64(len(sign)+len(digits)) < width {
		digits = "0" + digits
	}
	return sign + digits
}
