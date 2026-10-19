package project

import (
	"strconv"

	"studio-engine/internal/pyjson"
)

// LabTitle and LabNotice tell a lab-network project that GOOSE now stays inside
// ScadaBench unless shared (projects/goose_reach.py, ADR-0073).
const (
	LabTitle  = "GOOSE now stays inside ScadaBench unless you share it."
	LabNotice = "GOOSE now stays inside ScadaBench by default. Before, a lab-network project also sent " +
		"its GOOSE to the network. To keep doing that, open Settings > Network & GOOSE and " +
		"switch on \"Also share with the network\"."
)

// migrationNotice is the notice restore() shows: the upgrade message for a legacy format,
// the GOOSE notice alone for a current lab project saved before ADR-0073, else nil.
func migrationNotice(version int64, notices []any, backup string, gooseNote bool) *pyjson.Object {
	legacy := version < CurrentFormat
	if !legacy {
		if !gooseNote {
			return nil
		}
		return pyjson.NewObject().Set("message", LabTitle).Set("details", []any{LabNotice}).Set("backup", "")
	}
	details := append([]any{}, notices...)
	if len(details) == 0 {
		if version >= 11 {
			details = []any{"No equipment was changed."}
		} else {
			details = []any{"Existing controls remain in standard IEC 61850 mode."}
		}
	}
	if gooseNote {
		details = append(details, LabNotice)
	}
	return pyjson.NewObject().Set("message", upgradeMessage(version, len(notices) > 0)).
		Set("details", details).Set("backup", backup)
}

func upgradeMessage(version int64, changed bool) string {
	head := "Project upgraded to format " + strconv.Itoa(CurrentFormat)
	standard := " Every command follows standard IEC 61850."
	switch {
	case version == 14:
		return head + "." + standard
	case version == 13:
		return head + ": the gateway tool can now also act as SCADA master." + standard
	case version >= 11:
		if changed {
			return head + ". See the details for what changed."
		}
		return head + ". Nothing in your diagram changed."
	case version >= 10:
		return head + ": network settings were added. Every IED is still simulated on this PC only, exactly as before."
	case version >= 9:
		return head + ": push buttons no longer accept wires."
	}
	return head + "." + standard
}
