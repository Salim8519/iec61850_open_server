package scan

import (
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/numlog"
	"studio-engine/internal/tools/pytext"
)

// run probes every host (64 at a time, results taken in address order like the
// ThreadPoolExecutor loop), then asks every open port who it is.
func (s *Service) run(hosts []string, port int64, portValue any, stop, done chan struct{}) {
	defer close(done)
	defer func() {
		if failure := recover(); failure != nil {
			s.mu.Lock()
			s.state.Set("state", "failed").Set("message", "The scan stopped: "+pytext.Str(failure))
			s.mu.Unlock()
		}
	}()
	results := make([]chan probeResult, len(hosts))
	slots := make(chan struct{}, Parallel)
	for index, host := range hosts {
		results[index] = make(chan probeResult, 1)
		go func(host string, out chan probeResult) {
			slots <- struct{}{}
			defer func() { <-slots }()
			if stopped(stop) {
				out <- probeResult{}
				return
			}
			ms, open := s.probe(host, port)
			out <- probeResult{ms, open}
		}(host, results[index])
	}
	var open []string
	for index, host := range hosts {
		result := <-results[index]
		s.mu.Lock()
		probed, _ := s.state.Get("probed")
		s.state.Set("probed", probed.(int64)+1)
		if result.open {
			s.found = append(s.found, pyjson.NewObject().Set("ip", host).Set("port", portValue).
				Set("tcp_ms", result.ms).Set("identified", nil).Set("vendor", "").Set("model", "").
				Set("revision", "").Set("devices", []any{}).Set("reason", "Identifying\xe2\x80\xa6"))
		}
		s.mu.Unlock()
		if result.open {
			open = append(open, host)
		}
	}
	if !stopped(stop) && len(open) > 0 {
		s.identify(open, portValue, stop)
	}
	s.finish(stop)
}

type probeResult struct {
	ms   float64
	open bool
}

func (s *Service) identify(hosts []string, port any, stop chan struct{}) {
	s.mu.Lock()
	s.state.Set("message", "Asking "+strconv.Itoa(len(hosts))+" device(s) who they are")
	s.mu.Unlock()
	process, err := s.spawn(s.executable, nil)
	if err != nil {
		panic(err.Error())
	}
	defer process.Close()
	for _, host := range hosts {
		if stopped(stop) {
			break
		}
		fields := pyjson.NewObject().Set("ip", host).Set("port", port).Set("timeout_ms", int64(3000))
		reply, err := process.Request("identify", identifyWait, fields)
		answer, _ := reply.(*pyjson.Object)
		if err != nil {
			answer = pyjson.NewObject().Set("identified", false).Set("reason", err.Error())
		} else if answer == nil {
			answer = pyjson.NewObject()
		}
		s.update(host, answer)
	}
}

func (s *Service) update(host string, answer *pyjson.Object) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, entry := range s.found {
		if ip, _ := entry.Get("ip"); ip != host {
			continue
		}
		for _, key := range []string{"identified", "vendor", "model", "revision", "devices", "connect_ms"} {
			if value, found := answer.Get(key); found {
				entry.Set(key, value)
			}
		}
		var notes []string
		for _, key := range []string{"reason", "identity_note", "devices_note"} {
			if value, _ := answer.Get(key); pytext.Truthy(value) {
				notes = append(notes, pytext.Str(value))
			}
		}
		entry.Set("reason", strings.Join(notes, "; "))
	}
}

func (s *Service) finish(stop chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	started, _ := s.state.Get("started")
	s.state.Set("seconds", round1(numlog.Now()-started.(float64)))
	named := 0
	for _, entry := range s.found {
		if value, _ := entry.Get("identified"); pytext.Truthy(value) {
			named++
		}
	}
	count := strconv.Itoa(len(s.found))
	if stopped(stop) {
		s.state.Set("state", "stopped").Set("message", "Stopped. "+count+" device(s) found so far.")
		return
	}
	message := "Done. No device answered on this port."
	if len(s.found) > 0 {
		port, _ := s.state.Get("port")
		message = "Done. " + count + " address(es) answer on port " + pytext.Str(port) + ", " +
			strconv.Itoa(named) + " identified as IEC 61850."
	}
	s.state.Set("state", "done").Set("message", message)
}
