//go:build windows

package labnet

import (
	"os/exec"
	"strings"
	"testing"
)

// The script's DNS guard, run by Windows PowerShell with a Set-DnsClient that fails (an interface
// that does not exist): the reason comes out, the next step runs and the script succeeds. A
// failing step outside the guard still fails the script. Nothing on the PC is changed.
func TestCardScriptDnsGuardInPowerShell(t *testing.T) {
	script := cardScript("{guid}")
	guard := script[strings.Index(script, "try {"):strings.Index(script, "Disable-NetAdapterBinding")]
	guard = strings.ReplaceAll(guard, "$a.ifIndex", "2147480000")
	output, err := powershell("$ErrorActionPreference='Stop'; " + guard + "Write-Output 'next step ran'")
	if err != nil {
		t.Fatalf("guarded DNS failure stopped the script: %v\n%s", err, output)
	}
	if dnsNote(output) == "" || !strings.Contains(output, "next step ran") {
		t.Fatalf("output %q", output)
	}
	if output, err := powershell("$ErrorActionPreference='Stop'; " + guard +
		"Set-NetIPInterface -InterfaceIndex 2147480000 -InterfaceMetric 9999; Write-Output 'kept going'"); err == nil ||
		strings.Contains(output, "kept going") {
		t.Fatalf("an unguarded failure did not stop the script: %q", output)
	}
}

func powershell(script string) (string, error) {
	output, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	return string(output), err
}
