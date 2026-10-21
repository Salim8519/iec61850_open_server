//go:build !windows

package netcfg

import "errors"

// localRuleNames off Windows: there is no rule store, so the PowerShell path decides.
func localRuleNames() ([]string, error) { return nil, errors.New("no Windows Firewall here") }
