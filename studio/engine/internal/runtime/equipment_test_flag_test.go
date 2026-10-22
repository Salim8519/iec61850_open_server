package runtime

import (
	"strings"
	"testing"

	"studio-engine/internal/runtime/pyval"
)

// Change 0176 (B17): a command with Test=TRUE never moves the simulated equipment, whether it is
// sent at once or after a held selection; a normal command still does.
func TestTestFlagCommandDoesNotMoveTheEquipment(t *testing.T) {
	c, bus, _, _ := sboRun(t, 24200, true)
	normal, err := c.Operate("cb1", nil, "close")
	if err != nil {
		t.Fatal(err)
	}
	if len(pyval.List(normal, "feedback")) == 0 {
		t.Fatal("the fixture's breaker needs plant feedback for this test")
	}
	node, _ := c.findNode("cb1")
	if got := pyval.Str(pyval.Map(node, "data"), "initial_position"); got != "closed" {
		t.Fatalf("a normal close is remembered as closed, got %q", got)
	}
	result, err := c.OperateWith("cb1", nil, "open", pyval.Obj("test", true))
	if err != nil {
		t.Fatal(err)
	}
	if pyval.Get(result, "message") != TestCommandMessage || len(pyval.List(result, "feedback")) != 0 {
		t.Fatalf("a test command must not write plant feedback: %v", pyval.Repr(result))
	}
	node, _ = c.findNode("cb1")
	data := pyval.Map(node, "data")
	if got := pyval.Str(data, "initial_position"); got != "closed" {
		t.Fatalf("a test open must not change the start position, got %q", got)
	}
	if pyval.Get(pyval.Map(data, "command_options"), "test") != true {
		t.Fatal("the sent options are still remembered")
	}
	if got := eventKinds(bus, "equipment_test"); len(got) != 1 {
		t.Fatalf("one equipment_test_command event, got %v", got)
	}
}

func TestTestFlagSelectionDoesNotMoveTheEquipment(t *testing.T) {
	c, _, _, _ := sboRun(t, 24210, false)
	if _, err := c.EquipmentSelect("cb1", "close", pyval.Obj("test", true)); err != nil {
		t.Fatal(err)
	}
	result, err := c.Operate("cb1", nil, "close") // Operate after Select carries the held options
	if err != nil || pyval.Get(result, "message") != TestCommandMessage {
		t.Fatalf("operate of a test selection: %v %v", result, err)
	}
	node, _ := c.findNode("cb1")
	if got := pyval.Str(pyval.Map(node, "data"), "initial_position"); got == "closed" {
		t.Fatal("a test close after Select must not be remembered as closed")
	}
}

// Change 0176 (B1): a Cancel the device refuses keeps the selection, so Operate still works,
// and a model without Cancel is explained in plain words.
func TestRefusedCancelKeepsTheSelection(t *testing.T) {
	t.Setenv(fakeNoCancelEnv, "1")
	c, _, _, _ := sboRun(t, 24220, false)
	if _, err := c.EquipmentSelect("cb1", "close", nil); err != nil {
		t.Fatal(err)
	}
	_, err := c.EquipmentCancel("cb1")
	if err == nil || !strings.Contains(err.Error(), "model has no Cancel for") {
		t.Fatalf("a refused Cancel must say the model has no Cancel, got %v", err)
	}
	if _, status := c.selections.peek(selectionKey{node: "cb1"}); status != selectionLive {
		t.Fatal("the device still holds the selection, so the engine keeps it")
	}
	if result, err := c.Operate("cb1", nil, "close"); err != nil || pyval.Get(result, "action") != "close" {
		t.Fatalf("Operate within the window still works after a refused Cancel: %v %v", result, err)
	}
}
