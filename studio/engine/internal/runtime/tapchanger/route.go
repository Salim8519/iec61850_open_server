package tapchanger

import (
	"errors"
	"strconv"
)

// Binding is one IED point of a tap changer.
type Binding struct{ IedID, Ref string }

// Set says whether the binding names a point.
func (b Binding) Set() bool { return b.IedID != "" && b.Ref != "" }

// Commands is how one transformer is commanded: Mode "step" with one BSC or ISC control
// (StepCDC), "spc_pair" with two SPC controls, or "none".
type Commands struct {
	Mode               string
	Step, Raise, Lower Binding
	StepCDC            string
}

// Command is one routed raise or lower: Typed sends Value to Target through the typed control
// path (ADR-0104: "higher"/"lower" for BSC, the new tap for ISC); otherwise Target is an SPC
// operated with true through the existing command path.
type Command struct {
	Typed  bool
	Target Binding
	Value  any
	Next   int64 // the tap expected after the step (only meaningful when the tap is known)
}

// Route chooses the command for one raise or lower. The limits are checked when the tap is
// known; an ISC needs the tap, because it is sent the new tap number itself.
func Route(s Settings, commands Commands, action string, tap int64, known bool) (Command, error) {
	if action != Raise && action != Lower {
		return Command{}, errors.New("Unknown tap action. Use raise or lower.")
	}
	next, allowed := s.Next(tap, action)
	if known && !allowed {
		if action == Raise {
			return Command{}, errors.New("The tap is already at the highest tap (" + strconv.FormatInt(s.MaxTap, 10) +
				"), so it cannot be raised further.")
		}
		return Command{}, errors.New("The tap is already at the lowest tap (" + strconv.FormatInt(s.MinTap, 10) +
			"), so it cannot be lowered further.")
	}
	ready := (commands.Mode == "step" && commands.Step.Set()) ||
		(commands.Mode == "spc_pair" && commands.Raise.Set() && commands.Lower.Set())
	if commands.Mode != "none" && !ready {
		return Command{}, errors.New("The raise/lower command is not complete yet. Choose its controls in the " +
			"properties while the IEDs are stopped.")
	}
	switch commands.Mode {
	case "step":
		switch commands.StepCDC {
		case "BSC":
			value := "higher"
			if action == Lower {
				value = "lower"
			}
			return Command{Typed: true, Target: commands.Step, Value: value, Next: next}, nil
		case "ISC":
			if !known {
				return Command{}, errors.New("An ISC step command sends the new tap number, but the tap position " +
					"is not known yet. Wait for the tap position, then try again.")
			}
			return Command{Typed: true, Target: commands.Step, Value: next, Next: next}, nil
		}
		return Command{}, errors.New("The step command needs a BSC or ISC control. Choose it again in the properties.")
	case "spc_pair":
		target := commands.Raise
		if action == Lower {
			target = commands.Lower
		}
		return Command{Target: target, Value: true, Next: next}, nil
	}
	return Command{}, errors.New("This transformer has no raise/lower command yet. Choose one in its properties " +
		"while the IEDs are stopped.")
}
