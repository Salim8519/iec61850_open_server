package main

import (
	"os"
	"strings"
)

// nativeSeal is set by the Go release build (-X main.nativeSeal=name=sha256,name=sha256): the
// SHA-256 of every shipped native program, Python's sealed integrity.EXPECTED (ADR-0083).
// Empty in development, where nothing is checked.
var nativeSeal = ""

// parseSeal reads "name=hex,name=hex" into a map; malformed parts are ignored, so a broken
// seal can only make the integrity check fail, never pass.
func parseSeal(text string) map[string]string {
	seal := map[string]string{}
	for _, part := range strings.Split(text, ",") {
		name, digest, found := strings.Cut(strings.TrimSpace(part), "=")
		if found && name != "" && digest != "" {
			seal[name] = strings.ToLower(digest)
		}
	}
	if len(seal) == 0 {
		return nil
	}
	return seal
}

// announceSignedModule is entrypoint._announce_signed_module for the Go release: the native
// workers verify this exact executable, but only when a signature was shipped beside it
// (ScadaBench.exe.sig). An unsigned build sets nothing, so it never locks itself out.
func announceSignedModule() {
	executable, err := os.Executable()
	if err != nil {
		return
	}
	if info, err := os.Stat(executable + ".sig"); err == nil && !info.IsDir() {
		_ = os.Setenv("STUDIO_MODULE_PATH", executable)
		_ = os.Setenv("STUDIO_MODULE_SIG", executable+".sig")
	}
}
