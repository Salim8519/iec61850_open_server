package licensing

import (
	"fmt"
	"strings"

	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// Salted hashes of the three machine parts, sent when a free trial is requested
// (licensing/trial_parts.py). The server refuses a new trial only when at least two of
// the three parts match an earlier trial (ADR-0046 addendum 2026-10-19). Raw identifiers
// never leave the PC: part = SHA256_HEX(TrialPartPrefix + kind + "|" + value), with the
// machine ID's normalisation. Placeholder values that many unrelated PCs share are sent
// as "" and never compared; keep the list equal to the server's parts.go.
const TrialPartPrefix = "scadabench-trial-part-v1|"

// placeholderUUIDs are the displayed forms; the raw SMBIOS byte order is derived.
var placeholderUUIDs = []string{
	"03000200-0400-0500-0006-000700080009",
	"00020003-0004-0005-0006-000700080009",
	"12345678-1234-5678-90ab-cddeefaabbcc",
}

// RawTrialParts are the raw identifiers; a nil pointer is a part that could not be read.
type RawTrialParts struct {
	MachineGUID  *string
	VolumeSerial *uint32
	FirmwareUUID string
}

// PartHash is trial_parts.part_hash.
func PartHash(kind, value string) string { return sha256Hex(TrialPartPrefix + kind + "|" + value) }

// SwapUUIDFields converts between displayed UUID order and raw SMBIOS bytes (the first
// three fields little-endian); any length other than 32 is returned unchanged.
func SwapUUIDFields(hex32 string) string {
	if len(hex32) != 32 {
		return hex32
	}
	flip := func(field string) string {
		out := make([]byte, 0, len(field))
		for index := len(field) - 2; index >= 0; index -= 2 {
			out = append(out, field[index:index+2]...)
		}
		return string(out)
	}
	return flip(hex32[0:8]) + flip(hex32[8:12]) + flip(hex32[12:16]) + hex32[16:]
}

func isPlaceholderUUID(value string) bool {
	for _, uuid := range placeholderUUIDs {
		plain := strings.ReplaceAll(uuid, "-", "")
		if value == plain || value == SwapUUIDFields(plain) {
			return true
		}
	}
	return false
}

// IsJunkFirmware is trial_parts.is_junk_firmware: true for empty, malformed,
// one-repeated-byte (all 0 / all F) or placeholder UUIDs.
func IsJunkFirmware(raw string) bool {
	value := strings.ToLower(pytext.Strip(raw))
	if len([]rune(value)) != 32 || strings.Trim(value, "0123456789abcdef") != "" {
		return true
	}
	repeated := true
	for index := 2; index < 32; index += 2 {
		repeated = repeated && value[index:index+2] == value[0:2]
	}
	return repeated || isPlaceholderUUID(value)
}

// ComposeTrialParts is trial_parts.compose_parts: {"guid", "volume", "firmware"} hashes,
// "" for a part that is unreadable or a known placeholder.
func ComposeTrialParts(raw RawTrialParts) *pyjson.Object {
	guid, volume, firmware := "", "", ""
	if raw.MachineGUID != nil {
		if text := normalizedGUID(*raw.MachineGUID); text != "" {
			guid = PartHash("guid", text)
		}
	}
	if raw.VolumeSerial != nil && *raw.VolumeSerial != 0 && *raw.VolumeSerial != 0xFFFFFFFF {
		volume = PartHash("volume", fmt.Sprintf("%08X", *raw.VolumeSerial))
	}
	if !IsJunkFirmware(raw.FirmwareUUID) {
		firmware = PartHash("firmware", strings.ToLower(pytext.Strip(raw.FirmwareUUID)))
	}
	return pyjson.NewObject().Set("guid", guid).Set("volume", volume).Set("firmware", firmware)
}

// MachineTrialParts is trial_parts.machine_parts: this PC's part hashes; every unreadable
// part is simply left empty.
func MachineTrialParts() (*pyjson.Object, error) {
	return ComposeTrialParts(readRawTrialParts()), nil
}
