package push

import (
	"strings"
	"testing"
)

func TestCatalogTicksOnlyWithCondition(t *testing.T) {
	for _, spec := range Specs(CatalogOptions{}) {
		if spec.Topic.Every != 0 {
			t.Errorf("%s re-reads periodically without a Running/Scanning condition", spec.Topic.Name)
		}
	}
	running := func() bool { return true }
	ticking := 0
	for _, spec := range Specs(CatalogOptions{Running: running, Scanning: running}) {
		if spec.Topic.Every > 0 {
			ticking++
			if spec.Topic.While == nil {
				t.Errorf("%s ticks without a condition", spec.Topic.Name)
			}
		}
		if spec.Topic.Gap != 0 && spec.Topic.Gap != DefaultGap && !strings.HasPrefix(spec.Topic.Name, "health") &&
			spec.Topic.Name != "activity" && !strings.HasSuffix(spec.Topic.Name, ".points") {
			t.Errorf("%s: unexpected gap %v", spec.Topic.Name, spec.Topic.Gap)
		}
	}
	if ticking != 6 { // health, goose.links, goose.overview, sv, live.sessions, scan
		t.Fatalf("%d ticking topics", ticking)
	}
}

func TestModeRef(t *testing.T) {
	for ref, want := range map[string]bool{"LD0/LLN0.Mod.stVal": true, "LD0/XCBR1.Beh.stVal": true,
		"LD0/LPHD1.Sim.stVal": true, "": true, "LD0/XCBR1.Pos.stVal": false, "LD0/LLN0.Mod.q": false} {
		if ModeRef("ied", ref) != want {
			t.Errorf("ModeRef(%q) != %v", ref, want)
		}
	}
}
