package livedom

import (
	"math"
	"sort"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// compareHints are model_compare.HINTS.
var compareHints = map[string]string{
	"missing_on_device": "The SCD has this point but the device does not. The device may run a " +
		"different configuration (CID) than the SCD expects.",
	"extra_on_device": "The device has this point but the SCD does not. Usually harmless, but it can " +
		"mean the SCD is older than the device configuration.",
	"type_mismatch": "Both have the point but with different data types, so clients may misread it.",
}

// controlBlockFCs are the control-block service attributes (GoCB, RCB, logs, SV); they are
// not engineering data points.
var controlBlockFCs = map[string]bool{"GO": true, "GS": true, "MS": true, "US": true, "RP": true,
	"BR": true, "LG": true, "SV": true}

// leafKey is the (ref, fc) tuple of a data point.
type leafKey struct{ ref, fc string }

// Compare classifies every reference of the SCD model against a device discovery (compare)
// and returns {"summary", "signals", "datasets", "reports"} in Python key order. References
// and functional constraints are expected to be strings, as the SCL importer and the
// browse worker write them.
func Compare(ied, discovery *pyjson.Object) *pyjson.Object {
	device := indexDevice(discovery)
	rows := []any{}
	seen := map[leafKey]bool{}
	counts := map[string]int64{}
	addRow := func(row *pyjson.Object) {
		rows = append(rows, row)
		counts[pyv.GetStr(row, "category")]++
	}
	for _, signal := range pyv.Objects(pyv.GetList(ied, "signals")) {
		key := leafKey{pyv.PyStr(pyv.Get(signal, "ref")), pyv.PyStr(pyv.Get(signal, "fc"))}
		seen[key] = true
		category, deviceType := "match", any(nil)
		mmsType, found := device.leaves[key]
		switch {
		case !found:
			category = "missing_on_device"
		case !TypeMatches(pyv.GetStr(signal, "kind"), mmsType):
			category, deviceType = "type_mismatch", describeDevice(mmsType)
		default:
			deviceType = describeDevice(mmsType)
		}
		addRow(pyv.O("ref", pyv.Get(signal, "ref"), "fc", pyv.Get(signal, "fc"), "category", category,
			"scd_type", pyv.Get(signal, "kind"), "device_type", deviceType, "hint", compareHints[category]))
	}
	extras := make([]leafKey, 0, len(device.leaves))
	for key := range device.leaves {
		if !seen[key] {
			extras = append(extras, key)
		}
	}
	sort.Slice(extras, func(i, j int) bool {
		if extras[i].ref != extras[j].ref {
			return extras[i].ref < extras[j].ref
		}
		return extras[i].fc < extras[j].fc
	})
	for _, key := range extras {
		addRow(pyv.O("ref", key.ref, "fc", key.fc, "category", "extra_on_device", "scd_type", nil,
			"device_type", describeDevice(device.leaves[key]), "hint", compareHints["extra_on_device"]))
	}
	datasets, different := compareDatasets(ied, device.datasets)
	reports := []any{}
	var reportsMissing int64
	for _, report := range pyv.Objects(pyv.List(pyv.GetOr(ied, "reports", []any{}))) {
		status := "missing_on_device"
		if ref, ok := pyv.Get(report, "ref").(string); ok && device.reports[ref] {
			status = "match"
		} else {
			reportsMissing++
		}
		reports = append(reports, pyv.O("ref", pyv.Get(report, "ref"), "status", status))
	}
	summary := pyv.O("match", counts["match"], "missing_on_device", counts["missing_on_device"],
		"extra_on_device", counts["extra_on_device"], "type_mismatch", counts["type_mismatch"],
		"datasets_different", different, "reports_missing", reportsMissing)
	return pyv.O("summary", summary, "signals", rows, "datasets", datasets, "reports", reports)
}

// compareDatasets compares each SCD dataset with the device's members (sorted differences).
func compareDatasets(ied *pyjson.Object, device map[string][]string) ([]any, int64) {
	result := []any{}
	var different int64
	for _, dataset := range pyv.Objects(pyv.List(pyv.GetOr(ied, "datasets", []any{}))) {
		ref := pyv.Get(dataset, "ref")
		text, isText := ref.(string)
		members, found := device[text]
		if !isText || !found {
			different++
			result = append(result, pyv.O("ref", ref, "status", "missing_on_device", "missing", []any{},
				"extra", []any{}))
			continue
		}
		expected := map[string]bool{}
		for _, item := range pyv.List(pyv.GetOr(dataset, "fcda_refs", []any{})) {
			expected[pyv.PyStr(item)] = true
		}
		actual := map[string]bool{}
		for _, member := range members {
			actual[member] = true
		}
		missing, extra := setMinus(expected, actual), setMinus(actual, expected)
		status := "match"
		if len(missing) > 0 || len(extra) > 0 {
			status = "different"
			different++
		}
		result = append(result, pyv.O("ref", ref, "status", status, "missing", pyv.Strings(missing),
			"extra", pyv.Strings(extra)))
	}
	return result, different
}

// setMinus is sorted(a - b).
func setMinus(a, b map[string]bool) []string {
	result := []string{}
	for item := range a {
		if !b[item] {
			result = append(result, item)
		}
	}
	sort.Strings(result)
	return result
}

// deviceIndex is _device_index: leaves by (ref, fc), dataset members by dataset reference
// and the set of report block references the device has.
type deviceIndex struct {
	leaves   map[leafKey]any
	datasets map[string][]string
	reports  map[string]bool
}

func indexDevice(discovery *pyjson.Object) deviceIndex {
	index := deviceIndex{leaves: map[leafKey]any{}, datasets: map[string][]string{}, reports: map[string]bool{}}
	for _, device := range pyv.Objects(pyv.List(pyv.GetOr(discovery, "devices", []any{}))) {
		for _, node := range pyv.Objects(pyv.List(pyv.GetOr(device, "nodes", []any{}))) {
			owner := pyv.PyStr(pyv.Get(device, "name")) + "/" + pyv.PyStr(pyv.Get(node, "name"))
			for _, leaf := range pyv.Objects(pyv.List(pyv.GetOr(node, "leaves", []any{}))) {
				fc := pyv.PyStr(pyv.Get(leaf, "fc"))
				if !controlBlockFCs[fc] {
					index.leaves[leafKey{pyv.PyStr(pyv.Get(leaf, "ref")), fc}] = pyv.Get(leaf, "mms_type")
				}
			}
			members := pyv.Obj(pyv.GetOr(node, "dataset_members", pyjson.NewObject()))
			for _, name := range members.Keys() {
				entry := pyv.Obj(pyv.Get(members, name))
				var refs []string
				for _, member := range pyv.List(pyv.GetOr(entry, "members", []any{})) {
					refs = append(refs, strings.SplitN(pyv.PyStr(member), "[", 2)[0])
				}
				index.datasets[owner+"$"+name] = refs
			}
			for _, kind := range [][2]string{{"BR", "buffered_reports"}, {"RP", "unbuffered_reports"}} {
				for _, name := range pyv.List(pyv.GetOr(node, kind[1], []any{})) {
					index.reports[owner+"."+kind[0]+"."+pyv.PyStr(name)] = true
				}
			}
		}
	}
	return index
}

// describeDevice is describe_mms for a device type: Python's dict lookup also finds a float
// equal to a type code (4.0 is "integer"), which DescribeMMS does not handle.
func describeDevice(mmsType any) string {
	if f, ok := mmsType.(float64); ok && f == math.Trunc(f) && math.Abs(f) < 1<<53 {
		if name, found := mmsNames[int64(f)]; found {
			return name
		}
		return "MMS type " + pyv.PyStr(f)
	}
	return DescribeMMS(mmsType)
}
