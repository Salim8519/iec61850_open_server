package liveapi

import (
	"net/http"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/goosemsg"
	"studio-engine/internal/pyjson"
)

// GoOnlyRoutes are the live routes the frozen Python app does not have: the GOOSE messages of
// one IED and the exact leaf references of every message (ADR-0112).
var GoOnlyRoutes = map[string]bool{"GET /api/v1/ieds/{ied_id}/goose/messages": true, "GET /api/v1/goose/members": true}

// gooseMessages is GET /ieds/{ied_id}/goose/messages: the IED's sendable messages, who receives
// them, every member with its leaves (and their values while running), what it receives, and
// whether it can send now (a plain-words reason when not).
func (s *Service) gooseMessages(w http.ResponseWriter, r *http.Request) {
	ied, err := s.Deps.Projects.FindIED(r.PathValue("ied_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	project := s.Deps.Projects.Current()
	declarations, err := s.Declarations(project)
	if err != nil {
		writeError(w, err)
		return
	}
	ieds := []*pyjson.Object{}
	for _, item := range pyv.Objects(pyv.GetList(project, "ieds")) {
		ieds = append(ieds, pyv.O("id", pyv.Get(item, "id"), "name", pyv.Get(item, "name"),
			"role", pyv.GetOr(item, "role", "simulated")))
	}
	var partners []goosemsg.Partner
	for _, link := range pyv.Objects(pyv.GetList(goosedom.BuildLinkMap(declarations, ieds), "links")) {
		partners = append(partners, goosemsg.Partner{GocbRef: textOf(pyv.Get(link, "gocb_ref")),
			Publisher: textOf(pyv.Get(link, "publisher")), Subscriber: textOf(pyv.Get(link, "subscriber")),
			Block: textOf(pyv.Get(link, "block")), Status: textOf(pyv.Get(link, "status"))})
	}
	id, _ := pyv.Get(ied, "id").(string)
	running := s.running()
	state := goosemsg.SendState{Running: running, Refusal: goosemsg.NotRunning}
	if running {
		state.Refusal = goosemsg.Refusal(ied, s.Deps.Values.Has(id))
	}
	value := func(ref string) (any, bool) {
		current, _, found := s.Deps.Values.Get(id, ref)
		return current, found
	}
	role, _ := pyv.GetOr(ied, "role", "simulated").(string)
	writeJSON(w, http.StatusOK, goosemsg.Messages(goosemsg.For(ied), role, partners, value, state))
}

// gooseMembers is GET /goose/members: {ieds: {ied_id: {gocb_ref: [leaf refs]}}} for every project
// IED, the exact leaves of each sendable message with the member's functional constraint.
func (s *Service) gooseMembers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, goosemsg.LeafRefs(s.ieds()))
}

// textOf is a text field of a link ("" for null).
func textOf(value any) string {
	text, _ := value.(string)
	return text
}
