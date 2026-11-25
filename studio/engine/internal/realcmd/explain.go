package realcmd

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
)

// Plain-English reasons for a refused command (ADR-0149). The device's own answer (AddCause and
// MMS error) stays in the result and at the end of the sentence; the words before it say what it
// means and what to try, for people new to IEC 61850.

// causeWords are the AddCause values of IEC 61850-7-2 in words.
var causeWords = map[int64]string{
	1:  "The relay does not support this command.",
	2:  "The relay is in Local, or another control level holds the switching authority, so it refuses remote commands. Switch it to Remote.",
	3:  "The relay did not accept the selection.",
	4:  "The switch is in a position from which this command is not possible.",
	5:  "The switch is already in that position.",
	6:  "A setting changed while the command was running.",
	7:  "It is already at its last step.",
	8:  "This function of the relay is in a mode that blocks commands (Test, Blocked or Off).",
	9:  "The process blocks the command, for example an active trip or lockout.",
	10: "The relay's own interlocking does not allow this now. Check its interlock conditions, for example the positions of the other switches of the bay.",
	11: "The relay's synchrocheck did not allow closing.",
	12: "Another command to this object is still running.",
	13: "The relay blocks commands because this function is not healthy.",
	14: "Another switch is being operated, and the relay allows only one at a time.",
	15: "The command was cancelled.",
	16: "The switch did not reach the new position in time.",
	17: "A trip stopped the command.",
	18: "The relay says the object was not selected. Select it first, then Operate.",
	19: "Another client has already selected this object.",
	20: "The relay does not give this client the right to send commands.",
	21: "The value went past the target.",
	22: "The command was stopped because the value moved away from the target.",
	23: "The command was stopped because the connection was lost.",
	24: "Another command stopped this one.",
	26: "The relay found the parts of the command inconsistent.",
	27: "Another client has locked this object.",
}

// errorWords are the MMS answers (libiec61850 IedClientError) that need words without an AddCause.
var errorWords = map[int64]string{
	21: "The relay refused access to this command.",
	22: "The relay has no such command object.",
	24: "The relay does not allow this command to be written.",
	26: "The relay cannot take the command right now.",
	32: "The relay did not accept the value or the command parts.",
}

// modeWords says what the relay's behaviour (Beh.stVal) means for a command with or without Test;
// "" when the mode does not explain a refusal.
func modeWords(behaviour any, test bool) string {
	switch behaviour {
	case int64(3), int64(4):
		if !test {
			return "The relay is in Test mode, so it acts only on test commands. Tick Test command in Command options and send again."
		}
	case int64(1), int64(2):
		if test {
			return "The command was marked as a test, but the relay is not in Test mode. Untick Test command in Command options."
		}
	case int64(5):
		return "This function is switched off in the relay (behaviour Off)."
	}
	if behaviour == int64(2) || behaviour == int64(4) {
		return "This function of the relay is blocked (behaviour Blocked)."
	}
	return ""
}

// behaviourRef is the Beh.stVal of the logical node that holds the control object.
func behaviourRef(ref string) string {
	slash := strings.Index(ref, "/")
	dot := strings.Index(ref[max(slash, 0):], ".")
	if slash < 0 || dot < 0 {
		return ""
	}
	return ref[:max(slash, 0)+dot] + ".Beh.stVal"
}

// Explain is the sentence for a refused step: the meaning first, then the device's own answer.
// behaviour is the relay's Beh.stVal of the node (nil when not read).
func Explain(result *pyjson.Object, behaviour any, test bool) string {
	raw := text(result, "error")
	cause, _ := get(result, "add_cause").(int64)
	code, _ := get(result, "client_error").(int64)
	words := causeWords[cause]
	mode := modeWords(behaviour, test)
	if mode != "" && (words == "" || cause == 8 || cause == 0) {
		words = mode
	}
	if words == "" {
		words = errorWords[code]
	}
	switch {
	case words == "" && raw == "":
		return "The relay refused the command."
	case words == "":
		return raw
	case raw == "":
		return words
	}
	return words + " Device answer: " + raw + "."
}

// stepWord is the step in words for sentences.
func stepWord(action string) string {
	return map[string]string{"select": "Select", "operate": "Operate", "cancel": "Cancel"}[action]
}

// stepEvent is the runtime event of a finished step.
func stepEvent(action string, ok bool) string {
	if !ok {
		return "real_command_failed"
	}
	return map[string]string{"select": "real_command_selected", "operate": "real_command_applied",
		"cancel": "real_command_canceled"}[action]
}

// stepSummary is one sentence for a step the device accepted.
func stepSummary(action string, result *pyjson.Object) string {
	switch action {
	case "select":
		if selected, _ := get(result, "selected").(bool); !selected {
			return "This control operates directly; nothing was reserved. Press Operate to send it."
		}
		return "Selected. Press Operate before the time runs out, or Cancel."
	case "cancel":
		if canceled, _ := get(result, "canceled").(bool); canceled {
			return "The selection was cancelled. Nothing was sent."
		}
		return "Nothing was selected, so there was nothing to cancel."
	}
	return control.Summary(result)
}
