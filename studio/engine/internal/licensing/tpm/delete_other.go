//go:build !windows

package tpm

// deleteKey does nothing outside Windows.
func deleteKey(string) {}
