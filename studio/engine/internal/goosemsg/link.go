package goosemsg

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Link is an equipment's goose_link (ADR-0112): which IED message it sends and which values.
// Mode "one" sends `One`; mode "contacts" (breaker, disconnector, earth switch only) sends the
// two on/off values `Open` and `Closed` together.
type Link struct {
	IEDID, GocbRef, Mode, One, Open, Closed string
}

// ParseLink reads a validated goose_link object (nil when the node has none).
func ParseLink(value any) *Link {
	setting := object(value)
	if setting == nil {
		return nil
	}
	return &Link{IEDID: text(setting, "ied_id"), GocbRef: text(setting, "gocb_ref"), Mode: text(setting, "mode"),
		One: text(setting, "one"), Open: text(setting, "open"), Closed: text(setting, "closed")}
}

// IsSwitch says whether the kind has an open/closed position.
func IsSwitch(kind string) bool { return kind == "CB" || kind == "DS" || kind == "ES" }

// Refs are the leaves the link sends.
func (l *Link) Refs() []string {
	if l.Mode == "contacts" {
		return []string{l.Open, l.Closed}
	}
	return []string{l.One}
}

// Check returns a plain-words refusal for a link that cannot be sent as saved, or nil.
// kind is the equipment kind; iedName names the IED in the words.
func (l *Link) Check(kind string, catalog *Catalog) error {
	block := catalog.Block(l.GocbRef)
	if block == nil {
		return apperr.New(catalog.Name + " does not send the GOOSE message " + l.GocbRef + ". Choose one of the " +
			"messages it sends, or import its file again")
	}
	if l.Mode == "contacts" {
		if !IsSwitch(kind) {
			return apperr.New("Only a breaker, disconnector or earth switch sends open and closed as two values. " +
				"Choose one value instead")
		}
		if l.One != "" || l.Open == "" || l.Closed == "" {
			return apperr.New("Choose the open value and the closed value this switch sends")
		}
		if l.Open == l.Closed {
			return apperr.New("Open and closed must be two different values of the message")
		}
		for _, ref := range []string{l.Open, l.Closed} {
			if err := checkLeaf(block, ref, OnOff, "an on/off (BOOLEAN) value"); err != nil {
				return err
			}
		}
		return nil
	}
	if l.One == "" || l.Open != "" || l.Closed != "" {
		return apperr.New("Choose the one value this equipment sends")
	}
	leaf, err := memberLeaf(block, l.One)
	if err != nil {
		return err
	}
	kindOf := ValueType(leaf.Kind)
	if IsSwitch(kind) && kindOf == Number {
		return apperr.New(shortRef(leaf.Ref) + " is a number, but a switch sends a position (Dbpos) or one " +
			"on/off value (BOOLEAN). Choose another value of the message")
	}
	return checkLeaf(block, l.One, "", "")
}

func memberLeaf(block *Block, ref string) (Leaf, error) {
	leaf, found := block.Leaf(ref)
	if !found {
		return Leaf{}, apperr.New(shortRef(ref) + " is not part of the GOOSE message " + block.Name +
			". Choose a value from that message")
	}
	return leaf, nil
}

// checkLeaf: a member of the block, of the wanted value type (any sendable one when ""), and
// changeable in the simulated IED.
func checkLeaf(block *Block, ref, want, words string) error {
	leaf, err := memberLeaf(block, ref)
	if err != nil {
		return err
	}
	if want != "" && ValueType(leaf.Kind) != want {
		return apperr.New(shortRef(ref) + " is " + leaf.Kind + "; it must be " + words + ". Choose another value")
	}
	if ValueType(leaf.Kind) == "" {
		return apperr.New(shortRef(ref) + " (" + leaf.Kind + ") cannot be sent from the drawing. Choose an on/off " +
			"value, a position or a number of the message")
	}
	if !leaf.Sendable() {
		return apperr.New(shortRef(ref) + " is read-only in the simulated IED, so it cannot be changed. " +
			"Choose another value")
	}
	return nil
}

// Plan returns the one batch an action sends: "open"/"closed" (a position 1/2, a switch's one
// on/off value false/true, or both contacts at once), "on"/"off" (an on/off value) or "value"
// (a number). The link must have passed Check against the same block.
func (l *Link) Plan(kind string, block *Block, action string, value any) ([]Write, error) {
	if l.Mode == "contacts" {
		if action != "open" && action != "closed" {
			return nil, apperr.New("This switch sends Open or Closed.")
		}
		opened := action == "open"
		return []Write{{Ref: l.Open, Value: opened}, {Ref: l.Closed, Value: !opened}}, nil
	}
	leaf, err := memberLeaf(block, l.One)
	if err != nil {
		return nil, err
	}
	switch ValueType(leaf.Kind) {
	case Position:
		if action == "open" {
			return []Write{{Ref: leaf.Ref, Value: int64(1)}}, nil
		}
		if action == "closed" {
			return []Write{{Ref: leaf.Ref, Value: int64(2)}}, nil
		}
		return nil, apperr.New(shortRef(leaf.Ref) + " is a position: send Open or Closed.")
	case OnOff:
		switch {
		case IsSwitch(kind) && (action == "open" || action == "closed"):
			return []Write{{Ref: leaf.Ref, Value: action == "closed"}}, nil
		case !IsSwitch(kind) && (action == "on" || action == "off"):
			return []Write{{Ref: leaf.Ref, Value: action == "on"}}, nil
		}
		if IsSwitch(kind) {
			return nil, apperr.New("This switch sends Open or Closed.")
		}
		return nil, apperr.New(shortRef(leaf.Ref) + " is an on/off value: send On or Off.")
	}
	if action != "value" {
		return nil, apperr.New(shortRef(leaf.Ref) + " is a number: type a number and press Send.")
	}
	wire, err := wireValue(leaf, value)
	if err != nil {
		return nil, err
	}
	return []Write{{Ref: leaf.Ref, Value: wire}}, nil
}

// PlanValue is one value sent from the IED component's own message list.
func PlanValue(block *Block, ref string, value any) ([]Write, error) {
	if err := checkLeaf(block, ref, "", ""); err != nil {
		return nil, err
	}
	leaf, _ := block.Leaf(ref)
	wire, err := wireValue(leaf, value)
	if err != nil {
		return nil, err
	}
	return []Write{{Ref: ref, Value: wire}}, nil
}

// Object is the link as stored, for tests and answers.
func (l *Link) Object() *pyjson.Object {
	none := func(text string) any {
		if text == "" {
			return nil
		}
		return text
	}
	return pyjson.NewObject().Set("ied_id", l.IEDID).Set("gocb_ref", l.GocbRef).Set("mode", l.Mode).
		Set("one", none(l.One)).Set("open", none(l.Open)).Set("closed", none(l.Closed))
}
