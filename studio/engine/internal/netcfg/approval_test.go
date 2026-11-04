package netcfg

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

const ethGUID = "{7B29C0B1-72E8-4C7D-9FAA-6A695893211D}"

func aliasItem(ip, action string) *pyjson.Object {
	return pyjson.NewObject().Set("kind", "alias").Set("adapter_guid", ethGUID).Set("adapter_index", 15).
		Set("adapter_name", "Ethernet").Set("ip", ip).Set("prefix", 24).Set("for", "P545").
		Set("owned", action == "add").Set("action", action)
}

// IED and SNMP addresses both become approvals (ADR-0136): nothing is added that Stop leaves.
func TestIEDAndSNMPAddressesBecomeApprovals(t *testing.T) {
	items := []*pyjson.Object{aliasItem("192.168.230.21", "add"), aliasItem("192.168.230.24", "add"),
		aliasItem("192.168.40.111", "keep"), aliasItem("192.168.40.21", "add")}
	approved := map[string]map[string]int{ethGUID: {"192.168.230.24": 24}}
	useApprovals(items, approved)
	kinds := []string{}
	for _, item := range items {
		kinds = append(kinds, pyStr(get(item, "kind"))+"/"+pyStr(get(item, "action")))
	}
	want := "approval/add approval/keep alias/keep approval/add"
	if strings.Join(kinds, " ") != want {
		t.Fatalf("kinds %v want %s", kinds, want)
	}
	if truthy(get(items[1], "owned")) {
		t.Fatal("an existing approval is owned again")
	}
}

func TestApprovalCommandsKeyAndValidation(t *testing.T) {
	item := aliasItem("192.168.230.21", "add").Set("kind", ApprovalKind)
	apply, err := ApplyCommand(item)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{`HKLM:\SYSTEM\CurrentControlSet\Services\ScadaBenchLabNet\Parameters\Approved`,
		"-Name '" + ethGUID + " 192.168.230.21' -Value 24 -PropertyType DWord"} {
		if !strings.Contains(apply, part) {
			t.Fatalf("apply %s lacks %s", apply, part)
		}
	}
	if strings.Contains(apply, "New-NetIPAddress") {
		t.Fatal("an approval adds the address at Apply")
	}
	remove, _ := RemoveCommand(item)
	if !strings.Contains(remove, "Remove-ItemProperty") || !strings.Contains(remove, "Remove-NetIPAddress -Confirm:$false") {
		t.Fatalf("remove %s", remove)
	}
	if key, _ := ItemKey(item); key != "approval:"+ethGUID+":192.168.230.21" {
		t.Fatalf("key %s", key)
	}
	if err := validateItem(item, vectorProgram); err != nil {
		t.Fatalf("a valid approval refused: %v", err)
	}
	if err := validateItem(item.Set("prefix", 40), vectorProgram); err == nil {
		t.Fatal("a bad prefix accepted")
	}
	text, _ := Describe(aliasItem("192.168.230.21", "add").Set("kind", ApprovalKind))
	if !strings.Contains(text, "added at Start, removed at Stop") {
		t.Fatalf("description %s", text)
	}
}
