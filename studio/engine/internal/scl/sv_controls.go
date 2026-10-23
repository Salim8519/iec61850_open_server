package scl

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

var (
	sampleModes = map[string]int64{"SmpPerPeriod": 0, "SmpPerSec": 1, "SecPerSmp": 2}
	svScale     = map[string]float64{"TCTR": 0.001, "TVTR": 0.01} // 9-2LE: mA and 10 mV
)

// svChannels is sv_controls._channels: names and scales of a 9-2LE dataset, or nil.
func svChannels(fcdas []*xmldoc.Element) ([]any, []any) {
	var whole, split []*xmldoc.Element
	for _, item := range fcdas {
		if item.GetOr("daName", "") == "" {
			whole = append(whole, item)
		} else {
			split = append(split, item)
		}
	}
	var objects []*xmldoc.Element
	switch {
	case len(whole) == 8 && len(split) == 0:
		objects = whole
	case len(split) == 16 && len(whole) == 0 && pairedChannels(split):
		for i := 0; i < len(split); i += 2 {
			objects = append(objects, split[i])
		}
	default:
		return nil, nil
	}
	names, scales := []any{}, []any{}
	for _, item := range objects {
		class, _ := item.Get("lnClass")
		scale, ok := svScale[class]
		if !ok {
			return nil, nil
		}
		do, hasDO := item.Get("doName")
		names = append(names, item.GetOr("prefix", "")+class+item.GetOr("lnInst", "")+"."+noneOr(do, hasDO))
		scales = append(scales, scale)
	}
	return names, scales
}

func pairedChannels(split []*xmldoc.Element) bool {
	for i := 0; i < len(split); i += 2 {
		a, b := split[i], split[i+1]
		if a.GetOr("daName", "") != "instMag.i" || b.GetOr("daName", "") != "q" {
			return false
		}
		for _, key := range []string{"lnClass", "lnInst", "doName"} {
			va, pa := a.Get(key)
			vb, pb := b.Get(key)
			if va != vb || pa != pb {
				return false
			}
		}
	}
	return true
}

// svControls is sv_controls.sv_controls for one LN0.
func svControls(ln0 *xmldoc.Element, domain, inst string, smv []*pyjson.Object, iedName string, warnings *[]string,
	rejected map[[2]string]bool) []*pyjson.Object {
	addresses := map[[2]string]*pyjson.Object{}
	for _, item := range smv {
		addresses[[2]string{get(item, "ld_inst").(string), get(item, "cb_name").(string)}] = item
	}
	datasets := map[string]*xmldoc.Element{}
	for _, item := range ln0.FindAll("DataSet") {
		if name, ok := item.Get("name"); ok {
			datasets[name] = item
		}
	}
	result := []*pyjson.Object{}
	for _, element := range ln0.FindAll("SampledValueControl") {
		name := element.GetOr("name", "")
		where := "Sampled Values block " + iedName + "/" + inst + "/" + name
		address := addresses[[2]string{inst, name}]
		if address == nil {
			if !rejected[[2]string{inst, name}] || !adr0114() { // an unusable SMV address was already explained (ADR-0114)
				*warnings = append(*warnings, where+" has no network address (SMV) in the SCD; it is not listened to")
			}
			continue
		}
		var numbers []any
		readable := true
		for _, field := range []struct{ key, fallback string }{{"confRev", "0"}, {"smpRate", "0"}, {"nofASDU", "1"}} {
			value, err := pyrules.IntBig(element.GetOr(field.key, field.fallback), 10)
			if err != nil {
				readable = false
				break
			}
			numbers = append(numbers, intValue(value))
		}
		if !readable {
			*warnings = append(*warnings, where+" has an unreadable confRev, smpRate or nofASDU; it is not listened to")
			continue
		}
		var dataset *xmldoc.Element
		if datSet, ok := element.Get("datSet"); ok {
			dataset = datasets[datSet]
		}
		var channels, scales []any
		members := int64(0)
		if dataset != nil {
			fcdas := dataset.FindAll("FCDA")
			channels, scales = svChannels(fcdas)
			members = int64(len(fcdas))
		}
		layout := "raw"
		if channels != nil {
			layout = "9-2LE"
		} else {
			channels, scales = []any{}, []any{}
		}
		mode, ok := sampleModes[element.GetOr("smpMod", "SmpPerPeriod")]
		if !ok {
			mode = 0
		}
		datSet := element.GetOr("datSet", "")
		result = append(result, pyjson.NewObject().Set("name", name).Set("ld_inst", inst).
			Set("svcb_ref", domain+"/LLN0."+name).Set("sv_id", element.GetOr("smvID", "")).
			Set("dataset", datSet).Set("dataset_ref", domain+"/LLN0$"+datSet).Set("conf_rev", numbers[0]).
			Set("smp_rate", numbers[1]).Set("smp_mod", mode).Set("nof_asdu", numbers[2]).
			Set("mac", get(address, "mac")).Set("app_id", get(address, "app_id")).
			Set("vlan_id", get(address, "vlan_id")).Set("layout", layout).Set("channels", channels).
			Set("scale", scales).Set("members", members))
	}
	return result
}
