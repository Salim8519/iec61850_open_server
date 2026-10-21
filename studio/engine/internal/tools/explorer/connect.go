package explorer

import (
	"os"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Connect is ExplorerService.connect: one connect at a time; each starts from a worker
// with no association, then the key points are watched by themselves.
func (s *Service) Connect(host, port any) (*pyjson.Object, error) {
	address, checkedPort, err := CheckEndpoint(host, port)
	if err != nil {
		return nil, err
	}
	defer s.changed()
	s.connectMu.Lock()
	defer s.connectMu.Unlock()
	s.mu.Lock()
	same := s.state == "connected" && s.port != nil && s.host == address && pytext.Equal(s.port, checkedPort)
	s.mu.Unlock()
	if same && s.alive() {
		return s.Status(), nil
	}
	return s.connect(address, checkedPort)
}

func (s *Service) connect(host string, port any) (*pyjson.Object, error) {
	s.Disconnect()
	if _, err := os.Stat(s.executable); err != nil {
		return nil, apperr.RuntimeError("The live client (studio_live.exe) is not built yet; rebuild the native programs")
	}
	endpoint := host + ":" + pytext.Str(port)
	s.mu.Lock()
	s.state, s.message, s.host, s.port = "connecting", "Connecting to "+endpoint, host, port
	s.mu.Unlock()
	s.changed()
	process, err := s.worker()
	if err == nil {
		fields := pyjson.NewObject().Set("ip", host).Set("port", port).Set("timeout_ms", int64(5000))
		_, err = process.Request("connect", 15*time.Second, fields)
	}
	if err != nil {
		s.mu.Lock()
		s.state, s.message = "failed", err.Error()
		s.mu.Unlock()
		return nil, apperr.RuntimeError("Could not open an IEC 61850 association to " + endpoint +
			". Check the address, the cable and that the device allows another client.")
	}
	s.mu.Lock()
	s.state, s.message = "connected", "Connected to "+endpoint
	s.mu.Unlock()
	s.emit("explorer_connected", pyjson.NewObject().Set("host", host).Set("port", port))
	if s.memory != nil {
		status, err := s.startWatching()
		if err == nil {
			return status, nil
		}
		s.setMessage("Connected (read-only). Automatic watching did not start: " + err.Error())
	}
	return s.Status(), nil
}

// Disconnect ends the association; the worker stays warm for the next Connect.
func (s *Service) Disconnect() *pyjson.Object {
	defer s.changed()
	s.StopPolling()
	s.mu.Lock()
	s.cache, s.values = map[string]any{}, map[string]*pyjson.Object{}
	if s.state != "disconnected" {
		s.state, s.message = "disconnected", "Disconnected"
	}
	process := s.process
	s.mu.Unlock()
	// Always reset the worker: after a failed or overlapping connect it can still hold an association.
	if process != nil && process.Alive() {
		if _, err := process.Request("disconnect", 10*time.Second, nil); err != nil {
			s.Close()
		}
	}
	return s.Status()
}

// Close stops the worker (backend shutdown).
func (s *Service) Close() {
	s.mu.Lock()
	process := s.process
	s.process = nil
	s.mu.Unlock()
	if process != nil {
		process.Close()
	}
}
