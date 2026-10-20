//go:build !release

package main

import "io/fs"

// payload is empty outside the release build, so the exe explains how to build it.
var payload fs.FS = emptyPayload{}

type emptyPayload struct{}

func (emptyPayload) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }
