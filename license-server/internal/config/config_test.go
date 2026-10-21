package config

import (
	"os"
	"strings"
	"testing"
)

func TestOldConfigWithMoveAllowanceStillLoadsAndIsDroppedOnSave(t *testing.T) {
	dir := t.TempDir()
	old := `{"license_price": "149.00", "max_transfers_per_year": 2, "trial_days": 7}`
	if err := os.WriteFile(Path(dir), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFile(dir)
	if err != nil || cfg.LicensePrice != "149.00" || cfg.TrialDays != 7 {
		t.Fatalf("old config unreadable: %+v %v", cfg, err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	saved, _ := os.ReadFile(Path(dir))
	if strings.Contains(string(saved), "transfers") {
		t.Fatalf("the retired setting is still written: %s", saved)
	}
}
