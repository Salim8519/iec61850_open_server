package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/goosemsg"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// GOOSE links (ADR-0112): an equipment may send one message of a simulated IED from its sheet.
// The setting is Go-only and omittable; the leaves are checked against the message's dataset.

// noGooseLink are the kinds that take no GOOSE link, with the reason in words.
var noGooseLink = map[string]string{"JUNCTION": "A junction is a drawing point", "INFEED": "An infeed is a drawing point",
	PointPanelKind:   "A point panel sends GOOSE through its own points",
	IEDComponentKind: "An IED component sends its messages from its own sheet"}

// catalogOf is the GOOSE catalog of a stored IED (a variable only so tests can count builds).
var catalogOf = goosemsg.For

// checkGooseLink checks an equipment's goose_link: a kind that may send one, a project IED, one
// of the messages that IED sends, and leaves of that message's dataset with the member's
// functional constraint and the type the equipment sends (goosemsg.Link.Check).
func checkGooseLink(data *pyjson.Object, kind string, ieds *Lookup) error {
	raw := pyval.Lookup(data, "goose_link", nil)
	if raw == nil {
		return nil
	}
	if words, refused := noGooseLink[kind]; refused {
		return apperr.New(words + ", so it takes no GOOSE link")
	}
	link := goosemsg.ParseLink(raw)
	ied := ieds.IED(link.IEDID)
	if ied == nil {
		return apperr.New("The GOOSE link uses an IED that is not in the project. Choose another IED or import it again")
	}
	return link.Check(kind, catalogOf(ied))
}

// settleGooseLink drops a cleared link, so a node that no longer sends GOOSE keeps its old bytes.
func settleGooseLink(data *pyjson.Object) {
	if value, present := data.Get("goose_link"); present && value == nil {
		data.Delete("goose_link")
	}
}

// gooseUsesIED says whether a node is the component of, or sends GOOSE for, one of the IEDs.
func gooseUsesIED(data *pyjson.Object, ids map[string]bool) bool {
	if kindOf(data) == IEDComponentKind && ids[text(data, "ied_id")] {
		return true
	}
	return ids[text(object(pyval.Lookup(data, "goose_link", nil)), "ied_id")]
}

// dropGooseLinkOf clears the GOOSE link of a removed IED; the equipment stays.
func dropGooseLinkOf(data *pyjson.Object, ids map[string]bool) {
	if data != nil && ids[text(object(pyval.Lookup(data, "goose_link", nil)), "ied_id")] {
		data.Delete("goose_link")
	}
}
