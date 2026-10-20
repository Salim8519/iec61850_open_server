package runtimeapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/push"
)

// TestPushFullFramesMatchFlask replays the stopped-state Flask requests; every GET of a push
// topic is answered through the push path and its full frame must carry Flask's exact body.
func TestPushFullFramesMatchFlask(t *testing.T) {
	server, coordinator, memory := newServer(t, 21320)
	index := push.IndexRoutes(Area{Runtime: coordinator, Project: memory}.Routes())
	topics := map[string]*push.Topic{}
	for _, spec := range push.Specs(push.CatalogOptions{}) {
		if get, found := index.Getter(spec.Pattern); found {
			topic := spec.Topic
			topic.Get = get
			topics[spec.Pattern] = &topic
		}
	}
	var requests [][4]string
	var golden [][2]any
	rawRequests, _ := os.ReadFile(filepath.Join("testdata", "stopped_requests.json"))
	rawGolden, _ := os.ReadFile(filepath.Join("testdata", "stopped_golden.json"))
	if json.Unmarshal(rawRequests, &requests) != nil || json.Unmarshal(rawGolden, &golden) != nil ||
		len(golden) != len(requests) {
		t.Fatal("stopped_requests.json and stopped_golden.json do not match; run record_flask.py")
	}
	checked := map[string]int{}
	for at, request := range requests {
		status, body := int(golden[at][0].(float64)), golden[at][1].(string)
		topic := topics["/api/v1"+request[1]]
		if request[0] != "GET" || topic == nil {
			call(t, server, request[0], request[1], request[2], request[3])
			continue
		}
		want := topic.ExpectedFullFrame(1, status, []byte(body))
		if got := topic.FullFrame(nil, 1); string(got) != string(want) {
			t.Errorf("%s: full frame %s, Flask body gives %s", topic.Name, got, want)
		}
		checked[topic.Name]++
	}
	served, notPolled, uncovered := push.Coverage(Area{}.Routes())
	if fmt.Sprint(served) != "[scenario.runs test.panel]" || len(notPolled)+len(uncovered) > 0 {
		t.Errorf("topics %v, not polled %v, polled without a topic %v", served, notPolled, uncovered)
	}
	if checked["test.panel"] == 0 || checked["scenario.runs"] == 0 {
		t.Fatalf("the goldens must cover test.panel and scenario.runs: %v", checked)
	}
}
