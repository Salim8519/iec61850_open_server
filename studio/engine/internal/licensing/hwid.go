package licensing

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"studio-engine/internal/licensing/pytext"
)

// Machine ID v3 (ADR-0098), shared with the native guard (license_guard.cpp):
//
//	hwid = SHA256_HEX("scadabench-hwid-v3|" + firmware + "|" + board + "|" + extra)
//
// firmware is the SMBIOS type 1 system UUID as lowercase raw hex and board the SMBIOS type 2
// baseboard serial number, each "" when it is a placeholder every cheap PC shares. extra is
// "" unless BOTH are "", then it is the lowercased MachineGuid (such a PC gets a new ID when
// Windows is reinstalled). Only firmware data makes the ID, so a Windows reinstall keeps it
// and a new motherboard changes it. The frozen Python backend still uses the older v2 ID and
// is not a shipped engine; v1 and v2 certificates are not accepted by the Go engine.
const PrefixV3 = "scadabench-hwid-v3|"

// MachineParts are the raw identifiers the machine ID is built from.
type MachineParts struct {
	FirmwareUUID string // SMBIOS type 1 UUID, raw bytes as lowercase hex
	BoardSerial  string // SMBIOS type 2 serial number
	MachineGUID  string // HKLM\SOFTWARE\Microsoft\Cryptography\MachineGuid
}

func normalizedGUID(guid string) string { return strings.ToLower(pytext.Strip(guid)) }

func sha256Hex(text string) string {
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:])
}

// asciiSpace is what "trimmed" means for SMBIOS strings (the native guard trims the same).
const asciiSpace = " \t\r\n\v\f"

func asciiLower(text string) string {
	out := []byte(text)
	for index, c := range out {
		if c >= 'A' && c <= 'Z' {
			out[index] = c + 32
		}
	}
	return string(out)
}

// placeholderBoards are serial numbers firmware makers leave in unprogrammed boards.
var placeholderBoards = map[string]bool{"": true, "default string": true, "to be filled by o.e.m.": true,
	"none": true, "n/a": true, "0": true}

// CleanBoardSerial is the board part of the machine ID: the trimmed serial number, or ""
// for a placeholder (empty, "Default string", "To be filled by O.E.M.", "None", "N/A", "0",
// any text made of one repeated character), compared without regard to case.
func CleanBoardSerial(raw string) string {
	text := strings.Trim(raw, asciiSpace)
	if placeholderBoards[asciiLower(text)] {
		return ""
	}
	for index := 1; index < len(text); index++ {
		if text[index] != text[0] {
			return text
		}
	}
	return "" // one character, or one character repeated
}

// CleanFirmwareUUID is the firmware part of the machine ID: the lowercase raw UUID hex, or ""
// when it is empty, malformed, one repeated byte or a known placeholder (trial_parts.go).
func CleanFirmwareUUID(raw string) string {
	if IsJunkFirmware(raw) {
		return ""
	}
	return strings.ToLower(pytext.Strip(raw))
}

// Compose is the v3 machine ID of the raw identifiers (junk is dropped here).
func Compose(parts MachineParts) string {
	firmware, board := CleanFirmwareUUID(parts.FirmwareUUID), CleanBoardSerial(parts.BoardSerial)
	extra := ""
	if firmware == "" && board == "" {
		extra = normalizedGUID(parts.MachineGUID)
	}
	return sha256Hex(PrefixV3 + firmware + "|" + board + "|" + extra)
}

// errNoIdentity stands for a computer whose firmware reveals nothing and whose MachineGuid
// cannot be read: every such PC would share one ID, so none is made.
var errNoIdentity = errors.New("Cannot read this computer's hardware identity")

// ComposeChecked is Compose, refusing the case where nothing identifies this computer.
func ComposeChecked(parts MachineParts) (string, error) {
	if CleanFirmwareUUID(parts.FirmwareUUID) == "" && CleanBoardSerial(parts.BoardSerial) == "" &&
		normalizedGUID(parts.MachineGUID) == "" {
		return "", errNoIdentity
	}
	return Compose(parts), nil
}

// smbiosData is the table of a RawSMBIOSData blob (8-byte header skipped), or nil.
func smbiosData(raw []byte) []byte {
	if len(raw) < 8 {
		return nil
	}
	return raw[8:]
}

// SMBIOSSystemUUID is hwid._smbios_system_uuid: lowercase hex of the 16 raw UUID bytes of
// the SMBIOS type 1 table in a RawSMBIOSData blob (8-byte header), or "". Bytes are used
// verbatim so Python, C++ and Go agree.
func SMBIOSSystemUUID(raw []byte) string {
	data := smbiosData(raw)
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

// SMBIOSBoardSerial is the serial number string of the first SMBIOS type 2 (baseboard)
// structure in a RawSMBIOSData blob, or "". The serial's string number is the byte at offset
// 7 of the structure; its text is that string of the string set after the formatted part
// (strings end with a NUL, the set with a second NUL). Native twin: license_guard.cpp.
func SMBIOSBoardSerial(raw []byte) string {
	data := smbiosData(raw)
	for pos := 0; pos+4 <= len(data); {
		kind, length := data[pos], int(data[pos+1])
		if length < 4 || pos+length > len(data) {
			return ""
		}
		start, end := pos+length, pos+length
		for end+1 < len(data) && !(data[end] == 0 && data[end+1] == 0) {
			end++
		}
		if kind == 2 && length >= 8 {
			return smbiosString(data[start:min(end, len(data))], int(data[pos+7]))
		}
		if kind == 127 {
			return ""
		}
		pos = end + 2
	}
	return ""
}

// smbiosString is string number index (1-based) of a string set without its last NUL.
func smbiosString(set []byte, index int) string {
	if index < 1 || len(set) == 0 {
		return ""
	}
	texts := strings.Split(string(set), "\x00")
	if index > len(texts) {
		return ""
	}
	return texts[index-1]
}

// MachineID returns this computer's v3 machine ID (Windows only).
func MachineID() (string, error) {
	parts, err := readMachineParts()
	if err != nil {
		return "", err
	}
	return ComposeChecked(parts)
}
