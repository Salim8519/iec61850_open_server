package realcmd

import (
	"strconv"
	"strings"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/typedcmd"
)

// typedClasses are the command classes beyond SPC/DPC that Devices sends (ADR-0104 list).
var typedClasses = map[string]bool{"INC": true, "ENC": true, "APC": true, "BAC": true, "BSC": true, "ISC": true}

// commandOf describes the control of this reference: SPC/DPC from the IED's `controls`, the
// other classes from its typed list (Extended). An unknown reference gives an empty Ref.
func (s *Service) commandOf(ied *pyjson.Object, ref string) (typedcmd.Control, string) {
	if entry := controlOf(ied, ref); entry != nil {
		return typedcmd.FromStandard(entry), text(entry, "status")
	}
	if s.Extended == nil {
		return typedcmd.Control{}, ""
	}
	for _, raw := range s.Extended(ied) {
		if entry, ok := raw.(*pyjson.Object); ok && text(entry, "ref") == ref && typedClasses[text(entry, "cdc")] {
			return typedcmd.FromExtended(entry), text(entry, "status")
		}
	}
	return typedcmd.Control{}, ""
}

// feedbackFC is the functional constraint of a feedback point: MX for an analogue mxVal (APC,
// BAC), ST for positions, states and tap positions.
func feedbackFC(status string) string {
	if strings.Contains(status, ".mxVal") {
		return "MX"
	}
	return "ST"
}

// want is the feedback a successful command leaves, or nil when it cannot be told in advance
// (a set point may be rounded by the device; a Tcmd step moves from wherever it is). A DPC
// shows closed 2 / open 1; an ISC Raise/Lower ends one position from `before`.
func want(target typedcmd.Control, value, before any) any {
	switch target.ValueKind {
	case "bool":
		if flag, ok := value.(bool); ok && target.CDC == "DPC" {
			return map[bool]int64{true: 2, false: 1}[flag]
		} else if ok {
			return flag
		}
	case "int", "enum":
		if _, wire, err := typedcmd.Wire(target, value, before); err == nil {
			return wire
		}
	}
	return nil
}

// observeStep reads the feedback until it shows the expected value or the time is up. Without an
// expected value it reports where the feedback went from `before`.
func (s *Service) observeStep(session Session, target typedcmd.Control, status string, value, before any) (string, bool) {
	goal := want(target, value, before)
	limit, pause := s.Observe, s.Poll
	if limit == 0 {
		limit = 3 * time.Second
	}
	if pause == 0 {
		pause = 500 * time.Millisecond
	}
	deadline := time.Now().Add(limit)
	seen := "not read"
	for {
		read, found, err := session.ReadOne(status, feedbackFC(status))
		if err == nil && found {
			seen = valueText(target.CDC, read)
			if (goal != nil && read == goal) || (goal == nil && before != nil && read != before) {
				return status + " is " + seen, true
			}
		}
		if time.Now().After(deadline) {
			return status + " still " + seen + " after " + limit.String(), false
		}
		time.Sleep(pause)
	}
}

// valueText says a feedback value in words: positions for a DPC, on/off, else the number.
func valueText(cdc string, value any) string {
	switch v := value.(type) {
	case bool:
		return map[bool]string{true: "on", false: "off"}[v]
	case int64:
		if cdc == "DPC" {
			return positionText(v)
		}
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'g', 6, 64)
	case string:
		return v
	}
	return "unknown"
}
