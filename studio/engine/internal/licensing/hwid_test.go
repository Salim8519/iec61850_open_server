package licensing

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hwidVectors are the machine ID v3 vectors shared with the native guard test
// (studio/tests/unit/test_native_license.py); an independent Python reference made them.
func hwidVectors(t *testing.T) []struct {
	Name, Firmware, Board, GUID, HWID string
} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "tests", "fixtures", "licensing", "hwid_v3.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Cases []struct {
			Name, Firmware, Board, GUID, HWID string
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatal(err)
	}
	return file.Cases
}

func TestMachineIDV3MatchesTheSharedVectors(t *testing.T) {
	cases := hwidVectors(t)
	if len(cases) < 10 {
		t.Fatalf("only %d vectors", len(cases))
	}
	for _, c := range cases {
		parts := MachineParts{FirmwareUUID: c.Firmware, BoardSerial: c.Board, MachineGUID: c.GUID}
		if got := Compose(parts); got != c.HWID {
			t.Errorf("%s: %s, want %s", c.Name, got, c.HWID)
		}
	}
}

func TestOnlyFirmwareDataMakesTheIDSoAWindowsReinstallKeepsIt(t *testing.T) {
	real := MachineParts{FirmwareUUID: "7a1b2c3d4e5f60718293a4b5c6d7e8f9", BoardSerial: "SN-1", MachineGUID: "guid-1"}
	reinstalled := real
	reinstalled.MachineGUID = "guid-2"
	if Compose(real) != Compose(reinstalled) {
		t.Error("a new MachineGuid changed the ID of a PC with firmware identity")
	}
	newBoard := real
	newBoard.BoardSerial = "SN-2"
	if Compose(real) == Compose(newBoard) {
		t.Error("a new motherboard kept the ID")
	}
	newFirmware := real
	newFirmware.FirmwareUUID = "00112233445566778899aabbccddeeff"
	if Compose(real) == Compose(newFirmware) {
		t.Error("a new firmware UUID kept the ID")
	}
	bare := MachineParts{FirmwareUUID: "", BoardSerial: "None", MachineGUID: "guid-1"}
	other := bare
	other.MachineGUID = "guid-2"
	if Compose(bare) == Compose(other) {
		t.Error("PCs with no firmware identity share an ID: the MachineGuid must tell them apart")
	}
	if !strings.HasPrefix(PrefixV3, "scadabench-hwid-v3|") {
		t.Error("the v3 prefix changed")
	}
}

func TestNothingReadableMakesNoID(t *testing.T) {
	if _, err := ComposeChecked(MachineParts{BoardSerial: "Default string", FirmwareUUID: "ffffffffffffffffffffffffffffffff"}); err == nil {
		t.Fatal("a computer that reveals nothing got an ID that every such computer would share")
	}
	if _, err := ComposeChecked(MachineParts{MachineGUID: "x"}); err != nil {
		t.Fatalf("the MachineGuid fallback was refused: %v", err)
	}
}

func TestBoardPlaceholdersAreNeverPartOfTheID(t *testing.T) {
	for _, junk := range []string{"", " ", "Default string", "DEFAULT STRING", "To be filled by O.E.M.", "to be filled by o.e.m.",
		"None", "NONE", "N/A", "n/a", "0", "00000000", "xxxxxxxx", "-", "........", "  0  "} {
		if got := CleanBoardSerial(junk); got != "" {
			t.Errorf("%q counted as a serial number: %q", junk, got)
		}
	}
	for text, want := range map[string]string{" AB-12 ": "AB-12", "0000000001": "0000000001", "Ab": "Ab", "SN 1": "SN 1"} {
		if got := CleanBoardSerial(text); got != want {
			t.Errorf("CleanBoardSerial(%q) = %q, want %q", text, got, want)
		}
	}
}

// structure builds one SMBIOS structure: the formatted part (header included, length
// bytes) and its string set.
func structure(kind byte, length int, set ...string) []byte {
	formatted := make([]byte, length)
	formatted[0], formatted[1] = kind, byte(length)
	return append(formatted, stringSet(set)...)
}

func stringSet(set []string) []byte {
	if len(set) == 0 {
		return []byte{0, 0}
	}
	var out []byte
	for _, text := range set {
		out = append(append(out, text...), 0)
	}
	return append(out, 0)
}

func smbiosBlob(structures ...[]byte) []byte {
	blob := make([]byte, 8)
	for _, one := range structures {
		blob = append(blob, one...)
	}
	return append(blob, structure(127, 4)...)
}

func baseboard(serialIndex byte, set ...string) []byte {
	one := structure(2, 15, set...)
	one[7] = serialIndex
	return one
}

func TestBoardSerialIsReadFromTheBaseboardStructure(t *testing.T) {
	system := structure(1, 27, "Maker", "Product")
	system[7] = 2 // a serial number string on the system structure must not be used
	bios := structure(0, 24, "Vendor", "1.0")
	blob := smbiosBlob(bios, system, baseboard(3, "Board maker", "Board", "SN-4711"))
	if got := SMBIOSBoardSerial(blob); got != "SN-4711" {
		t.Fatalf("board serial = %q", got)
	}
	for name, test := range map[string]struct {
		blob []byte
		want string
	}{
		"serial string number 0 means none":      {smbiosBlob(baseboard(0, "Maker", "Board")), ""},
		"serial number beyond the string set":    {smbiosBlob(baseboard(9, "Maker", "Board")), ""},
		"baseboard without any strings":          {smbiosBlob(baseboard(1)), ""},
		"first of several baseboards":            {smbiosBlob(baseboard(1, "FIRST"), baseboard(1, "SECOND")), "FIRST"},
		"no baseboard at all":                    {smbiosBlob(system, bios), ""},
		"table too short for a header":           {[]byte{1, 2, 3}, ""},
		"truncated structure":                    {append(make([]byte, 8), 2, 15, 0, 0, 1), ""},
		"structure length below the header size": {append(make([]byte, 8), 2, 3, 0, 0), ""},
		"baseboard too short to hold the serial": {smbiosBlob(structure(2, 6, "Maker")), ""},
	} {
		if got := SMBIOSBoardSerial(test.blob); got != test.want {
			t.Errorf("%s: %q, want %q", name, got, test.want)
		}
	}
	if uuid := SMBIOSSystemUUID(blob); len(uuid) != 32 {
		t.Errorf("the system UUID reader no longer finds the type 1 structure: %q", uuid)
	}
}

// TestCertificatesForTheOlderMachineIDsAreRefused: v1 and v2 IDs of the very same computer
// are not accepted (no customers existed when they were removed, ADR-0098).
func TestCertificatesForTheOlderMachineIDsAreRefused(t *testing.T) {
	parts := MachineParts{FirmwareUUID: "7a1b2c3d4e5f60718293a4b5c6d7e8f9", BoardSerial: "SN-1", MachineGUID: "guid-1"}
	env := newBindEnv(t, nil)
	env.hwid = Compose(parts)
	env.service = env.newService()
	v2 := sha256Hex("substation-studio-hwid-v2|guid-1|0000ABCD|" + parts.FirmwareUUID)
	v1 := sha256Hex("substation-studio-hwid-v1|guid-1|0000ABCD")
	for name, hwid := range map[string]string{"v2": v2, "v1": v1} {
		for _, issuedAt := range []int64{1_000_000_000, 1_760_000_000} { // before and after the old cut-off logic
			signed := env.server.sign(map[string]any{"email": "", "expires_at": 0, "hwid": hwid, "issued_at": issuedAt,
				"license_id": bindKey, "max_machines": 1, "type": "lifetime", "version": 1})
			text, _ := json.Marshal(signed)
			_, err := env.service.ActivateOffline(string(text))
			var refused *ActionError
			if !errors.As(err, &refused) || refused.Message != WrongMachine {
				t.Errorf("%s certificate issued at %d: %v", name, issuedAt, err)
			}
		}
	}
	signed := env.server.sign(map[string]any{"email": "", "expires_at": 0, "hwid": env.hwid, "issued_at": 1_760_000_000,
		"license_id": bindKey, "max_machines": 1, "type": "lifetime", "version": 1})
	text, _ := json.Marshal(signed)
	if _, err := env.service.ActivateOffline(string(text)); err != nil {
		t.Fatalf("the v3 certificate of this computer was refused: %v", err)
	}
}
