package push

import (
	"fmt"
	"testing"
)

// Change 0224 (trials U2, U5): Undo/Redo, the auto save switch, the GOOSE message lists and the
// lab network status are topics, so the browser stops asking for them after every diagram edit
// and stops polling them every 2-3 s.
func TestProjectTopics(t *testing.T) {
	byName := map[string]Spec{}
	for _, spec := range Specs(CatalogOptions{}) {
		byName[spec.Topic.Name] = spec
	}
	want := map[string]string{"project.history": "/api/v1/project/history", "project.autosave": "/api/v1/project/autosave",
		"goose.members": "/api/v1/goose/members", "goose.messages": "/api/v1/goose/messages", "network.lab": "/api/v1/network/lab"}
	for name, pattern := range want {
		if spec, found := byName[name]; !found || spec.Pattern != pattern {
			t.Errorf("%s: %+v", name, spec)
		}
	}
	autosave := byName["project.autosave"].Topic
	if !autosave.triggeredBy(ChangeProjectSaved) {
		t.Error("project.autosave is not re-read after Save or the switch")
	}
	messages := byName["goose.messages"].Topic
	if fmt.Sprint(messages.Params) != "[ied]" || messages.Values == nil || messages.Gap != GooseMessagesGap ||
		!messages.triggeredBy("goose_status") {
		t.Errorf("goose.messages: %+v", messages)
	}
	lab := byName["network.lab"].Topic
	if lab.Every != LabEvery || lab.While == nil || !lab.While() {
		t.Errorf("network.lab must re-read every 3 s while shown: %+v", lab)
	}
	// A diagram edit (project_changed) re-reads every active topic; an unchanged answer sends nothing.
	for name := range want {
		if !globalTriggers["project_changed"] {
			t.Fatalf("%s: project_changed no longer re-reads every topic", name)
		}
	}
}
