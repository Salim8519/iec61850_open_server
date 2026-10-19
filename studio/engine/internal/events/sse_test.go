package events

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

func readFrames(t *testing.T, reader *bufio.Reader, count int) []string {
	var frames []string
	var frame strings.Builder
	for len(frames) < count {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("stream ended after %q: %v", frames, err)
		}
		frame.WriteString(line)
		if line == "\n" {
			frames = append(frames, frame.String())
			frame.Reset()
		}
	}
	return frames
}

func open(t *testing.T, url, lastEventID string) *bufio.Reader {
	request, _ := http.NewRequest("GET", url, nil)
	if lastEventID != "" {
		request.Header.Set("Last-Event-ID", lastEventID)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { response.Body.Close() })
	if response.Header.Get("Content-Type") != "text/event-stream; charset=utf-8" ||
		response.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("headers %v", response.Header)
	}
	return bufio.NewReader(response.Body)
}

// TestStreamFraming: full replay on connect, live events, heartbeat when quiet, resync
// for a cursor older than the ring (cursor < oldest id - 1), all with api/events.py framing.
func TestStreamFraming(t *testing.T) {
	bus := NewBus(3)
	clock := 1760900000.0
	bus.SetClock(func() float64 { clock += 0.5; return clock })
	for _, ref := range []string{"a", "b", "c", "d", "e"} {
		_, _ = bus.Publish("signal", pyjson.NewObject().Set("ref", ref).Set("value", 1.0))
	}
	server := httptest.NewServer(Area{Bus: bus, Heartbeat: 100 * time.Millisecond}.Routes()[0].Handler)
	t.Cleanup(server.Close) // runs after the streams' bodies are closed (cleanups run last-in first-out)
	frames := readFrames(t, open(t, server.URL, ""), 5)
	want := []string{"retry: 1000\n\n",
		"id: 3\ndata: {\"id\": 3, \"kind\": \"signal\", \"timestamp\": 1760900001.5, \"ref\": \"c\", \"value\": 1.0}\n\n",
		"id: 4\ndata: {\"id\": 4, \"kind\": \"signal\", \"timestamp\": 1760900002.0, \"ref\": \"d\", \"value\": 1.0}\n\n",
		"id: 5\ndata: {\"id\": 5, \"kind\": \"signal\", \"timestamp\": 1760900002.5, \"ref\": \"e\", \"value\": 1.0}\n\n",
		": heartbeat\n\n"}
	for index := range want {
		if frames[index] != want[index] {
			t.Errorf("frame %d: %q, want %q", index, frames[index], want[index])
		}
	}
	resync := readFrames(t, open(t, server.URL, "1"), 2)
	if resync[1] != "id: 5\ndata: {\"id\": 5, \"kind\": \"resync\", \"timestamp\": 1760900003.0}\n\n" {
		t.Errorf("resync frame %q", resync[1])
	}
	live := open(t, server.URL, " +5 ")
	readFrames(t, live, 1)
	go func() {
		time.Sleep(20 * time.Millisecond)
		_, _ = bus.Publish("runtime", pyjson.NewObject().Set("state", "stopped"))
	}()
	if frame := readFrames(t, live, 1)[0]; !strings.HasPrefix(frame, "id: 6\ndata: {\"id\": 6, \"kind\": \"runtime\"") {
		t.Errorf("live frame %q", frame)
	}
}
