package tapchanger

import (
	"strings"
	"testing"
)

var (
	step  = Binding{"ied1", "IED1LD0/YLTC1.TapChg"}
	raise = Binding{"ied1", "IED1LD0/GGIO1.SPCSO1"}
	lower = Binding{"ied1", "IED1LD0/GGIO1.SPCSO2"}
)

func TestRouteBSCSendsHigherOrLower(t *testing.T) {
	commands := Commands{Mode: "step", Step: step, StepCDC: "BSC"}
	got, err := Route(settings(), commands, Raise, 9, true)
	if err != nil || !got.Typed || got.Value != "higher" || got.Target != step || got.Next != 10 {
		t.Fatalf("raise: %+v %v", got, err)
	}
	got, err = Route(settings(), commands, Lower, 9, true)
	if err != nil || got.Value != "lower" || got.Next != 8 {
		t.Fatalf("lower: %+v %v", got, err)
	}
	if got, err = Route(settings(), commands, Raise, 0, false); err != nil || got.Value != "higher" {
		t.Fatalf("a BSC works without a known tap: %+v %v", got, err)
	}
}

func TestRouteISCSendsTheNewTap(t *testing.T) {
	commands := Commands{Mode: "step", Step: step, StepCDC: "ISC"}
	got, err := Route(settings(), commands, Raise, 9, true)
	if err != nil || !got.Typed || got.Value != int64(10) {
		t.Fatalf("raise: %+v %v", got, err)
	}
	if _, err = Route(settings(), commands, Raise, 0, false); err == nil || !strings.Contains(err.Error(), "not known") {
		t.Fatalf("an ISC without a known tap must be refused: %v", err)
	}
}

func TestRouteSPCPairPicksTheCommand(t *testing.T) {
	commands := Commands{Mode: "spc_pair", Raise: raise, Lower: lower}
	got, err := Route(settings(), commands, Raise, 9, true)
	if err != nil || got.Typed || got.Target != raise || got.Value != true || got.Next != 10 {
		t.Fatalf("raise: %+v %v", got, err)
	}
	got, err = Route(settings(), commands, Lower, 9, true)
	if err != nil || got.Target != lower || got.Next != 8 {
		t.Fatalf("lower: %+v %v", got, err)
	}
}

func TestRouteRefusesLimitsAndMissingCommands(t *testing.T) {
	pair := Commands{Mode: "spc_pair", Raise: raise, Lower: lower}
	if _, err := Route(settings(), pair, Raise, 17, true); err == nil || !strings.Contains(err.Error(), "highest tap (17)") {
		t.Fatalf("raise at max: %v", err)
	}
	if _, err := Route(settings(), pair, Lower, 1, true); err == nil || !strings.Contains(err.Error(), "lowest tap (1)") {
		t.Fatalf("lower at min: %v", err)
	}
	if _, err := Route(settings(), Commands{Mode: "none"}, Raise, 9, true); err == nil {
		t.Fatal("no command must be refused")
	}
	if _, err := Route(settings(), Commands{Mode: "step", Step: step, StepCDC: "SPC"}, Raise, 9, true); err == nil {
		t.Fatal("a step command on an SPC must be refused")
	}
	half := Commands{Mode: "spc_pair", Raise: raise}
	if _, err := Route(settings(), half, Lower, 9, true); err == nil || !strings.Contains(err.Error(), "not complete") {
		t.Fatalf("a pair without its lower command: %v", err)
	}
	if _, err := Route(settings(), pair, "up", 9, true); err == nil {
		t.Fatal("an unknown action must be refused")
	}
}
