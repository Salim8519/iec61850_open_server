package labnet

import (
	"strings"
	"testing"
)

// Only the DNS step is inside try/catch; every other step keeps stopping the script.
func TestCardScriptOnlyTheDnsStepMayFail(t *testing.T) {
	script := cardScript("{guid}")
	if strings.Count(script, "try {") != 1 || strings.Count(script, "catch {") != 1 {
		t.Fatalf("want exactly one try/catch:\n%s", script)
	}
	guarded := script[strings.Index(script, "try {"):strings.Index(script, "catch {")]
	if !strings.Contains(guarded, "Set-DnsClient") {
		t.Fatalf("the try block is not the DNS step: %s", guarded)
	}
	for _, step := range []string{"$ErrorActionPreference='Stop'", "throw 'card not found'", "Rename-NetAdapter",
		"Set-NetIPInterface", "Disable-NetAdapterBinding"} {
		if !strings.Contains(script, step) || strings.Contains(guarded, step) {
			t.Fatalf("step %q missing or guarded", step)
		}
	}
	if strings.Index(script, "Disable-NetAdapterBinding") < strings.Index(script, "catch {") {
		t.Fatal("the IPv6 step must run after the DNS step")
	}
}

func TestDnsNote(t *testing.T) {
	if note := dnsNote("\r\nsomething else\r\n"); note != "" {
		t.Fatalf("note without a skipped step: %q", note)
	}
	note := dnsNote("dns-skipped: Invalid class \r\n")
	if note != "Windows did not turn off DNS registration on the card (Invalid class); the card works without it." {
		t.Fatalf("note %q", note)
	}
}
