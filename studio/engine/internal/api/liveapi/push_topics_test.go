package liveapi

import (
	"fmt"
	"testing"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/push"
)

// TestPushFullFramesMatchPython replays the Flask scenario of TestRoutesMatchPython, but
// answers every GET of a push topic through the push path: the topic's full frame must
// carry exactly the body Flask recorded (speed lane G).
func TestPushFullFramesMatchPython(t *testing.T) {
	doc := loadScenario(t)
	e := newEnv(t, doc)
	defer e.service.Live.Stop()
	index := push.IndexRoutes(e.service.Routes())
	topics := map[string]*push.Topic{}
	for _, spec := range push.Specs(push.CatalogOptions{}) {
		if get, found := index.Getter(spec.Pattern); found {
			topic := spec.Topic
			topic.Get = get
			topics[spec.Pattern] = &topic
		}
	}
	checked := map[string]int{}
	for at, step := range pyv.Objects(pyv.GetList(doc, "steps")) {
		switch pyv.GetStr(step, "step") {
		case "project":
			e.projects.project = pyv.GetObj(step, "project")
			e.projects.runID = pyv.GetStr(step, "run_id")
			e.decls = goosedom.DeclarationsFromJSON(pyv.GetObj(step, "declarations"))
		case "http":
			topic := topics[pyv.GetStr(step, "path")]
			if topic == nil || pyv.GetStr(step, "method") != "GET" {
				e.http(at, step)
				continue
			}
			e.events.take()
			status, _ := pyv.Int64(pyv.Get(step, "status"))
			want := topic.ExpectedFullFrame(1, int(status), []byte(goExpected(pyv.GetStr(step, "path"), pyv.GetStr(step, "response"))))
			if got := topic.FullFrame(nil, 1); string(got) != string(want) {
				t.Errorf("step %d %s: full frame differs\n got: %s\nwant: %s", at, topic.Name, clip(string(got)), clip(string(want)))
			}
			checked[topic.Name]++
			e.compareEvents(at, "GET "+topic.Name, step, false)
		case "start":
			e.start(at, step)
		case "event":
			e.inject(at, step)
		case "stop":
			e.service.Live.Stop()
			e.compareEvents(at, "stop", step, false)
		}
	}
	served, notPolled, uncovered := push.Coverage(e.service.Routes())
	if fmt.Sprint(served) != "[goose.links goose.overview goose.share health live.sessions mms.access sv]" ||
		len(notPolled)+len(uncovered) > 0 {
		t.Errorf("topics %v, not polled %v, polled without a topic %v", served, notPolled, uncovered)
	}
	t.Logf("full frames compared with Flask bodies: %v", checked)
	for _, name := range []string{"health", "goose.links", "goose.overview", "sv", "live.sessions", "mms.access", "goose.share"} {
		if checked[name] == 0 {
			t.Errorf("the scenario never asked for %s", name)
		}
	}
}
