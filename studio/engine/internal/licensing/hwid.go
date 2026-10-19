package licensing

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"studio-engine/internal/licensing/pytext"
)

// Machine ID rules of licensing/hwid.py, shared with native license_guard.cpp:
// HWID v2 = SHA256_HEX(PrefixV2 + MachineGuid + "|" + %08X volume serial + "|" + SMBIOS
// system UUID); v1 (no UUID) is accepted only for certificates issued before
// LegacyIssuedBefore (ADR-0051).
const (
	PrefixV2           = "substation-studio-hwid-v2|"
	PrefixLegacy       = "substation-studio-hwid-v1|"
	LegacyIssuedBefore = 1814400000 // 2027-07-01T00:00:00Z; keep equal to license_guard.cpp
)

// MachineParts are the three raw identifiers the machine ID is built from.
type MachineParts struct {
	MachineGUID  string
	VolumeSerial uint32
	FirmwareUUID string
}

func normalizedGUID(guid string) string { return strings.ToLower(pytext.Strip(guid)) }

func sha256Hex(text string) string {
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:])
}

// Compose is hwid.compose: the v2 machine ID.
func Compose(parts MachineParts) string {
	return sha256Hex(fmt.Sprintf("%s%s|%08X|%s", PrefixV2, normalizedGUID(parts.MachineGUID),
		parts.VolumeSerial, parts.FirmwareUUID))
}

// ComposeLegacy is hwid.compose_legacy: the v1 machine ID of builds before HWID v2.
func ComposeLegacy(parts MachineParts) string {
	return sha256Hex(fmt.Sprintf("%s%s|%08X", PrefixLegacy, normalizedGUID(parts.MachineGUID),
		parts.VolumeSerial))
}

// BoundHWID is hwid.bound_hwid: the machine ID a certificate is checked against, the
// legacy one only for an old certificate bound to it. certificateHWID and issuedAt are
// the raw payload values; legacy "" means none. An error stands for the
// TypeError/ValueError of int(issued_at), after which Python uses the current ID.
func BoundHWID(certificateHWID, issuedAt any, current, legacy string) (string, error) {
	text, isText := certificateHWID.(string)
	if legacy == "" || !isText || text != legacy || text == current {
		return current, nil
	}
	issued, err := pyInt(issuedAt)
	if err != nil {
		return current, err
	}
	if issued < LegacyIssuedBefore {
		return legacy, nil
	}
	return current, nil
}

// SMBIOSSystemUUID is hwid._smbios_system_uuid: lowercase hex of the 16 raw UUID bytes of
// the SMBIOS type 1 table in a RawSMBIOSData blob (8-byte header), or "". Bytes are used
// verbatim so Python, C++ and Go agree.
func SMBIOSSystemUUID(raw []byte) string {
	if len(raw) < 8 {
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

// MachineID returns this computer's v2 machine ID (Windows only).
func MachineID() (string, error) {
	parts, err := readMachineParts(true)
	if err != nil {
		return "", err
	}
	return Compose(parts), nil
}

// LegacyMachineID returns this computer's v1 machine ID (Windows only).
func LegacyMachineID() (string, error) {
	parts, err := readMachineParts(false)
	if err != nil {
		return "", err
	}
	return ComposeLegacy(parts), nil
}
