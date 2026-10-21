//go:build !windows

package hostkey

import "errors"

type noRunKey struct{}

// SystemRunKey has no Run key off Windows.
func SystemRunKey() RunKey { return noRunKey{} }

func (noRunKey) Get() (string, bool) { return "", false }
func (noRunKey) Set(string) error    { return errors.New("the Run key needs Windows") }
func (noRunKey) Delete() error       { return nil }
