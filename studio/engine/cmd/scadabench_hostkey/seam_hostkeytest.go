//go:build hostkeytest

package main

import (
	"encoding/hex"
	"os"

	"studio-engine/internal/licensing"
	"studio-engine/internal/licensing/hostkey"
)

// This file exists only in the end-to-end test build (go build -tags hostkeytest). It lets the
// test serve on a loopback address and trust the test license server's key. No release code
// compiles it, and a test walks the release sources to prove nothing else reads these variables.
func init() {
	testSeam = func(config *helperConfig) {
		if bind := os.Getenv("SCADABENCH_HOSTKEY_TEST_BIND"); bind != "" {
			config.Extra = func() []hostkey.Endpoint { return []hostkey.Endpoint{{Addr: bind}} }
			config.Wanted = func() bool { return true } // the test never touches the real Run key
			config.Adapters = func() ([]hostkey.Adapter, error) { return nil, nil }
		}
		if public, err := hex.DecodeString(os.Getenv("SCADABENCH_HOSTKEY_TEST_PUBKEY")); err == nil && len(public) == 32 {
			config.Options.Keys = map[string][]byte{licensing.KeyID(public): public}
		}
	}
}
