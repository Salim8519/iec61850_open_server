package push

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"studio-engine/internal/events"
)

// Values subscribes notify to the value hub (runtime.Coordinator.Subscribe): one call per
// value, quality, timestamp or live-state write (iedID, ref), per IED state change
// (iedID, "") and per run-state change or resync ("", ""). notify never blocks.
type Values func(notify func(iedID, ref string)) (cancel func())

// Options configure an Area. Bus and Topics are required.
type Options struct {
	Bus    *events.Bus
	Values Values // nil: the bus "signal" events stand in for the value hub
	Topics []*Topic
	// Listen sees every bus event the area reads (after it marked the topics); it must not
	// block. The composition uses it for gateway.Manager.ProjectChanged on project_changed.
	Listen func(event *events.Event)
	Clock  Clock // nil: the wall clock
	// Heartbeat is the quiet time before ": heartbeat" (10 s, as /api/v1/events).
	Heartbeat time.Duration
	// Gather is how long bus events are collected after a write that carried events, so a
	// burst leaves in few writes; the first event after a quiet period leaves at once.
	Gather time.Duration
	// Settle is how long the first events after a quiet period wait for the rest of their
	// burst before they are written: one request publishes its events microseconds apart
	// (a test press: signal, equipment_state, energization), and without it only the first
	// leaves at once while the others wait out Gather.
	Settle time.Duration
	// Grace keeps a tab's subscriptions after its stream closed, for a reconnect.
	Grace      time.Duration
	MaxClients int
	Epoch      string // the boot id; random when empty
}

// Defaults of Options.
const (
	DefaultGather     = 16 * time.Millisecond
	DefaultSettle     = time.Millisecond
	DefaultGrace      = 30 * time.Second
	DefaultMaxClients = 32
	forever           = 24 * time.Hour
)

// Area is the push stream: GET /api/v1/capabilities, GET /api/v1/push and
// PUT /api/v1/push/topics. Changed is the OnChange hook for services.
type Area struct {
	bus     *events.Bus
	values  Values
	listen  func(event *events.Event)
	clock   Clock
	topics  map[string]*Topic
	names   []string
	epoch   string
	beat    time.Duration
	gather  time.Duration
	settle  time.Duration
	grace   time.Duration
	clients int

	mu           sync.Mutex
	byID         map[string]*client
	instances    map[string]*instance
	valueUsers   int
	cancelValues func()

	buildMu   sync.Mutex
	builds    []*instance
	buildWake chan struct{}
	stop      chan struct{}
	closing   sync.Once
	finished  sync.WaitGroup
}

// New starts an Area: one goroutine reading the bus and one building topics, both blocked
// on channels while nothing happens.
func New(o Options) *Area {
	a := &Area{bus: o.Bus, values: o.Values, listen: o.Listen, clock: o.Clock, topics: map[string]*Topic{}, names: Names(o.Topics),
		epoch: o.Epoch, beat: o.Heartbeat, gather: o.Gather, settle: o.Settle, grace: o.Grace, clients: o.MaxClients,
		byID: map[string]*client{}, instances: map[string]*instance{}, buildWake: make(chan struct{}, 1),
		stop: make(chan struct{})}
	for _, topic := range o.Topics {
		a.topics[topic.Name] = topic
	}
	if a.clock == nil {
		a.clock = WallClock
	}
	if a.epoch == "" {
		var raw [8]byte
		_, _ = rand.Read(raw[:])
		a.epoch = hex.EncodeToString(raw[:])
	}
	if a.beat <= 0 {
		a.beat = events.Heartbeat
	}
	if a.gather <= 0 {
		a.gather = DefaultGather
	}
	if a.settle <= 0 {
		a.settle = DefaultSettle
	}
	if a.grace <= 0 {
		a.grace = DefaultGrace
	}
	if a.clients <= 0 {
		a.clients = DefaultMaxClients
	}
	a.finished.Add(2)
	go a.buildLoop()
	go a.readBus()
	return a
}

// Epoch is the boot id carried by /capabilities and the stream's hello frame.
func (a *Area) Epoch() string { return a.epoch }

// Close stops the goroutines and timers (tests; the server lives as long as the process).
func (a *Area) Close() {
	a.closing.Do(func() {
		close(a.stop)
		a.mu.Lock()
		for _, c := range a.byID {
			a.dropLocked(c)
		}
		a.mu.Unlock()
	})
	a.finished.Wait()
}

// Changed marks the topics a change name or bus event kind triggers. It is the OnChange
// hook of the gateway, master, explorer and scan services; it never blocks.
func (a *Area) Changed(name string) {
	global := globalTriggers[name]
	a.mu.Lock()
	for _, in := range a.instances {
		if global || in.topic.triggeredBy(name) {
			in.throttle.mark()
		}
	}
	a.mu.Unlock()
}

// valueChanged marks the value-driven topics a value hub change concerns.
func (a *Area) valueChanged(iedID, ref string) {
	a.mu.Lock()
	for _, in := range a.instances {
		if values := in.topic.Values; iedID == "" || (values != nil && values(iedID, ref)) {
			in.throttle.mark()
		}
	}
	a.mu.Unlock()
}

// readBus turns bus events into marks; it reads the ring like any stream does.
func (a *Area) readBus() {
	defer a.finished.Done()
	if a.bus == nil {
		return
	}
	cursor := a.bus.Sequence()
	for {
		items := a.bus.After(cursor, forever, a.stop)
		select {
		case <-a.stop:
			return
		default:
		}
		for _, event := range items {
			cursor = event.ID
			if a.listen != nil && event.Kind != "signal" {
				a.listen(event)
			}
			if event.Kind == "signal" {
				if a.values == nil {
					ied, _ := event.Payload.Get("ied_id")
					ref, _ := event.Payload.Get("ref")
					iedText, _ := ied.(string)
					refText, _ := ref.(string)
					a.valueChanged(iedText, refText)
				}
				continue
			}
			a.Changed(event.Kind)
		}
	}
}

// queue hands a fired instance to the build goroutine.
func (a *Area) queue(in *instance) {
	a.buildMu.Lock()
	a.builds = append(a.builds, in)
	a.buildMu.Unlock()
	select {
	case a.buildWake <- struct{}{}:
	default:
	}
}

// buildLoop builds fired topics one after another, so a burst costs one core at most.
func (a *Area) buildLoop() {
	defer a.finished.Done()
	var batch []*instance
	for {
		select {
		case <-a.stop:
			return
		case <-a.buildWake:
		}
		a.buildMu.Lock()
		batch = append(batch[:0], a.builds...)
		clear(a.builds)
		a.builds = a.builds[:0]
		a.buildMu.Unlock()
		for _, in := range batch {
			in.build()
		}
		clear(batch)
	}
}

// publish offers frame (revision rev) to every client subscribed to in.
func (a *Area) publish(in *instance, frame []byte, rev int64) {
	a.mu.Lock()
	for c := range in.subs {
		c.offer(in, frame, rev)
	}
	a.mu.Unlock()
}
