package labnet

import "strings"

// dnsSkipped starts the line the card script writes when Windows refused the DNS step.
const dnsSkipped = "dns-skipped: "

// cardScript is the PowerShell that configures the lab card with the given GUID. Every step stops
// the script on an error, except the DNS registration step (ADR-0134): on some PCs Windows has no
// working DNS Client settings class (HRESULT 0x80041010), and a card that never uses DNS works the
// same without it. That step's reason is written out instead, and the IPv6 step still runs.
func cardScript(guid string) string {
	return "$ErrorActionPreference='Stop'; $a = Get-NetAdapter -IncludeHidden | Where-Object { " +
		"$_.InterfaceGuid -eq '" + guid + "' }; if (-not $a) { throw 'card not found' }; " +
		"if ($a.Name -ne '" + CardName + "') { Rename-NetAdapter -InputObject $a -NewName '" + CardName + "' }; " +
		"Set-NetIPInterface -InterfaceIndex $a.ifIndex -AddressFamily IPv4 -InterfaceMetric 9999 -Dhcp Disabled; " +
		"try { Set-DnsClient -InterfaceIndex $a.ifIndex -RegisterThisConnectionsAddress $false } " +
		"catch { Write-Output ('" + dnsSkipped + "' + $_.Exception.Message) }; " +
		"Disable-NetAdapterBinding -Name '" + CardName + "' -ComponentID ms_tcpip6"
}

// dnsNote is the note for the install result when the script skipped the DNS step ("" otherwise).
func dnsNote(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if reason, found := strings.CutPrefix(strings.TrimSpace(line), dnsSkipped); found {
			return "Windows did not turn off DNS registration on the card (" + strings.TrimSpace(reason) +
				"); the card works without it."
		}
	}
	return ""
}
