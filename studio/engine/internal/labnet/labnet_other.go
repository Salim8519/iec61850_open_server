//go:build !windows

package labnet

import "errors"

// ErrNoService means the lab network service is not running (it exists only on Windows).
var ErrNoService = errors.New("the lab network needs Windows")

// Call is unavailable outside Windows.
func Call(Request) (Reply, error) { return Reply{}, ErrNoService }

// ServiceState is never installed outside Windows.
func ServiceState() (installed, running bool) { return false, false }

// InstalledCard is none outside Windows.
func InstalledCard() string { return "" }

// AskOn never finds a device outside Windows.
func AskOn(string, string) bool { return false }

// Ping is unavailable outside Windows.
func Ping(string, uint32) (Pinged, error) { return Pinged{}, ErrNoService }
