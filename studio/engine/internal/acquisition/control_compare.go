package acquisition

import (
	"fmt"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// compareControls classes every control as match, differs or unreadable against what the
// device answered (ADR-0138) and lists the Loc points that read Local. Pure: no I/O.
func compareControls(controls []*pyjson.Object, locs []string, values, errors *pyjson.Object) *pyjson.Object {
	rows, counts := []any{}, map[string]int64{"match": 0, "differs": 0, "unreadable": 0}
	for _, control := range controls {
		row := compareControl(control, values, errors)
		counts[pyv.GetStr(row, "result")]++
		rows = append(rows, row)
	}
	local, unreadLoc := []any{}, []any{}
	for _, ref := range locs {
		value, read := values.Get(ref)
		switch {
		case !read:
			unreadLoc = append(unreadLoc, ref)
		case value == true:
			local = append(local, ref)
		}
	}
	summary := pyv.O("total", int64(len(rows)), "match", counts["match"], "differs", counts["differs"],
		"unreadable", counts["unreadable"], "local", int64(len(local)), "loc_checked", int64(len(locs)))
	return pyv.O("summary", summary, "rows", rows, "local", local, "loc_unreadable", unreadLoc)
}

func compareControl(control, values, errors *pyjson.Object) *pyjson.Object {
	ref := pyv.GetStr(control, "ref")
	cidModel, _ := modelOf(pyv.Get(control, "model"))
	cidTimeout, _ := pyv.Int64(pyv.Get(control, "sbo_timeout_ms"))
	row := pyv.O("ref", ref, "cdc", pyv.Get(control, "cdc"), "cid_model", cidModel,
		"cid_model_text", modelWords[cidModel], "device_model", nil, "device_model_text", "",
		"cid_sbo_ms", cidTimeout, "device_sbo_ms", nil)
	raw, read := values.Get(ref + ".ctlModel")
	deviceModel, valid := modelOf(raw)
	if !read || !valid {
		reason := "the device did not answer its control model"
		if failure, found := errors.Get(ref + ".ctlModel"); found {
			reason = fmt.Sprintf("the device could not read its control model (error %s)", pyv.PyStr(failure))
		} else if read {
			reason = "the device answered a control model that is not 0 to 4 (" + pyv.PyStr(raw) + ")"
		}
		return row.Set("result", "unreadable").Set("note", reason)
	}
	row.Set("device_model", deviceModel).Set("device_model_text", modelWords[deviceModel])
	if deviceModel != cidModel {
		note := "The CID says " + modelWords[cidModel] + "; the device uses " + modelWords[deviceModel] +
			". ScadaBench sends with the device's own model."
		if deviceModel == 0 {
			note = "The device has this object as status only, so it refuses every command to it."
		}
		return row.Set("result", "differs").Set("note", note)
	}
	if isSBO(deviceModel) {
		if timeout, found := values.Get(ref + ".sboTimeout"); found {
			row.Set("device_sbo_ms", timeout)
			if number, ok := pyv.Int64(timeout); ok && cidTimeout > 0 && number != cidTimeout {
				return row.Set("result", "differs").Set("note", fmt.Sprintf(
					"The selection lasts %d ms on the device, %d ms in the CID.", number, cidTimeout))
			}
		}
	}
	return row.Set("result", "match").Set("note", "")
}
