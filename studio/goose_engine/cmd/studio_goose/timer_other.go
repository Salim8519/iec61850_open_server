//go:build !windows

package main

// millisecondTimer is only needed on Windows.
func millisecondTimer() func() { return func() {} }
