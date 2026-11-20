package push

import "time"

// ChangeProjectSaved is reported after File › Save project and after the auto save switch
// changed (projectapi): the "unsaved changes" answer moves without a project_changed.
const ChangeProjectSaved = "project.saved"

// LabEvery is how often the lab network status is re-read while a page shows it: the 3 s
// poll it replaces (the card's addresses and the service change outside any event).
const LabEvery = 3 * time.Second

// GooseMessagesGap coalesces one IED's GOOSE messages (their leaves carry live values):
// at most once a second, never more than the 2 s poll it replaces asked.
const GooseMessagesGap = time.Second

// projectSpecs are the Go-only topics of change 0224 (trials U2, U5): Undo/Redo and the auto
// save switch, the GOOSE message lists of the SLD and the lab network status. A project change
// re-reads every active topic already (globalTriggers), and an unchanged answer sends nothing,
// so a diagram edit costs no request for them.
func projectSpecs() []Spec {
	whole := func(name, pattern string, triggers ...string) Spec {
		return Spec{Pattern: "/api/v1" + pattern, Topic: Topic{Name: name, Triggers: triggers}}
	}
	messages := whole("goose.messages", "/goose/messages", "goose_*")
	messages.Topic.Params, messages.Topic.Values, messages.Topic.Gap = []string{"ied"}, AnyValue, GooseMessagesGap
	lab := whole("network.lab", "/network/lab")
	lab.Topic.Every, lab.Topic.While = LabEvery, func() bool { return true }
	return []Spec{
		whole("project.history", "/project/history"),
		whole("project.autosave", "/project/autosave", ChangeProjectSaved),
		whole("goose.members", "/goose/members"),
		messages,
		lab,
	}
}
