package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// hwidPrefix and the formula below must stay byte-identical to
// studio/backend/studio_app/licensing/hwid.py and native compose_hwid().
const hwidPrefix = "substation-studio-hwid-v2|"

func composeHWID(machineGUID string, volumeSerial uint32, firmwareUUID string) string {
	text := hwidPrefix + strings.ToLower(strings.TrimSpace(machineGUID)) + "|" +
		fmt.Sprintf("%08X", volumeSerial) + "|" + firmwareUUID
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// smbiosSystemUUID returns the lowercase hex of the 16 raw SMBIOS Type 1 UUID bytes
// from a RawSMBIOSData buffer, or "" when absent. Mirrors hwid._smbios_system_uuid.
func smbiosSystemUUID(raw []byte) string {
	if len(raw) <= 8 {
		return ""
	}
	data := raw[8:]
	pos := 0
	for pos+4 <= len(data) {
		kind, length := data[pos], int(data[pos+1])
		if length < 4 {
			break
		}
		if kind == 1 && pos+24 <= len(data) {
			return hex.EncodeToString(data[pos+8 : pos+24])
		}
		pos += length
		for pos+1 < len(data) && !(data[pos] == 0 && data[pos+1] == 0) {
			pos++
		}
		pos += 2
	}
	return ""
}
