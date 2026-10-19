package netcfg

import (
	"strings"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Read-only Windows Firewall questions for the station gateway (firewall_rules.py, ADR-0062).

const firewallReadError = "Windows Firewall could not be read: "

// GatewayRulesScript is the script list_gateway_rules runs.
const GatewayRulesScript = "Get-NetFirewallRule -Name 'ScadaBench-Gateway-*' -ErrorAction SilentlyContinue | " +
	"Select-Object -ExpandProperty Name"

// BlockingRule is an enabled inbound Windows Block rule for the gateway worker.
type BlockingRule struct {
	Name    string
	Display string
}

// ListGatewayRules is firewall_rules.list_gateway_rules.
func ListGatewayRules(run Executor) ([]string, error) {
	text, err := shellQuery(run, GatewayRulesScript, 20*time.Second, firewallReadError)
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(text), nil
}

// BlockingRulesScript is the script blocking_rules runs for program.
func BlockingRulesScript(program string) string {
	return "$found = @(Get-NetFirewallRule -Direction Inbound -Action Block -Enabled True " +
		"-ErrorAction SilentlyContinue | Where-Object { ($_ | Get-NetFirewallApplicationFilter).Program " +
		"-eq " + psString(program) + " -and ($_ | Get-NetFirewallPortFilter).Protocol -in 'TCP','Any' } " +
		"| ForEach-Object { @{ name = $_.Name; display = $_.DisplayName } }); " +
		"ConvertTo-Json -InputObject $found -Compress"
}

// BlockingRules is firewall_rules.blocking_rules: enabled inbound TCP Block rules naming
// exactly this program.
func BlockingRules(program string, run Executor) ([]BlockingRule, error) {
	text, err := shellQuery(run, BlockingRulesScript(program), 60*time.Second, firewallReadError)
	if err != nil {
		return nil, err
	}
	found := []BlockingRule{}
	if text = pyStrip(text); text == "" {
		return found, nil
	}
	value, err := pyjson.Decode([]byte(text))
	if err != nil {
		return nil, apperr.ValueError(err.Error())
	}
	list, isList := value.([]any)
	if object, isObject := value.(*pyjson.Object); isObject {
		list, isList = []any{object}, true
	}
	if !isList || len(objects(list)) != len(list) {
		return nil, &pyCrash{"Windows Firewall rules are not objects (Python: TypeError)"}
	}
	for _, rule := range objects(list) {
		name, ok := rule.Get("name")
		if !ok {
			return nil, apperr.KeyError("name")
		}
		found = append(found, BlockingRule{Name: pyStr(name), Display: field(rule, "display", "")})
	}
	return found, nil
}

// GatewayStateScript is the one script gateway_state runs for the clean-up list.
func GatewayStateScript(unblockNames []string) string {
	quoted := make([]string, len(unblockNames))
	for index, name := range unblockNames {
		quoted[index] = psString(name)
	}
	names := strings.Join(quoted, ",")
	if names == "" {
		names = "$null"
	}
	return "$g = @(Get-NetFirewallRule -Name 'ScadaBench-Gateway-*' -ErrorAction SilentlyContinue | " +
		"ForEach-Object { $_.Name }); " +
		"$d = @(foreach ($n in @(" + names + ")) { if ($n) { $r = Get-NetFirewallRule -Name $n -ErrorAction " +
		"SilentlyContinue; if ($r -and \"$($r.Enabled)\" -eq 'False') { $n } } }); " +
		"ConvertTo-Json -InputObject @{ gateway = $g; disabled = $d } -Compress"
}

// textSet is set(value or []) for a decoded JSON value (a string gives its characters).
func textSet(value any) map[string]bool {
	found := map[string]bool{}
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			found[pyStr(item)] = true
		}
	case string:
		for _, r := range v {
			found[string(r)] = true
		}
	}
	return found
}

// GatewayState is firewall_rules.gateway_state: ScadaBench's gateway rule names, and which
// of the given Windows rules exist and are switched off. One PowerShell call (change 0089).
func GatewayState(unblockNames []string, run Executor) (gateway, disabled map[string]bool, err error) {
	text, err := shellQuery(run, GatewayStateScript(unblockNames), 60*time.Second, firewallReadError)
	if err != nil {
		return nil, nil, err
	}
	object := pyjson.NewObject()
	if text = pyStrip(text); text != "" {
		value, err := pyjson.Decode([]byte(text))
		if err != nil {
			return nil, nil, apperr.ValueError(err.Error())
		}
		decoded, ok := value.(*pyjson.Object)
		if !ok {
			return nil, nil, &pyCrash{"Windows Firewall answer is not an object (Python: AttributeError)"}
		}
		object = decoded
	}
	g, _ := object.Get("gateway")
	d, _ := object.Get("disabled")
	return textSet(g), textSet(d), nil
}
