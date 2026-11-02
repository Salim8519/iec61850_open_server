//go:build !windows

package portholder

// Find reads no table off Windows (the product ships for Windows): the holder stays unnamed.
func Find(address string, port int) (Holder, bool) { return Holder{}, false }
