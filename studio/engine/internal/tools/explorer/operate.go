package explorer

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
)

// Operate sends one confirmed command to the connected device and reads the position after.
func (s *Service) Operate(reference string, value, options any, confirmed bool) (*pyjson.Object, error) {
	if !confirmed {
		return nil, apperr.ValueError("Confirm the command first")
	}
	if s.control == nil {
		return nil, apperr.RuntimeError("Commands are not available in this copy of ScadaBench")
	}
	s.mu.Lock()
	state, host, port := s.state, s.host, s.port
	s.mu.Unlock()
	if state != "connected" || port == nil {
		return nil, apperr.ValueError("Connect to a device first")
	}
	result, err := s.control.Operate(host, port, reference, value, options)
	if err != nil {
		return nil, err
	}
	summary := control.Summary(result)
	ok, _ := result.Get("ok")
	s.emit("explorer_command", pyjson.NewObject().Set("host", host).Set("ref", reference).Set("value", value).
		Set("ok", ok).Set("result", summary))
	var observed any
	status := reference + ".stVal"
	if values, err := s.Read([]*pyjson.Object{pyjson.NewObject().Set("ref", status).Set("fc", "ST")}); err == nil {
		observed = get(values, status)
	}
	return pyjson.NewObject().Set("result", result).Set("summary", summary).Set("observed", observed), nil
}
