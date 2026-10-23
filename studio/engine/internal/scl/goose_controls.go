package scl

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

const geMultilin = "http://www.gedigitalenergy.com/multilin"

// geFunction is goose_vendor.ge_function: "enabled", "disabled" or "".
func geFunction(e *xmldoc.Element) string {
	value, _ := e.Get("{" + geMultilin + "}Function")
	value = pyrules.Lower(pyrules.Strip(value))
	if value == "enabled" || value == "disabled" {
		return value
	}
	return ""
}

func usableGoID(value string) bool {
	return value != "" && len([]rune(value)) <= 129 && !pyrules.HasSpace(value) && !strings.ContainsAny(value, "();{}")
}

// gooseGaps are what an earlier step already explained (ADR-0114): GSE addresses present but
// unusable, and datasets left out. A block using one is not published, without a second,
// contradictory warning.
type gooseGaps struct {
	rejected map[[2]string]bool
	datasets map[string]bool
}

// gooseControls is goose_controls.goose_controls for one LN0; datasets are the LN0's
// compiled datasets ({name, members}).
func gooseControls(ln0 *xmldoc.Element, domain, ldInst string, gse []*pyjson.Object,
	datasets []*pyjson.Object, iedName string, warnings *[]string, gaps gooseGaps) []*pyjson.Object {
	var controls []*pyjson.Object
	addresses := map[[2]string]*pyjson.Object{}
	for _, item := range gse {
		key := [2]string{get(item, "ld_inst").(string), get(item, "cb_name").(string)}
		addresses[key] = item
	}
	names := map[string]*pyjson.Object{}
	for _, item := range datasets {
		names[get(item, "name").(string)] = item
	}
	warn := func(text string) {
		if warnings != nil {
			*warnings = append(*warnings, text)
		}
	}
	for _, element := range ln0.FindAll("GSEControl") {
		name := element.GetOr("name", "")
		where := "GOOSE block " + iedName + "/" + ldInst + "/" + name
		if kind, ok := element.Get("type"); ok && kind != "GOOSE" {
			warn(where + " is " + kind + "; only GOOSE is published")
			continue
		}
		if geFunction(element) == "disabled" {
			warn(where + " is switched off in the file (vendor Function=Disabled), so it is not sent, like on the real device")
			continue
		}
		dataset := element.GetOr("datSet", "")
		if names[dataset] == nil {
			if gaps.datasets[dataset] {
				warn(where + " uses dataset " + dataset + ", which could not be used (see the dataset note), so it is not published")
			} else {
				warn(where + " has no dataset in this logical device, so it is not published")
			}
			continue
		}
		address := addresses[[2]string{ldInst, name}]
		if address == nil {
			if !gaps.rejected[[2]string{ldInst, name}] || !adr0114() {
				warn(where + " has no network address (GSE) in the SCD Communication section, so it is not published")
			}
			continue
		}
		reference := domain + "/LLN0$GO$" + name
		goID := element.GetOr("appID", "")
		if goID == "" {
			goID = reference
		}
		if !usableGoID(goID) {
			warn(where + " has a goID the native engine cannot represent; not published")
			continue
		}
		confText := element.GetOr("confRev", "0")
		confRev, err := pyrules.IntBig(confText, 10)
		if err != nil {
			warn(where + " has an unreadable confRev; not published")
			continue
		}
		control := pyjson.NewObject()
		for _, key := range []string{"mac", "app_id", "vlan_id", "vlan_priority", "min_time", "max_time"} {
			control.Set(key, get(address, key))
		}
		members := append([]any{}, get(names[dataset], "members").([]any)...)
		control.Set("name", name).Set("ld_inst", ldInst).Set("gocb_ref", reference).Set("go_id", goID).
			Set("dataset", dataset).Set("dataset_ref", domain+"/LLN0$"+dataset).Set("conf_rev", intValue(confRev)).
			Set("fixed_offs", element.GetOr("fixedOffs", "") == "true").Set("members", members)
		controls = append(controls, control)
	}
	return controls
}

// gooseCfgLines is goose_cfg_lines.
func gooseCfgLines(control *pyjson.Object) []string {
	minimum, maximum := "-1", "-1"
	if value := get(control, "min_time"); value != nil {
		minimum = pyStr(value)
	}
	if value := get(control, "max_time"); value != nil {
		maximum = pyStr(value)
	}
	mac := strings.ToLower(strings.ReplaceAll(get(control, "mac").(string), "-", ""))
	fixed := "0"
	if get(control, "fixed_offs").(bool) {
		fixed = "1"
	}
	return []string{
		"GC(" + get(control, "name").(string) + " " + get(control, "go_id").(string) + " " +
			get(control, "dataset").(string) + " " + pyStr(get(control, "conf_rev")) + " " + fixed + " " +
			minimum + " " + maximum + "){",
		"PA(" + pyStr(get(control, "vlan_priority")) + " " + pyStr(get(control, "vlan_id")) + " " +
			pyStr(get(control, "app_id")) + " " + mac + ");",
		"}",
	}
}
