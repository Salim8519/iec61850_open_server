package push

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

func TestFullThenOnlyChangedBytes(t *testing.T) {
	clock := newFakeClock()
	route := newBody(`{"a":1,"checked_at":"10:00:00"}`)
	area, _ := testArea(t, clock, &Topic{Name: "health", Get: route.get, Ignore: "checked_at", Triggers: []string{"live_*"}})
	tab := attachTab(t, area, "tab1")
	tab.subscribe("health")
	first := tab.frames(1)[0]
	if first["_raw"] != `{"t":"health","rev":1,"full":{"a":1,"checked_at":"10:00:00"}}` {
		t.Fatalf("first frame must be the exact body: %s", first["_raw"])
	}
	clock.advance(time.Second)
	route.set(`{"a":1,"checked_at":"10:00:01"}`) // only the ignored member
	area.Changed("live_status")
	settle()
	tab.frames(0)
	area.Changed("unrelated_kind")
	settle()
	clock.advance(time.Second)
	route.set(`{"a":2,"checked_at":"10:00:02"}`)
	area.Changed("live_status")
	second := tab.frames(1)[0]
	if second["_raw"] != `{"t":"health","rev":2,"full":{"a":2,"checked_at":"10:00:02"}}` {
		t.Fatalf("changed body: %s", second["_raw"])
	}
	if reads := route.reads.Load(); reads != 3 {
		t.Fatalf("an unrelated kind must not read the route: %d reads", reads)
	}
}

func TestCoalescesBurstIntoOneBuild(t *testing.T) {
	clock := newFakeClock()
	route := newBody(`{"n":0}`)
	area, _ := testArea(t, clock, &Topic{Name: "x", Get: route.get, Triggers: []string{"x"}})
	tab := attachTab(t, area, "tab1")
	tab.subscribe("x")
	tab.frames(1)
	for index := 1; index <= 50; index++ { // 50 changes within the gap
		route.set(`{"n":` + string(rune('0'+index%10)) + `}`)
		area.Changed("x")
	}
	route.set(`{"n":"last"}`)
	settle()
	tab.frames(0)
	clock.advance(DefaultGap)
	frames := tab.frames(1)
	if num(frames[0]["rev"]) != 2 || route.reads.Load() != 2 {
		t.Fatalf("one trailing build expected: %v, %d reads", frames, route.reads.Load())
	}
}

func TestKeyedDeltasAndResync(t *testing.T) {
	clock := newFakeClock()
	filler := ""
	for index := 0; index < 10; index++ { // rows that never change keep deltas smaller than bodies
		filler += fmt.Sprintf(`,{"id":"z%d","v":"unchanged row"}`, index)
	}
	list := func(head, rows, tail string) string { return head + rows + filler + tail }
	route := newBody(list(`{"note":"x","points":[`, `{"id":"a","v":1},{"id":"b","v":2},{"id":"c","v":3}`, `],"total":3}`))
	area, _ := testArea(t, clock, &Topic{Name: "pts", Get: route.get, Kind: Keyed, List: "points", Key: "id",
		Triggers: []string{"pts"}})
	tab := attachTab(t, area, "tab1")
	tab.subscribe("pts")
	shown := &mirror{}
	check := func(frame map[string]any) {
		t.Helper()
		if err := shown.apply(frame["_raw"].(string)); err != nil {
			t.Fatal(err)
		}
		if _, want := route.get(""); shown.text+"\n" != string(want) {
			t.Fatalf("client text %s, GET body %s", shown.text, want)
		}
	}
	first := tab.frames(1)[0]
	if !strings.HasPrefix(first["_raw"].(string), `{"t":"pts","rev":1,"rows":{"head":22,"keys":["a","b","c","z0",`) ||
		!strings.Contains(first["_raw"].(string), `"ends":[38,55,72,`) {
		t.Fatalf("first frame: %s", first["_raw"])
	}
	check(first)
	step := func(text string) map[string]any {
		t.Helper()
		route.set(text)
		clock.advance(time.Second)
		area.Changed("pts")
		frame := tab.frames(1)[0]
		check(frame)
		return frame
	}
	frame := step(list(`{"note":"x","points":[`, `{"id":"a","v":1},{"id":"b","v":20},{"id":"c","v":3}`, `],"total":3}`))
	if frame["_raw"] != `{"t":"pts","rev":2,"base":1,"upsert":{"b":"{\"id\":\"b\",\"v\":20}"}}` {
		t.Fatalf("one changed row: %s", frame["_raw"])
	}
	frame = step(list(`{"note":"y","points":[`, `{"id":"a","v":1},{"id":"c","v":3}`, `,{"id":"d","v":4}],"total":3}`))
	if frame["_raw"] != `{"t":"pts","rev":3,"base":2,"head":"{\"note\":\"y\",\"points\":[",`+
		`"upsert":{"d":"{\"id\":\"d\",\"v\":4}"},"remove":["b"],"add":["d"]}` {
		t.Fatalf("remove, add and head: %s", frame["_raw"])
	}
	frame = step(list(`{"note":"y","points":[`, `{"id":"c","v":3}`, `,{"id":"d","v":4},{"id":"e","v":5}],"total":4}`))
	if frame["base"] == nil || frame["tail"] != `],"total":4}` || fmt.Sprint(frame["remove"]) != "[a]" {
		t.Fatalf("a sliding window (a traffic page) is a delta: %s", frame["_raw"])
	}
	frame = step(list(`{"note":"y","points":[`, `{"id":"d","v":4},{"id":"c","v":3}`, `],"total":2}`))
	if _, full := frame["full"]; !full || num(frame["rev"]) != 5 {
		t.Fatalf("a reorder needs a full frame: %s", frame["_raw"])
	}
	frame = step(`{"error":"Choose IEC 104"}`)
	if frame["_raw"] != `{"t":"pts","rev":6,"full":{"error":"Choose IEC 104"}}` {
		t.Fatalf("an answer without the list is sent whole: %s", frame["_raw"])
	}
	frame = step(`{"points":[ {"id":"a"} ]}`)
	if _, rows := frame["rows"]; rows {
		t.Fatalf("a list with spaces cannot be cut exactly: %s", frame["_raw"])
	}
	if err := area.subscribe("tab1", []want{{topic: area.topics["pts"]}}, []string{"pts"}); err != nil {
		t.Fatal(err)
	}
	frame = tab.frames(1)[0]
	if _, full := frame["full"]; !full || num(frame["rev"]) != 7 {
		t.Fatalf("resync answers the full frame of the current revision: %s", frame["_raw"])
	}
}

// TestKeyedRandomEditsRebuildExactText drives random row edits, inserts at the end,
// removals and member changes through the hub; the mirror client must hold the exact GET
// text after every frame.
func TestKeyedRandomEditsRebuildExactText(t *testing.T) {
	clock := newFakeClock()
	random := rand.New(rand.NewPCG(7, 61850))
	route := newBody(`{"entries":[],"last":0}`)
	area, _ := testArea(t, clock, &Topic{Name: "log", Get: route.get, Kind: Keyed, List: "entries", Key: "seq",
		Triggers: []string{"log"}})
	tab := attachTab(t, area, "tab1")
	tab.subscribe("log")
	shown := &mirror{}
	if err := shown.apply(tab.frames(1)[0]["_raw"].(string)); err != nil {
		t.Fatal(err)
	}
	var seqs []int
	texts := map[int]string{}
	next, deltas := 1, 0
	for round := 0; round < 300; round++ {
		for count := 1 + random.IntN(3); count > 0; count-- {
			seqs = append(seqs, next)
			texts[next] = fmt.Sprintf("line %d é\\\"", next)
			next++
		}
		if len(seqs) > 0 && random.IntN(3) == 0 {
			texts[seqs[random.IntN(len(seqs))]] += " applied"
		}
		if len(seqs) > 20 {
			seqs = seqs[len(seqs)-20:]
		}
		rows := make([]string, len(seqs))
		for index, seq := range seqs {
			encoded, _ := pyjson.Flask(pyjson.NewObject().Set("seq", int64(seq)).Set("text", texts[seq]))
			rows[index] = strings.TrimSuffix(string(encoded), "\n")
		}
		route.set(`{"entries":[` + strings.Join(rows, ",") + `],"last":` + fmt.Sprint(next-1) + `}`)
		clock.advance(time.Second)
		area.Changed("log")
		frames := tab.frames(1)
		if frames[0]["base"] != nil {
			deltas++
		}
		if err := shown.apply(frames[0]["_raw"].(string)); err != nil {
			t.Fatal(err)
		}
		if _, want := route.get(""); shown.text+"\n" != string(want) {
			t.Fatalf("round %d: client text\n%s\nGET body\n%s", round, shown.text, want)
		}
	}
	if deltas < 200 {
		t.Fatalf("only %d of 300 changes travelled as deltas", deltas)
	}
}

func TestSharedInstanceAndOutboxCollision(t *testing.T) {
	clock := newFakeClock()
	route := newBody(`{"points":[{"id":"a","v":1}]}`)
	area, _ := testArea(t, clock, &Topic{Name: "pts", Get: route.get, Kind: Keyed, List: "points", Key: "id",
		Triggers: []string{"pts"}})
	one, two := attachTab(t, area, "one"), attachTab(t, area, "two")
	one.subscribe("pts")
	one.frames(1)
	two.subscribe("pts")
	two.frames(1)
	route.set(`{"points":[{"id":"a","v":2}]}`)
	clock.advance(time.Second)
	area.Changed("pts")
	settle()
	route.set(`{"points":[{"id":"a","v":3}]}`) // "two" does not read between the changes
	clock.advance(time.Second)
	area.Changed("pts")
	settle()
	if frame := one.frames(1)[0]; frame["_raw"] != `{"t":"pts","rev":3,"rows":{"head":11,"keys":["a"],"ends":[27]},"full":{"points":[{"id":"a","v":3}]}}` {
		t.Fatalf("two queued deltas collapse into the newest full frame: %s", frame["_raw"])
	}
	if frame := two.frames(1)[0]; num(frame["rev"]) != 3 {
		t.Fatalf("second tab: %s", frame["_raw"])
	}
	if reads := route.reads.Load(); reads != 4 { // first build, refresh for "two", two changes
		t.Fatalf("one read per change for all tabs, got %d", reads)
	}
}

func TestEmptySubscriptionStopsWork(t *testing.T) {
	clock := newFakeClock()
	route := newBody(`{"n":1}`)
	var cancelled, subscribed int
	values := func(notify func(string, string)) func() {
		subscribed++
		return func() { cancelled++ }
	}
	area := New(Options{Bus: nil, Values: values, Clock: clock, Topics: []*Topic{{Name: "panel", Get: route.get,
		Values: AnyValue, Every: time.Second, Triggers: []string{"x"}}}})
	defer area.Close()
	tab := attachTab(t, area, "tab1")
	tab.subscribe("panel")
	tab.frames(1)
	if subscribed != 1 || clock.armed() == 0 {
		t.Fatalf("value hub subscribed once (%d) and the refresh armed", subscribed)
	}
	tab.subscribe() // the tab is hidden
	area.Changed("x")
	area.valueChanged("ied", "ref")
	clock.advance(time.Minute)
	settle()
	if cancelled != 1 || route.reads.Load() != 1 || clock.armed() != 0 || len(area.instances) != 0 {
		t.Fatalf("hidden tab: cancelled %d, reads %d, timers %d, instances %d", cancelled,
			route.reads.Load(), clock.armed(), len(area.instances))
	}
}

func TestParamsSelectInstance(t *testing.T) {
	clock := newFakeClock()
	var queries []string
	get := func(query string) (int, []byte) {
		queries = append(queries, query)
		return 200, []byte(`{"points":[]}`)
	}
	area, _ := testArea(t, clock, &Topic{Name: "gateway.points", Get: get, Kind: Keyed, List: "points", Key: "id",
		Params: []string{"protocol"}})
	tab := attachTab(t, area, "tab1")
	tab.subscribeWith("gateway.points", pyjson.NewObject().Set("protocol", "modbus").Set("other", "x"))
	tab.frames(1)
	if len(queries) != 1 || queries[0] != "protocol=modbus" {
		t.Fatalf("params become the route query: %v", queries)
	}
}

func TestDetachedClientExpires(t *testing.T) {
	clock := newFakeClock()
	route := newBody(`{"n":1}`)
	area, _ := testArea(t, clock, &Topic{Name: "x", Get: route.get})
	tab := attachTab(t, area, "tab1")
	tab.subscribe("x")
	tab.frames(1)
	area.detach(tab.c, tab.gen)
	clock.advance(DefaultGrace - time.Second)
	if len(area.instances) != 1 {
		t.Fatal("subscriptions stay during the grace period")
	}
	again := attachTab(t, area, "tab1") // reconnect: full frame, no expiry
	if frame := again.frames(1)[0]; frame["full"] == nil {
		t.Fatalf("reconnect sends full: %v", frame)
	}
	area.detach(again.c, again.gen)
	clock.advance(DefaultGrace)
	if len(area.instances) != 0 || len(area.byID) != 0 || clock.armed() != 0 {
		t.Fatalf("expired: %d instances, %d clients, %d timers", len(area.instances), len(area.byID), clock.armed())
	}
}
