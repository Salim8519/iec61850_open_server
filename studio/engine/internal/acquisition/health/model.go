package health

import (
	"fmt"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// ModelRows is model_rows: data points, datasets and report blocks of a connected real IED
// from the model comparison the live session made.
func ModelRows(ied, comparison *pyjson.Object, when any) []any {
	summary := pyv.GetObj(comparison, "summary")
	missing := pyv.Get(summary, "missing_on_device")
	mismatched := pyv.Get(summary, "type_mismatch")
	extra := pyv.Get(summary, "extra_on_device")
	var model *pyjson.Object
	switch {
	case pyv.Truthy(missing) || pyv.Truthy(mismatched):
		model = row("model", "Data points", "problem", fmt.Sprintf("%s SCD points are missing on the device and "+
			"%s have another data type.", pyv.PyStr(missing), pyv.PyStr(mismatched)), when, "mms_read", nil)
	case pyv.Truthy(extra):
		model = row("model", "Data points", "warning", fmt.Sprintf("The device has %s points the SCD does not "+
			"describe. The SCD may be older than the device configuration.", pyv.PyStr(extra)), when, "mms_read", nil)
	default:
		model = row("model", "Data points", "ok", "Every SCD data point exists on the device with the same type.",
			when, "mms_read", nil)
	}
	rows := []any{model}
	datasets := false
	for _, item := range pyv.Objects(pyv.GetList(comparison, "datasets")) {
		status := pyv.Get(item, "status")
		if pyv.Equal(status, "match") {
			continue
		}
		message := "This dataset is missing on the device."
		if !pyv.Equal(status, "missing_on_device") {
			message = fmt.Sprintf("Members differ: %d missing and %d extra on the device.",
				pyLen(pyv.Get(item, "missing")), pyLen(pyv.Get(item, "extra")))
		}
		rows = append(rows, row("datasets", pyv.Get(item, "ref"), "problem", message, when, "mms_read", nil))
		datasets = true
	}
	if !datasets {
		rows = append(rows, row("datasets", "Datasets", "ok", "Every SCD dataset matches the device.", when, "mms_read", nil))
	}
	for _, item := range pyv.Objects(pyv.GetList(comparison, "reports")) {
		if !pyv.Equal(pyv.Get(item, "status"), "match") {
			rows = append(rows, row("reports", pyv.Get(item, "ref"), "problem", "This report block is missing on the device.",
				when, "mms_read", nil))
		}
	}
	return rows
}

// ReportSettingRows is report_setting_rows: dataset and confRev of each readable report
// block (rcbs = {blocks, checked_at}) against the SCD. A nil rcbs (Python None) means
// polling access, which never reads report blocks.
func ReportSettingRows(ied, rcbs *pyjson.Object) []any {
	if rcbs == nil {
		return []any{notChecked("report_settings", "Report blocks",
			"Polling access never reads report blocks, so their settings were not compared.")}
	}
	rows := []any{}
	when := pyv.Get(rcbs, "checked_at")
	blocks := pyv.GetObj(rcbs, "blocks")
	for _, report := range pyv.Objects(pyv.List(pyv.GetOr(ied, "reports", []any{}))) {
		var block *pyjson.Object
		if ref, isText := pyv.Get(report, "ref").(string); isText {
			block = pyv.GetObj(blocks, ref)
		}
		if !filled(block) || !pyv.Truthy(pyv.Get(block, "readable")) {
			continue
		}
		expected := ""
		if value := pyv.Get(report, "dataset_ref"); pyv.Truthy(value) {
			expected = pyv.PyStr(value)
		}
		var differences []string
		onDevice := pyv.PyStr(pyv.GetOr(block, "dataset", ""))
		if strings.ReplaceAll(onDevice, ".", "$") != strings.ReplaceAll(expected, ".", "$") {
			differences = append(differences, fmt.Sprintf("dataset is %s on the device but %s in the SCD",
				orNone(pyv.Get(block, "dataset")), orNone(expected)))
		}
		if pyv.Has(report, "conf_rev") && !pyv.Equal(pyv.Get(block, "conf_rev"), pyv.Get(report, "conf_rev")) {
			differences = append(differences, fmt.Sprintf("confRev is %s on the device but %s in the SCD",
				pyv.PyStr(pyv.Get(block, "conf_rev")), pyv.PyStr(pyv.Get(report, "conf_rev"))))
		}
		ref := pyv.Get(report, "ref")
		if len(differences) > 0 {
			scd := pyv.O("dataset", expected, "conf_rev", pyv.Get(report, "conf_rev"))
			rows = append(rows, row("report_settings", ref, "problem", Sentence(differences), when, "mms_read",
				pyv.O("device", block, "scd", scd)))
		} else {
			rows = append(rows, row("report_settings", ref, "ok", "Dataset and confRev match the SCD.", when, "mms_read", nil))
		}
	}
	if len(rows) == 0 {
		rows = append(rows, row("report_settings", "Report blocks", "ok", "No report blocks to compare.", when, "mms_read", nil))
	}
	return rows
}

// orNone is f"{value or '(none)'}".
func orNone(value any) string {
	if !pyv.Truthy(value) {
		return "(none)"
	}
	return pyv.PyStr(value)
}
