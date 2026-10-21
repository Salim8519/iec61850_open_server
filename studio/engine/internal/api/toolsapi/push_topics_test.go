package toolsapi

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	"studio-engine/internal/push"
	"studio-engine/internal/pyjson"
)

var rowCuts = regexp.MustCompile(`,"rows":\{[^}]*\}`)

// pushCheck compares the push topics of this area with the Flask answers of the replay.
type pushCheck struct {
	topics  map[string]*push.Topic // by GET path under /api/v1
	checked map[string]int
}

func newPushCheck(deps Deps) *pushCheck {
	index := push.IndexRoutes(New(deps).Routes())
	check := &pushCheck{topics: map[string]*push.Topic{}, checked: map[string]int{}}
	for _, spec := range push.Specs(push.CatalogOptions{}) {
		if get, found := index.Getter(spec.Pattern); found {
			topic := spec.Topic
			topic.Get = get
			check.topics[strings.TrimPrefix(spec.Pattern, "/api/v1")] = &topic
		}
	}
	return check
}

// compare: when the replayed GET is a push topic (with only the topic's parameters), its
// full frame must carry Flask's exact body (times normalised as the replay does).
func (p *pushCheck) compare(t *testing.T, number int, method, path string, status int64, want string) {
	t.Helper()
	route, query, _ := strings.Cut(path, "?")
	topic := p.topics[route]
	if method != "GET" || topic == nil {
		return
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return
	}
	params := pyjson.NewObject()
	for name := range values {
		if !slices.Contains(topic.Params, name) {
			return // a cursor such as ?after= belongs to polling, not to the topic
		}
		params.Set(name, values.Get(name))
	}
	// Row offsets follow the run's own times ("at"), so they are compared through the body.
	got := normalise(rowCuts.ReplaceAllString(string(topic.FullFrame(params, 1)), ""))
	expected := normalise(rowCuts.ReplaceAllString(string(topic.ExpectedFullFrame(1, int(status), []byte(want))), ""))
	if got != expected {
		t.Errorf("request %d push %s full frame\n got %s\nwant %s", number, topic.Name, got, expected)
	}
	p.checked[topic.Name]++
}

func (p *pushCheck) report(t *testing.T) {
	t.Helper()
	served, notPolled, uncovered := push.Coverage(New(Deps{}).Routes())
	if fmt.Sprint(served) != "[activity explorer gateway.commands gateway.points gateway.status gateway.traffic "+
		"master.commands master.points master.status master.traffic scan]" || len(uncovered) > 0 ||
		fmt.Sprint(notPolled) != "[gateway.config master.config]" { // read once per view today
		t.Errorf("topics %v, not polled %v, polled without a topic %v", served, notPolled, uncovered)
	}
	t.Logf("push full frames compared with Flask bodies: %v", p.checked)
	// GET /tools/explorer is not in the Flask scenarios (it warms a studio_live worker).
	for _, name := range []string{"gateway.status", "gateway.config", "gateway.points", "gateway.commands",
		"gateway.traffic", "master.status", "master.config", "master.points", "master.commands", "master.traffic",
		"activity", "scan"} {
		if p.checked[name] == 0 {
			t.Errorf("the Flask scenarios never asked for push topic %s", name)
		}
	}
}
