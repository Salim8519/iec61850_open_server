package service

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// deviceState is a simulated device's run state; a device is listed only while it runs or
// after it failed to start.
type deviceState struct {
	State string // "running" or "failed"
	Error string
}

// onEvent handles studio_snmp's events on the worker's reader goroutine; it never blocks.
func (s *Service) onEvent(event *pyjson.Object) {
	kind := pytext.Str(field(event, "event"))
	switch kind {
	case "agent_request":
		s.logRequest(event)
	case "agent_written":
		s.logWrite(event)
	case "trap":
		s.logTrap(event)
	case "read_values":
		s.mergeValues(event)
	case "walk_progress":
		s.mu.Lock()
		if count, isInt := field(event, "count").(int64); isInt {
			s.reader.walkCount = count
		}
		s.mu.Unlock()
		s.changed(ChangeStatus)
	case "native_diagnostic":
		s.logf("studio_snmp: %s", pytext.Str(field(event, "message")))
	case "worker_failed":
		s.mu.Lock()
		s.process = nil
		s.failure = "The SNMP engine stopped unexpectedly: " + pytext.Str(field(event, "error"))
		for id := range s.devices {
			s.devices[id] = deviceState{State: "failed", Error: "stopped with the SNMP engine"}
		}
		if s.reader.state != "disconnected" {
			s.reader.state, s.reader.message = "failed", s.failure
		}
		s.listening, s.trapText = false, s.failure
		s.mu.Unlock()
		s.logf("%s", s.failure)
		s.changed(ChangeStatus)
	}
}

func field(event *pyjson.Object, key string) any {
	value, _ := event.Get(key)
	return value
}

// logRequest adds one request a simulated device answered (or ignored) to the request log.
func (s *Service) logRequest(event *pyjson.Object) {
	entry := pyjson.NewObject()
	for _, key := range []string{"device", "peer", "version", "op", "count", "oid", "result", "held"} {
		entry.Set(key, field(event, key))
	}
	entry.Set("name", s.MIBs.Name(pytext.Str(field(event, "oid"))))
	s.Requests.Add(entry)
}

// logWrite records values a manager changed with SET.
func (s *Service) logWrite(event *pyjson.Object) {
	points, _ := field(event, "points").([]any)
	for _, item := range points {
		point, _ := item.(*pyjson.Object)
		if point == nil {
			continue
		}
		oid := pytext.Str(field(point, "oid"))
		s.Requests.Add(pyjson.NewObject().Set("device", field(event, "device")).Set("peer", field(event, "peer")).
			Set("version", "").Set("op", "written").Set("count", int64(1)).Set("oid", oid).
			Set("result", "now "+pytext.Str(field(point, "value"))).Set("held", int64(0)).Set("name", s.MIBs.Name(oid)))
	}
	s.changed(ChangeValues)
}

// logTrap adds a received trap or inform with the names of its notification and values.
func (s *Service) logTrap(event *pyjson.Object) {
	entry := pyjson.NewObject()
	for _, key := range []string{"from", "version", "kind", "community", "user", "notification", "uptime",
		"agent", "problem"} {
		if value, found := event.Get(key); found {
			entry.Set(key, value)
		}
	}
	notification := pytext.Str(field(event, "notification"))
	entry.Set("notification_name", s.MIBs.Name(notification))
	rows, _ := field(event, "varbinds").([]any)
	entry.Set("varbinds", s.MIBs.Decorate(rows))
	s.Traps.Add(entry)
}
