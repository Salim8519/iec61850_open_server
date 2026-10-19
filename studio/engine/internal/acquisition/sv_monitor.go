package acquisition

import (
	"sync"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/acquisition/svdom"
	"studio-engine/internal/pyjson"
)

var (
	svUnknownFields = []string{"app_id", "mac", "src_mac", "vlan_id", "sv_id", "simulate"}
	svReceived      = []string{"asdus", "rate", "sv_id", "conf_rev", "smp_rate", "smp_mod", "smp_synch", "lost",
		"restarts", "invalid_quality", "data_size", "simulate", "values", "raw"}
)

// SvMonitor listens to Sampled Values on the lab adapter and supervises each SCD stream
// (acquisition/sv_monitor.py).
type SvMonitor struct {
	deps Deps

	mu            sync.Mutex
	process       Worker
	watches       *pyjson.Object
	notSupervised []*pyjson.Object
	status        *pyjson.Object
}

// NewSvMonitor returns a stopped monitor.
func NewSvMonitor(deps Deps) *SvMonitor {
	return &SvMonitor{deps: deps, watches: pyjson.NewObject(), status: offStatus()}
}

func (s *SvMonitor) setStatus(status *pyjson.Object) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}

// Start supervises the project's SV streams when the lab adapter and Npcap allow it.
func (s *SvMonitor) Start(project *pyjson.Object, npcapInstalled bool) error {
	network := pyv.GetObj(project, "network")
	nominal := pyv.GetOr(network, "nominal_hz", int64(50))
	supervised, extra := svdom.WatchList(pyv.Objects(pyv.GetList(project, "ieds")), nominal)
	watches := pyjson.NewObject()
	for _, watch := range supervised {
		watches.Set(pyv.GetStr(watch, "key"), watch)
	}
	s.mu.Lock()
	s.watches, s.notSupervised = watches, extra
	s.mu.Unlock()
	switch {
	case watches.Len() == 0:
		s.setStatus(pyv.O("state", "nothing_to_watch", "unknown", []any{},
			"message", "No IED in this project sends Sampled Values."))
		return nil
	case !pyv.Equal(pyv.Get(network, "mode"), "lab") || !pyv.Truthy(pyv.Get(network, "adapter_guid")):
		s.setStatus(pyv.O("state", "off", "unknown", []any{},
			"message", "Sampled Values need a network adapter. Set up the lab network to use them."))
		return nil
	case !npcapInstalled:
		s.setStatus(pyv.O("state", "needs_npcap", "unknown", []any{},
			"message", "Install the free Npcap driver to receive Sampled Values."))
		return nil
	}
	streams := []any{}
	s.mu.Lock()
	for _, key := range watches.Keys() {
		watch := pyv.GetObj(watches, key)
		watch.Set("state", "waiting").Set("problems", []any{})
		streams = append(streams, svdom.NativeStream(watch))
	}
	s.mu.Unlock()
	process, err := s.deps.Start(s.deps.LiveExecutable, s.onEvent)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.process = process
	s.mu.Unlock()
	if _, err := process.Request("sv_start", 30*time.Second,
		pyv.O("adapter_guid", pyv.Get(network, "adapter_guid"), "streams", streams)); err != nil {
		return err
	}
	label := pyv.Get(network, "adapter_label")
	if !pyv.Truthy(label) {
		label = "the lab adapter"
	}
	s.setStatus(pyv.O("state", "listening", "unknown", []any{}, "nominal_hz", nominal,
		"message", "Listening for Sampled Values on "+pyv.PyStr(label)))
	return nil
}

func (s *SvMonitor) onEvent(event *pyjson.Object) {
	switch pyv.GetStr(event, "event") {
	case "sv_rx":
		key, _ := pyv.Get(event, "key").(string)
		s.mu.Lock()
		watch := pyv.GetObj(s.watches, key)
		if watch == nil {
			s.mu.Unlock()
			return
		}
		previous := pyv.O("lost", pyv.Get(watch, "received_lost"), "invalid_quality", pyv.Get(watch, "received_invalid_quality"))
		verdict := svdom.Assess(watch, event, previous)
		changed := !pyv.Equal(verdict, pyv.O("state", pyv.Get(watch, "state"), "problems", pyv.Get(watch, "problems")))
		pyv.Update(watch, verdict)
		for _, name := range svReceived {
			watch.Set("received_"+name, pyv.Get(event, name))
		}
		watch.Set("observed_at", pyv.ISOSeconds(s.deps.Clock.Now()))
		s.mu.Unlock()
		if changed {
			s.deps.Events.Emit("sv_status", pyv.Merge(pyv.O("key", pyv.Get(watch, "key")), verdict))
		}
	case "sv_unknown":
		entry := pyjson.NewObject()
		for _, name := range svUnknownFields {
			entry.Set(name, pyv.Get(event, name))
		}
		s.mu.Lock()
		s.status = pyv.Shallow(s.status)
		s.status.Set("unknown", append(append([]any{}, pyv.GetList(s.status, "unknown")...), entry))
		s.mu.Unlock()
	case "sv_unknown_cap":
		s.mu.Lock()
		s.status = pyv.Shallow(s.status)
		s.status.Set("unknown_limit", pyv.Get(event, "limit"))
		s.mu.Unlock()
	case "worker_failed":
		s.mu.Lock()
		s.status = pyv.Shallow(s.status)
		s.status.Set("state", "failed").Set("message", "The Sampled Values listener stopped unexpectedly")
		s.mu.Unlock()
	}
}

// Snapshot is {status..., watches, not_supervised}.
func (s *SvMonitor) Snapshot() *pyjson.Object {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := pyv.Shallow(s.status)
	result.Set("unknown", append([]any{}, pyv.GetList(s.status, "unknown")...))
	watches := []any{}
	for _, key := range s.watches.Keys() {
		watches = append(watches, pyv.Shallow(pyv.GetObj(s.watches, key)))
	}
	extra := []any{}
	for _, item := range s.notSupervised {
		extra = append(extra, pyv.O("key", pyv.Get(item, "key"), "publisher", pyv.Get(item, "publisher")))
	}
	result.Set("watches", watches)
	result.Set("not_supervised", extra)
	return result
}

// Stop closes the listener and returns to "off".
func (s *SvMonitor) Stop() {
	s.mu.Lock()
	process := s.process
	s.process = nil
	s.status = offStatus()
	s.mu.Unlock()
	if process != nil {
		process.Close()
	}
}
