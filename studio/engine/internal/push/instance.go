package push

import (
	"bytes"
	"strconv"
	"sync"
	"sync/atomic"

	"studio-engine/internal/pyjson"
)

// instance is one topic with one query string, shared by every client that subscribes to
// it: the body is computed once per change and its frame encoded once for all of them.
type instance struct {
	area     *Area
	topic    *Topic
	key      string // name + "?" + query
	query    string
	throttle *throttle
	closed   atomic.Bool
	subs     map[*client]struct{} // guarded by area.mu

	mu      sync.Mutex
	rev     int64
	status  int
	body    []byte
	compare []byte
	parsed  rows
	full    []byte // the encoded full frame of rev; nil until asked for
	refresh Timer  // the Every re-read while armed
}

func newInstance(area *Area, topic *Topic, query string) *instance {
	in := &instance{area: area, topic: topic, key: topic.Name + "?" + query, query: query,
		subs: map[*client]struct{}{}}
	gap := topic.Gap
	if gap <= 0 {
		gap = DefaultGap
	}
	in.throttle = newThrottle(area.clock, gap, func() { area.queue(in) })
	return in
}

// build reads the route, compares with what was sent and publishes the frame of a change.
// It runs on the area's build goroutine, never under a lock of the area or a client.
func (in *instance) build() {
	defer in.throttle.done()
	if in.closed.Load() {
		return
	}
	status, body := in.topic.Get(in.query)
	body = bytes.TrimRight(body, "\n")
	compare := body
	if in.topic.Ignore != "" {
		compare = withoutMember(body, in.topic.Ignore)
	}
	in.mu.Lock()
	if in.rev > 0 && status == in.status && bytes.Equal(compare, in.compare) {
		if !bytes.Equal(body, in.body) {
			in.body, in.full = body, nil // newer ignored member for the next subscriber's full
		}
		in.mu.Unlock()
		in.armRefresh()
		return
	}
	var parsed rows
	if in.topic.Kind != Whole && status == 200 {
		parsed = in.topic.parse(body, &in.parsed)
	}
	rev := in.rev + 1
	var frame []byte
	if in.rev > 0 && status == 200 && in.status == 200 && in.topic.Kind != Whole {
		if members, ok := in.topic.delta(&in.parsed, &parsed, nil); ok && len(members) < len(body) {
			frame = deltaFrame(in.topic.Name, rev, in.rev, members)
		}
	}
	in.rev, in.status, in.body, in.compare, in.parsed, in.full = rev, status, body, compare, parsed, nil
	if frame == nil {
		frame = fullFrame(in.topic.Name, rev, status, body, &in.parsed)
		in.full = frame
	}
	in.mu.Unlock()
	in.area.publish(in, frame, rev)
	in.armRefresh()
}

// fullFrame returns the full frame of the current revision and the revision, or nil
// before the first build.
func (in *instance) fullFrame() ([]byte, int64) {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.rev == 0 {
		return nil, 0
	}
	if in.full == nil {
		in.full = fullFrame(in.topic.Name, in.rev, in.status, in.body, &in.parsed)
	}
	return in.full, in.rev
}

// armRefresh re-reads after Every while the topic's condition holds (counters, staleness).
func (in *instance) armRefresh() {
	topic := in.topic
	if topic.Every <= 0 || in.closed.Load() || (topic.While != nil && !topic.While()) {
		return
	}
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.refresh == nil {
		in.refresh = in.area.clock.AfterFunc(topic.Every, func() {
			in.mu.Lock()
			in.refresh = nil
			in.mu.Unlock()
			in.throttle.mark()
		})
	}
}

// close stops the instance's timers; a queued build then does nothing.
func (in *instance) close() {
	in.closed.Store(true)
	in.throttle.close()
	in.mu.Lock()
	if in.refresh != nil {
		in.refresh.Stop()
		in.refresh = nil
	}
	in.mu.Unlock()
}

// FullFrame reads the route now and returns the full frame a new subscriber gets at rev
// (route parity tests in the area packages compare it with the recorded Flask body).
func (t *Topic) FullFrame(params *pyjson.Object, rev int64) []byte {
	status, body := t.Get(t.query(params))
	return t.frameOf(rev, status, bytes.TrimRight(body, "\n"))
}

// ExpectedFullFrame is the full frame of t for a recorded GET answer (status, body).
func (t *Topic) ExpectedFullFrame(rev int64, status int, body []byte) []byte {
	return t.frameOf(rev, status, bytes.TrimRight(body, "\n"))
}

func (t *Topic) frameOf(rev int64, status int, body []byte) []byte {
	var parsed rows
	if t.Kind == Keyed && status == 200 {
		parsed = t.parse(body, &rows{})
	}
	return fullFrame(t.Name, rev, status, body, &parsed)
}

// withoutMember returns the object body without its top-level member name (a copy).
func withoutMember(body []byte, name string) []byte {
	members, ok := objectFields(body, nil)
	if !ok {
		return body
	}
	key := quote(name)
	out := make([]byte, 0, len(body))
	out = append(out, '{')
	for _, member := range members {
		if bytes.Equal(member.key, key) {
			continue
		}
		if len(out) > 1 {
			out = append(out, ',')
		}
		out = append(append(append(out, member.key...), ':'), member.value...)
	}
	return append(out, '}')
}

// fullFrame encodes {"t":name,"rev":rev[,"status":status][,"rows":{..}],"full":body};
// status is present only when the route did not answer 200 (body is then its error
// object), rows only for a Keyed body that cuts into rows. "full" is always the last
// member, so the exact body text is the frame text between `"full":` and the final "}".
func fullFrame(name string, rev int64, status int, body []byte, parsed *rows) []byte {
	out := make([]byte, 0, len(body)+len(name)+48)
	out = frameHead(out, name, rev)
	if status != 200 {
		out = strconv.AppendInt(append(out, `,"status":`...), int64(status), 10)
	}
	if parsed != nil && parsed.valid {
		out = appendRows(out, parsed)
	}
	out = append(out, `,"full":`...)
	if len(body) == 0 {
		out = append(out, "null"...)
	}
	return append(append(out, body...), '}')
}

// deltaFrame encodes {"t":name,"rev":rev,"base":base<members>}.
func deltaFrame(name string, rev, base int64, members []byte) []byte {
	out := make([]byte, 0, len(members)+len(name)+48)
	out = frameHead(out, name, rev)
	out = strconv.AppendInt(append(out, `,"base":`...), base, 10)
	return append(append(out, members...), '}')
}

func frameHead(out []byte, name string, rev int64) []byte {
	out = pyjson.AppendString(append(out, `{"t":`...), name)
	return strconv.AppendInt(append(out, `,"rev":`...), rev, 10)
}
