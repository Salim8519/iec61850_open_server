package store

import (
	"strings"

	"substation-license-server/internal/certs"
)

// TrialParts are the client's salted SHA-256 hashes of the three machine parts
// (certs.PartHash): Windows MachineGuid, system volume serial and firmware UUID.
// Old clients send none and keep the original one-trial-per-machine-ID rule.
type TrialParts struct {
	GUID     string `json:"guid,omitempty"`
	Volume   string `json:"volume,omitempty"`
	Firmware string `json:"firmware,omitempty"`
}

type partEntry struct{ kind, key string }

// placeholderUUIDs are firmware UUIDs that many unrelated PCs and VMs share. Both the
// displayed form and the raw SMBIOS byte order are listed; keep equal to trial_parts.py.
var placeholderUUIDs = []string{
	"03000200-0400-0500-0006-000700080009",
	"00020003-0004-0005-0006-000700080009",
	"12345678-1234-5678-90ab-cddeefaabbcc",
}

var junkHashes = buildJunkHashes()

func buildJunkHashes() map[string]bool {
	junk := map[string]bool{certs.PartHash("firmware", ""): true, certs.PartHash("volume", "00000000"): true,
		certs.PartHash("volume", "FFFFFFFF"): true, certs.PartHash("guid", ""): true}
	const digits = "0123456789abcdef"
	for value := 0; value < 256; value++ {
		pair := string([]byte{digits[value>>4], digits[value&15]})
		junk[certs.PartHash("firmware", strings.Repeat(pair, 16))] = true
	}
	for _, displayed := range placeholderUUIDs {
		plain := strings.ReplaceAll(displayed, "-", "")
		junk[certs.PartHash("firmware", plain)] = true
		junk[certs.PartHash("firmware", SwapUUIDFields(plain))] = true
	}
	return junk
}

// SwapUUIDFields converts between the displayed UUID and the raw SMBIOS byte order
// (the first three fields are little-endian in the firmware table).
func SwapUUIDFields(hex32 string) string {
	if len(hex32) != 32 {
		return hex32
	}
	reverse := func(field string) string {
		out := ""
		for i := len(field) - 2; i >= 0; i -= 2 {
			out += field[i : i+2]
		}
		return out
	}
	return reverse(hex32[0:8]) + reverse(hex32[8:12]) + reverse(hex32[12:16]) + hex32[16:]
}

// IsJunkPart reports a hash that must never be compared (placeholder or empty value).
func IsJunkPart(hash string) bool { return junkHashes[hash] }

// Clean drops malformed hashes and well-known placeholders, so they never match.
func (p TrialParts) Clean() TrialParts {
	keep := func(hash string) string {
		hash = strings.ToLower(strings.TrimSpace(hash))
		if !certs.ValidHWID(hash) || IsJunkPart(hash) {
			return ""
		}
		return hash
	}
	return TrialParts{GUID: keep(p.GUID), Volume: keep(p.Volume), Firmware: keep(p.Firmware)}
}

// Count is the number of usable parts.
func (p TrialParts) Count() int { return len(p.entries()) }

func (p TrialParts) entries() []partEntry {
	entries := []partEntry{}
	for _, part := range []partEntry{{"guid", p.GUID}, {"volume", p.Volume}, {"firmware", p.Firmware}} {
		if part.key != "" {
			entries = append(entries, partEntry{part.kind, part.kind + ":" + part.key})
		}
	}
	return entries
}
