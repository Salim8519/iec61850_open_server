package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Expected values were produced by studio_app.licensing.hwid (the Python reference).
func TestHelperMachineIDMatchesStudioFormula(t *testing.T) {
	cases := []struct {
		guid   string
		serial uint32
		uuid   string
		want   string
	}{
		{" {ABCDEF12-3456-7890-ABCD-EF1234567890} \n", 0x1A2B3C4D, "00112233445566778899aabbccddeeff",
			"38577fe0c64a7bc39e83be68b5646eddb2af6e4bda5b9092d6a4eb02f5f5fb08"},
		{"guid", 7, "", "d12b4c041d3ae0ba9d93d108f967dee3c10dd51a1c4584bbd63f3dcad0114135"},
	}
	for _, c := range cases {
		if got := composeHWID(c.guid, c.serial, c.uuid); got != c.want {
			t.Errorf("composeHWID(%q) = %s, want %s", c.guid, got, c.want)
		}
	}
	raw := append(make([]byte, 8), 0, 4, 0, 0, 0, 0, 1, 27, 1, 0, 0, 0, 0, 0)
	for i := 0; i < 16; i++ {
		raw = append(raw, byte(i))
	}
	raw = append(raw, 0, 0, 0, 0, 0)
	if got := smbiosSystemUUID(raw); got != "000102030405060708090a0b0c0d0e0f" {
		t.Errorf("smbiosSystemUUID = %q", got)
	}
}

// On a PC with the Studio source and venv, the live machine ID must equal Python's.
func TestHelperMachineIDEqualsPythonOnThisPC(t *testing.T) {
	root, _ := filepath.Abs("../..")
	python := filepath.Join(root, "studio", ".venv", "Scripts", "python.exe")
	out, err := exec.Command(python, "-c", "import sys; sys.path.insert(0, r'"+
		filepath.Join(root, "studio", "backend")+"'); from studio_app.licensing import hwid; print(hwid.machine_id())").Output()
	if err != nil {
		t.Skip("Studio venv not available:", err)
	}
	mine, err := machineID()
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.TrimSpace(string(out)); mine != want {
		t.Fatalf("helper machine ID %s differs from Studio %s", mine, want)
	}
}
