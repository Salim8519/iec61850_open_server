//go:build !windows

package netcfg

// sendARP is arp_probe._answers off Windows: nothing answers.
func sendARP(address string) bool { return false }
