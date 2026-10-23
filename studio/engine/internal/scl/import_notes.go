package scl

import (
	"errors"
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
)

// importNotes are what one IED import changed or could not use without failing: each becomes
// one plain-words warning on the IED (ADR-0114: explicit, narrow, visible, reversible).
type importNotes struct {
	defaultTriggers int      // DAs that got the IEC 61850-7-3 trigger options
	textReplaced    []string // text values whose " or line breaks were replaced
	commandOnly     []string // commandable objects without status
	oddCommands     []string // commandable objects whose shape ScadaBench cannot operate
	reported        map[string]bool
}

// warnings renders the notes in a fixed order.
func (n *importNotes) warnings() []string {
	var result []string
	if !adr0114() {
		return nil
	}
	if n.defaultTriggers > 0 {
		result = append(result, strconv.Itoa(n.defaultTriggers)+" attribute(s) have no trigger options (dchg/qchg/dupd) "+
			"in the file. ScadaBench applied the IEC 61850-7-3 trigger options of each attribute (for example stVal "+
			"dchg, q qchg, mag dchg+dupd) so reports and GOOSE send their changes.")
	}
	if len(n.textReplaced) > 0 {
		result = append(result, "These text values contain a double quote or a line break, which the device model "+
			"cannot hold. ScadaBench simulates them with ' instead of \" and a space instead of each line break; the "+
			"file itself is unchanged: "+strings.Join(n.textReplaced, ", "))
	}
	if len(n.commandOnly) > 0 {
		result = append(result, strconv.Itoa(len(n.commandOnly))+" command(s) have no status attribute (stVal), so "+
			"ScadaBench cannot operate them yet (often LED or lockout resets): "+strings.Join(n.commandOnly, ", "))
	}
	if len(n.oddCommands) > 0 {
		result = append(result, strconv.Itoa(len(n.oddCommands))+" command(s) use a value or status type their "+
			"data class does not define, so ScadaBench cannot operate them: "+strings.Join(n.oddCommands, ", "))
	}
	return result
}

// odd records a commandable object whose status or value type its class does not define.
func (n *importNotes) odd(ref, cdc, leaf string, kind any) {
	if n.reported[ref] {
		return
	}
	text, _ := kind.(string)
	n.oddCommands = append(n.oddCommands, ref+" ("+cdc+": "+leaf+" is "+text+")")
	n.markReported(ref)
}

// markReported keeps one note per object when the Python warning already named it.
func (n *importNotes) markReported(ref string) {
	if n.reported == nil {
		n.reported = map[string]bool{}
	}
	n.reported[ref] = true
}

// simulatedText is the value the native model can hold: the cfg format ends a string at the
// first " and a record at a line break (ADR-0114).
func simulatedText(value string) (string, bool) {
	if !adr0114() || !strings.ContainsAny(value, "\"\r\n") {
		return value, false
	}
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.NewReplacer("\"", "'", "\r", " ", "\n", " ").Replace(value)
	return value, true
}

// located prefixes an import error with where it happened in the SCL ("at LD/LN.DO.DA").
// Errors that already carry a location are returned unchanged.
func located(err error, where string) error {
	var app *apperr.Error
	if !adr0114() || !errors.As(err, &app) || strings.HasPrefix(app.Message, "at ") {
		return err
	}
	copied := *app
	copied.Message = "at " + where + ": " + app.Message
	return &copied
}

// forIED names the IED in an import error: "IED X at LD/LN.DO.DA: ..." or "IED X: ...".
func forIED(err error, name string) error {
	var app *apperr.Error
	if !adr0114() || !errors.As(err, &app) {
		return err
	}
	copied := *app
	if strings.HasPrefix(app.Message, "at ") {
		copied.Message = "IED " + name + " " + app.Message
	} else {
		copied.Message = "IED " + name + ": " + app.Message
	}
	return &copied
}
