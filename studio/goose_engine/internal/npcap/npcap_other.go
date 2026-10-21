//go:build !windows

// Package npcap is Windows-only; elsewhere GOOSE stays inside the app.
package npcap

import "errors"

// ErrMissing means Npcap is not available on this system.
var ErrMissing = errors.New("Npcap is only available on Windows")

// Available always reports that Npcap is missing.
func Available() error { return ErrMissing }

// Handle is never opened outside Windows.
type Handle struct{}

// Open always fails outside Windows.
func Open(string) (*Handle, error) { return nil, ErrMissing }

// Next never delivers frames.
func (*Handle) Next() ([]byte, error) { return nil, ErrMissing }

// Event is never signalled.
func (*Handle) Event() uintptr { return 0 }

// Send never transmits.
func (*Handle) Send([]byte) error { return ErrMissing }

// Close does nothing.
func (*Handle) Close() {}
