package service

import (
	"strconv"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// event handles studio_master's unsolicited frames (values, link, probe, traffic).
func (m *Master) event(event *pyjson.Object) {
	switch get(event, "event") {
	case "master_values":
		values, _ := get(event, "values").([]any)
		m.tableMu.Lock()
		gateway.Merge(m.table, values) // a bad batch fails like Python's logged consumer error
		m.tableMu.Unlock()
		if m.OnValues != nil {
			m.OnValues()
		}
	case "master_link":
		m.mu.Lock()
		state := get(event, "state")
		if state == "open" {
			m.linkOpen = true
		} else if state == "closed" && m.process != nil && get(event, "reason") != "Disconnected by you" {
			reason, found := event.Get("reason")
			if !found {
				reason = "The link closed"
			}
			m.linkOpen, m.state = false, "lost"
			m.message = pytext.Str(reason) + ". Trying again every " + strconv.Itoa(RetrySeconds) + " s."
		}
		m.mu.Unlock()
	case "master_probe":
		m.Probe.progress(event)
	case "master_interrogation":
		line := "All values received"
		if state := get(event, "state"); state != "finished" {
			word, found := event.Get("state")
			if !found {
				word = "failed"
			}
			line = "Asking for all values " + pytext.Str(word)
		}
		m.Traffic.Add(info(line))
	case "gw_traffic":
		line := pytext.Str(get(event, "text"))
		if skipped := get(event, "skipped_before"); pytext.Truthy(skipped) {
			line += " (" + pytext.Str(skipped) + " lines skipped)"
		}
		m.Traffic.Add(pyjson.NewObject().Set("dir", orEmpty(event, "dir")).Set("peer", orEmpty(event, "peer")).
			Set("text", line))
	case "worker_failed":
		m.mu.Lock()
		m.linkOpen, m.state = false, "failed"
		m.message = "The master program stopped unexpectedly. Press Connect again."
		stopping := m.stopping
		m.mu.Unlock()
		closeChannel(stopping)
	}
}
