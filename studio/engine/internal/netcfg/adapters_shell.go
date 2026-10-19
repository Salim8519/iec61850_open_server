package netcfg

import (
	"time"
)

// AdapterScript is adapters.SCRIPT, the PowerShell fallback listing.
const AdapterScript = `
$ErrorActionPreference = 'Stop'
$routes = @(Get-NetRoute -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue |
  Select-Object -ExpandProperty ifIndex)
$result = @(Get-NetAdapter | ForEach-Object {
  $a = $_
  $ips = @(Get-NetIPAddress -InterfaceIndex $a.ifIndex -AddressFamily IPv4 -ErrorAction SilentlyContinue |
    ForEach-Object { @{ ip = $_.IPAddress; prefix = [int]$_.PrefixLength;
      skip_as_source = [bool]$_.SkipAsSource; origin = [string]$_.PrefixOrigin } })
  @{ guid = [string]$a.InterfaceGuid; name = $a.Name; description = $a.InterfaceDescription;
     status = [string]$a.Status; index = [int]$a.ifIndex; mac = $a.MacAddress;
     speed = [string]$a.LinkSpeed; virtual = [bool]$a.Virtual;
     default_route = ($routes -contains $a.ifIndex); addresses = $ips }
})
ConvertTo-Json -InputObject $result -Depth 5 -Compress
`

// RulesScript is adapters.RULES_SCRIPT: names of ScadaBench's MMS firewall rules.
const RulesScript = "Get-NetFirewallRule -Name 'SubstationStudio-MMS-*' -ErrorAction SilentlyContinue | " +
	"Select-Object -ExpandProperty Name"

const (
	adapterShellTimeout = 20 * time.Second
	adapterShellError   = "Windows could not list network adapters: "
)

// SystemAdapters is adapters.list_adapters: the IP Helper list (milliseconds), and PowerShell
// only when that fails. Native defaults to RawAdapters; Shell runs the fallback.
type SystemAdapters struct {
	Native func() ([]any, error)
	Shell  Executor
}

// Adapters lists and normalizes the adapters.
func (s SystemAdapters) Adapters() ([]Adapter, error) {
	native := s.Native
	if native == nil {
		native = RawAdapters
	}
	raw, err := native()
	if err == nil {
		return Normalize(raw), nil
	}
	if s.Shell == nil {
		return nil, err
	}
	text, err := shellQuery(s.Shell, AdapterScript, adapterShellTimeout, adapterShellError)
	if err != nil {
		return nil, err
	}
	return NormalizeJSON(text)
}

// nonEmptyLines is [line.strip() for line in text.splitlines() if line.strip()].
func nonEmptyLines(text string) []string {
	found := []string{}
	for _, line := range pySplitlines(text) {
		if line = pyStrip(line); line != "" {
			found = append(found, line)
		}
	}
	return found
}

// ListStudioRules is adapters.list_studio_rules: MMS rules ScadaBench created that still exist.
func ListStudioRules(run Executor) ([]string, error) {
	text, err := shellQuery(run, RulesScript, adapterShellTimeout, adapterShellError)
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(text), nil
}
