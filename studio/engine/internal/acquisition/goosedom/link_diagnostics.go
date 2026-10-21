package goosedom

import (
	"sort"
	"strconv"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// notSent is NOT_SENT: why a declared block is never sent, by link status.
var notSent = map[string]string{"disabled_in_file": "the file switches that block off",
	"no_dataset": "that block has no dataset", "no_address": "that block has no network address (GSE) in the file",
	"not_goose": "that block is not a GOOSE block", "not_sendable": "that block cannot be sent as written"}

const (
	unresolvedHint = "Check iedName, ldInst and srcCBName of this input in the configuration tool. " +
		"ScadaBench never guesses a publisher."
	notSentHint = "Normal for spare inputs. To use it, fix the block in the publisher's configuration tool " +
		"and export the file again."
	remoteHint = "Import that IED's file, or add it as a real device, to supervise it. " +
		"It is shown at the edge of the diagram."
	zeroHint = "Give each block its own APPID in the configuration tool if a switch or receiver filters " +
		"GOOSE by APPID."
)

func diagnostic(level, code string, ied, gocbRef any, message, hint string) *pyjson.Object {
	return pyv.O("level", level, "code", code, "ied", ied, "gocb_ref", gocbRef, "message", message, "hint", hint)
}

// linkDiagnostics is _link_diagnostics: unresolved inputs, never-sent and remote links.
func linkDiagnostics(links []*pyjson.Object, d Declarations, involved stringSet) []any {
	diagnostics := []any{}
	for _, name := range involved.sorted() {
		for _, message := range pyv.List(pyv.GetOr(d.Problems, name, []any{})) {
			diagnostics = append(diagnostics, diagnostic("problem", "unresolved_input", name, nil,
				pyv.PyStr(message), unresolvedHint))
		}
	}
	for _, link := range links {
		block := ""
		if value := pyv.Get(link, "block"); pyv.Truthy(value) {
			block = " " + pyv.PyStr(value)
		}
		where := pyv.PyStr(pyv.Get(link, "subscriber")) + " expects GOOSE" + block + " from " +
			pyv.PyStr(pyv.Get(link, "publisher"))
		status, _ := pyv.Get(link, "status").(string)
		if reason, found := notSent[status]; found {
			diagnostics = append(diagnostics, diagnostic("warning", status, pyv.Get(link, "subscriber"),
				pyv.Get(link, "gocb_ref"), where+", but "+reason+", so it is never sent.", notSentHint))
		} else if status == "remote" {
			mac := ""
			if value := pyv.Get(link, "mac"); pyv.Truthy(value) {
				mac = " (MAC " + pyv.PyStr(value) + " from the vendor input address)"
			}
			diagnostics = append(diagnostics, diagnostic("info", "remote_publisher", pyv.Get(link, "subscriber"),
				pyv.Get(link, "gocb_ref"), where+mac+", which is in none of this project's files.", remoteHint))
		}
	}
	return diagnostics
}

// addressDiagnostics is _address_diagnostics: APPID 0000, duplicate APPIDs and shared
// multicast MACs among sendable blocks.
func addressDiagnostics(blocks []*pyjson.Object) []any {
	var sendable, zero []*pyjson.Object
	for _, block := range blocks {
		if pyv.Equal(pyv.Get(block, "status"), "sendable") {
			sendable = append(sendable, block)
			if pyv.Equal(pyv.Get(block, "app_id"), int64(0)) {
				zero = append(zero, block)
			}
		}
	}
	diagnostics := []any{}
	if len(zero) > 0 {
		diagnostics = append(diagnostics, diagnostic("hint", "app_id_zero", nil, nil,
			strconv.Itoa(len(zero))+" GOOSE block(s) use APPID 0000: "+names(zero)+
				". ScadaBench tells them apart by their control block reference.", zeroHint))
	}
	for _, rule := range [...][4]string{{"app_id", "duplicate_app_id", "warning", "APPID"},
		{"mac", "shared_mac", "hint", "multicast MAC"}} {
		field, code, level, word := rule[0], rule[1], rule[2], rule[3]
		for _, group := range groupBlocks(sendable, field) {
			if len(group.members) < 2 {
				continue
			}
			shown := pyv.PyStr(group.value)
			if field == "app_id" {
				shown = pyv.Format04X(group.value)
			}
			diagnostics = append(diagnostics, diagnostic(level, code, nil, nil,
				names(group.members)+" share "+word+" "+shown+".",
				"Receivers still separate them by control block reference, but a filter on the "+word+
					" lets both through. Give each its own "+word+"."))
		}
	}
	return diagnostics
}

type blockGroup struct {
	value   any
	members []*pyjson.Object
}

// groupBlocks groups blocks by a field value (None, "" and 0 left out), sorted by str(value)
// with ties in first-seen order, like sorted(groups.items(), key=str(value)).
func groupBlocks(blocks []*pyjson.Object, field string) []blockGroup {
	var groups []blockGroup
	positions := map[string]int{}
	for _, block := range blocks {
		value := pyv.Get(block, field)
		if equalsAny(value, nil, "", int64(0)) {
			continue
		}
		key := dictKey(value)
		position, found := positions[key]
		if !found {
			position = len(groups)
			positions[key] = position
			groups = append(groups, blockGroup{value: value})
		}
		groups[position].members = append(groups[position].members, block)
	}
	sort.SliceStable(groups, func(i, j int) bool { return pyv.PyStr(groups[i].value) < pyv.PyStr(groups[j].value) })
	return groups
}
