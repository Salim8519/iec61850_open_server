package goosedom

import (
	"testing"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// Change 0185 item 18: a declared link whose publisher is in the file but not in the project
// is never heard while running; it counts as remote (outside the project), not "not heard".
func TestUnheardPublisherOutsideTheProjectIsRemote(t *testing.T) {
	link := func(publisher string) *pyjson.Object {
		return pyv.O("status", "declared", "gocb_ref", publisher+"LD0/LLN0$GO$g", "publisher", publisher, "subscriber", "SUB")
	}
	linkMap := pyv.O("ieds", []any{
		pyv.O("name", "SUB", "in_project", true), pyv.O("name", "IN", "in_project", true),
		pyv.O("name", "OUT", "in_project", false)},
		"links", []any{link("IN"), link("OUT")})
	watches := []*pyjson.Object{
		pyv.O("gocb_ref", "INLD0/LLN0$GO$g", "state", "timeout"),
		pyv.O("gocb_ref", "OUTLD0/LLN0$GO$g", "state", "waiting"),
	}
	summary := pyv.GetObj(pyv.GetObj(WithLive(linkMap, watches, true), "summary"), "SUB")
	if got, _ := pyv.Int64(pyv.Get(summary, "not_heard")); got != 1 {
		t.Errorf("not_heard %d, want 1 (only the project publisher)", got)
	}
	if got, _ := pyv.Int64(pyv.Get(summary, "remote")); got != 1 {
		t.Errorf("remote %d, want 1 (the publisher outside the project)", got)
	}
}
