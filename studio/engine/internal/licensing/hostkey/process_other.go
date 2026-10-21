//go:build !windows

package hostkey

import "errors"

type noProcess struct{}

// SystemProcess has no helper off Windows.
func SystemProcess() Process { return noProcess{} }

func (noProcess) Running() bool      { return false }
func (noProcess) Start(string) error { return errors.New("the host helper needs Windows") }

// Acquire always succeeds off Windows (there is nothing to share).
func Acquire() (release func(), ok bool) { return func() {}, true }
