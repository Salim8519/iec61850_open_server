package service

import (
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// event handles the frames studio_gateway sends by itself (on its reader, in order).
func (m *Manager) event(event *pyjson.Object) {
	switch get(event, "event") {
	case "gw_traffic":
		line := pytext.Str(get(event, "text"))
		if skipped := get(event, "skipped_before"); pytext.Truthy(skipped) {
			line += " (" + pytext.Str(skipped) + " lines skipped)"
		}
		m.Traffic.Add(pyjson.NewObject().Set("dir", orEmpty(event, "dir")).Set("peer", orEmpty(event, "peer")).
			Set("text", line))
	case "gw_client":
		peer := pytext.Str(get(event, "peer"))
		m.mu.Lock()
		if get(event, "state") == "open" {
			m.clients[peer] = true
		} else {
			delete(m.clients, peer)
		}
		m.mu.Unlock()
	case "gw_command":
		m.mu.Lock()
		process := m.process
		m.mu.Unlock()
		m.executor.submit(process, event)
	case "worker_failed":
		m.mu.Lock()
		m.state, m.message = "failed", "The gateway program stopped unexpectedly. Press Start again."
		sync := m.sync
		m.mu.Unlock()
		if sync != nil {
			sync.halt()
		}
	}
}

// orEmpty is event.get(key, "").
func orEmpty(event *pyjson.Object, key string) any {
	if value, found := event.Get(key); found {
		return value
	}
	return ""
}

// Stop halts the gateway; remember=false keeps auto_start for a stop caused by the app closing.
func (m *Manager) Stop(remember bool) (*pyjson.Object, error) {
	if config := m.Config(); remember && pytext.Truthy(get(config, "auto_start")) {
		if err := m.projects.SaveGateway(with(config, "auto_start", false)); err != nil {
			return nil, err
		}
	}
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	m.mu.Lock()
	process, sync := m.process, m.sync
	m.process, m.sync = nil, nil
	m.mu.Unlock()
	if sync != nil {
		sync.close()
	}
	if process != nil {
		_, err := process.Request("halt", 10*time.Second, nil)
		process.Close()
		if err != nil {
			return nil, err // Python lets the halt error through and keeps the old state
		}
		m.Traffic.Add(pyjson.NewObject().Set("dir", "info").Set("peer", "").Set("text", "Gateway stopped"))
		m.emit("gateway_stopped", pyjson.NewObject())
	}
	m.mu.Lock()
	m.state, m.message, m.clients = "stopped", "", map[string]bool{}
	m.mu.Unlock()
	return m.Status(), nil
}

// Send runs a command typed on the page like a SCADA command and logs it the same way.
func (m *Manager) Send(pointID string, value bool) *pyjson.Object {
	ok, reason := m.executor.run(pyjson.NewObject().Set("point_id", pointID).Set("value", value).
		Set("protocol", "manual").Set("peer", "this page"))
	return pyjson.NewObject().Set("ok", ok).Set("reason", reason)
}

// MarkRemote tells a running converter that the master's device values changed.
func (m *Manager) MarkRemote() {
	m.mu.Lock()
	sync := m.sync
	m.mu.Unlock()
	if sync != nil {
		sync.markRemote()
	}
}
