package netcfg

import (
	"strconv"
	"strings"

	"studio-engine/internal/labnet"
	"studio-engine/internal/pyjson"
)

// ApprovalKind is a real-card address the ScadaBench network service may add at Start and remove
// at Stop (ADR-0132). Apply writes the approval (one administrator prompt) instead of adding the
// address; the address is on the card only while its IED runs, like on the lab card.
const ApprovalKind = "approval"

// approvalPath is the service's Approved key as PowerShell names it.
const approvalPath = `HKLM:\` + labnet.ApprovedKey

// useApprovals turns the plan's address items into approvals when the service can add them
// (protocol 3) and the card is a real one: IEDs (added at Run) and SNMP devices (added when the
// devices start, ADR-0136). Addresses already on the card stay "keep", and so does an approval
// already in the service's key (approved holds those: GUID -> ip -> prefix).
func useApprovals(items []*pyjson.Object, approved map[string]map[string]int) []*pyjson.Object {
	for _, item := range items {
		f := fieldsOf(item)
		if f.text("kind") != "alias" || f.text("action") != "add" {
			continue
		}
		item.Set("kind", ApprovalKind)
		if prefix := approved[strings.ToUpper(f.text("adapter_guid"))][f.text("ip")]; prefix > 0 &&
			f.integer("prefix") == strconv.Itoa(prefix) {
			item.Set("owned", false).Set("action", "keep")
		}
	}
	return items
}

// runningHints: an approved address is on the card while its IED runs; the planner calls it
// "already on the adapter", but ScadaBench put it there and takes it off at Stop.
func runningHints(rows []Row, adapter Adapter, approved map[string]map[string]int) {
	for index, row := range rows {
		if row.Status == "already_on_adapter" && approved[strings.ToUpper(adapter.GUID)][row.Proposed] > 0 {
			rows[index].Hint = "ScadaBench put this address on while the IED runs and takes it off at Stop."
		}
	}
}

func approvalName(f *itemFields) string {
	return psString(labnet.ApprovalName(f.text("adapter_guid"), f.text("ip")))
}

// approvalApply writes the approval: one DWORD value holding the prefix.
func approvalApply(f *itemFields) string {
	return "if (-not (Test-Path " + psString(approvalPath) + ")) { New-Item -Path " + psString(approvalPath) +
		" -Force | Out-Null }; New-ItemProperty -Path " + psString(approvalPath) + " -Name " + approvalName(f) +
		" -Value " + f.integer("prefix") + " -PropertyType DWord -Force"
}

// approvalRemove deletes the approval and the address, if it is on the card now.
func approvalRemove(f *itemFields) string {
	return "Remove-ItemProperty -Path " + psString(approvalPath) + " -Name " + approvalName(f) +
		" -ErrorAction SilentlyContinue; Get-NetIPAddress -InterfaceIndex " + f.integer("adapter_index") +
		" -IPAddress " + f.text("ip") + " -ErrorAction SilentlyContinue | Remove-NetIPAddress -Confirm:$false"
}

func describeApproval(f *itemFields) string {
	if f.text("action") == "keep" {
		return "ScadaBench already puts " + f.text("ip") + " on '" + f.text("adapter_name") + "' while " +
			f.text("for") + " runs"
	}
	return "Let ScadaBench put " + f.text("ip") + "/" + f.text("prefix") + " on '" + f.text("adapter_name") +
		"' while " + f.text("for") + " runs (added at Start, removed at Stop)"
}

func describeApprovalRemoval(f *itemFields) string {
	return "Stop ScadaBench putting " + f.text("ip") + " on '" + f.text("adapter_name") + "'"
}

// approvalPresent says whether the approval is still in the service's key.
func approvalPresent(f *itemFields) bool {
	return labnet.ReadApproved()[strings.ToUpper(f.text("adapter_guid"))][f.text("ip")] > 0
}
