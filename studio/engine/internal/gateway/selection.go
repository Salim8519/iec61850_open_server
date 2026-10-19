package gateway

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// The smart point list (gateway_selection.py): positions, alarms and trips, measurement
// magnitudes and device health; GGIO spare inputs only when the SCD uses them.
var (
	statusClasses      = []string{"XCBR", "XSWI", "CSWI", "CILO", "GGIO", "SIMG", "SIML", "SARC", "SPDC", "SCBR", "SOPM"}
	measurementClasses = []string{"MMXU", "MMXN", "MSQI", "MHAI", "MMTR", "MSTA", "GGIO", "ATCC", "YLTC"}
	healthObjects      = []string{"Health", "Beh", "PhyHealth"}
	priority           = []string{"double", "command_double", "command_single", "single", "integer", "float"}
	harmonicTotals     = []string{"ThdA", "ThdPhV", "ThdPPV", "ThdAmp", "ThdVol"}
)

// afterDevice is ref.split("/", 1)[-1].
func afterDevice(ref string) string {
	if index := strings.IndexByte(ref, '/'); index >= 0 {
		return ref[index+1:]
	}
	return ref
}

// NodeClass is the LN class of a leaf reference (re.search(r"([A-Z]{4})\d*$", node)).
func NodeClass(ref string) string {
	node := afterDevice(ref)
	if index := strings.IndexByte(node, '.'); index >= 0 {
		node = node[:index]
	}
	if node == "LLN0" {
		return "LLN0"
	}
	node = strings.TrimSuffix(node, "\n") // "$" also matches before one final newline
	end := len(node)
	for end > 0 {
		r, size := utf8.DecodeLastRuneInString(node[:end])
		if !unicode.Is(unicode.Nd, r) { // \d is any Unicode decimal digit
			break
		}
		end -= size
	}
	if end < 4 {
		return ""
	}
	for _, c := range []byte(node[end-4 : end]) {
		if c < 'A' || c > 'Z' {
			return ""
		}
	}
	return node[end-4 : end]
}

// nodeClassOf is node_class(value): AttributeError when the reference is not a str.
func nodeClassOf(ref any) (string, error) {
	text, ok := ref.(string)
	if !ok {
		return "", noAttribute(ref, "split")
	}
	return NodeClass(text), nil
}

func dataObject(ref string) string {
	parts := strings.Split(afterDevice(ref), ".")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// SmartKind is smart_kind(signal) for a well-formed signal; "" stands for None.
func SmartKind(signal *pyjson.Object) string {
	kind, _ := smartKind(signal)
	return kind
}

// smartKind is smart_kind with Python's KeyError/AttributeError for malformed signals.
func smartKind(signal *pyjson.Object) (string, error) {
	values := [3]any{}
	for index, key := range []string{"ref", "kind", "fc"} {
		value, err := subscript(signal, key)
		if err != nil {
			return "", err
		}
		values[index] = value
	}
	refValue, kind, fc := values[0], values[1], values[2]
	lnclass, err := nodeClassOf(refValue)
	if err != nil {
		return "", err
	}
	ref := refValue.(string)
	status := fc == "ST" && strings.HasSuffix(ref, ".stVal")
	switch {
	case status && kind == "Dbpos":
		return "double", nil
	case status && kind == "BOOLEAN":
		return pick(contains(statusClasses, lnclass), "single"), nil
	case fc == "ST" && strings.HasSuffix(ref, ".general") && kind == "BOOLEAN":
		return pick(strings.HasPrefix(lnclass, "P") || strings.HasPrefix(lnclass, "R"), "single"), nil
	case status && contains(healthObjects, dataObject(ref)):
		if lnclass == "LLN0" || lnclass == "LPHD" {
			if kind == "Enum" {
				return "integer", nil
			}
			isInt, err := startsWith(kind, "INT")
			return pick(isInt, "integer"), err
		}
		return "", nil
	case lnclass == "MHAI" && !contains(harmonicTotals, dataObject(ref)):
		return "", nil
	}
	if fc == "MX" && strings.HasSuffix(ref, ".mag.f") {
		isFloat, err := startsWith(kind, "FLOAT")
		if err != nil || isFloat {
			return pick(contains(measurementClasses, lnclass), "float"), err
		}
	}
	if fc == "MX" && strings.HasSuffix(ref, ".mag.i") {
		isInt, err := startsWith(kind, "INT")
		if err != nil || isInt {
			return pick(contains(measurementClasses, lnclass), "integer"), err
		}
	}
	return "", nil
}

func pick(condition bool, kind string) string {
	if condition {
		return kind
	}
	return ""
}

// startsWith is value.startswith(prefix): AttributeError when value is not a str.
func startsWith(value any, prefix string) (bool, error) {
	text, ok := value.(string)
	if !ok {
		return false, noAttribute(value, "startswith")
	}
	return strings.HasPrefix(text, prefix), nil
}

// DatasetRefs is dataset_refs(ied) for a well-formed IED: nil stands for None.
func DatasetRefs(ied *pyjson.Object) map[string]bool {
	refs, _ := datasetRefs(ied)
	return refs
}

// datasetRefs collects the leaf references of every dataset. A non-str member still makes
// the set non-empty (Python's set is not None then) but can never match a str reference.
func datasetRefs(ied *pyjson.Object) (map[string]bool, error) {
	datasets, err := iterate(getOr(ied, "datasets", []any{}))
	if err != nil {
		return nil, err
	}
	refs, members := map[string]bool{}, 0
	for _, dataset := range datasets {
		value, err := getAny(dataset, "signal_refs", []any{})
		if err != nil {
			return nil, err
		}
		items, err := iterate(value)
		if err != nil {
			return nil, err
		}
		for _, ref := range items {
			if err := hashable(ref); err != nil {
				return nil, err
			}
			members++
			if text, ok := ref.(string); ok {
				refs[text] = true
			}
		}
	}
	if members == 0 {
		return nil, nil
	}
	return refs, nil
}

// Engineered is engineered(signal, refs): false for a GGIO value nothing in the SCD uses.
func Engineered(signal *pyjson.Object, refs map[string]bool) bool {
	ref, _ := get(signal, "ref").(string)
	if NodeClass(ref) != "GGIO" || refs == nil {
		return true
	}
	return refs[ref] || pytext.Truthy(get(signal, "description"))
}

// rankOf is rank(candidate): (generic, PRIORITY.index(kind)).
func rankOf(kind string, ref any) (bool, int, error) {
	lnclass, err := nodeClassOf(ref)
	if err != nil {
		return false, 0, err
	}
	for index, item := range priority {
		if item == kind {
			return lnclass == "GGIO", index, nil
		}
	}
	return lnclass == "GGIO", len(priority), nil
}
