package main

import (
	"errors"
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

// signatureMissing is entrypoint.SIGNATURE_MISSING: shown (in a message box when the exe was
// double-clicked) when a packaged ScadaBench has no readable signature file.
const signatureMissing = "ScadaBench cannot start because its signature file is missing or unreadable, so this copy " +
	"may have been changed. Reinstall ScadaBench from the official download at scadabench.com."

// announceSignedModule is entrypoint._announce_signed_module for the Go release: the native
// workers verify this exact executable against ScadaBench.exe.sig, which every packaged build
// must ship beside it. A packaged build without a readable signature refuses to start
// (ADR-0097), so deleting the .sig file cannot switch the self-check off.
func announceSignedModule() error {
	executable, err := os.Executable()
	if err != nil {
		return errors.New(signatureMissing)
	}
	return announceModule(executable)
}

// announceModule points the native workers at executable and its signature file.
func announceModule(executable string) error {
	signature := executable + ".sig"
	if data, err := os.ReadFile(signature); err != nil || len(data) == 0 {
		return errors.New(signatureMissing)
	}
	_ = os.Setenv("STUDIO_MODULE_PATH", executable)
	_ = os.Setenv("STUDIO_MODULE_SIG", signature)
	return nil
}
