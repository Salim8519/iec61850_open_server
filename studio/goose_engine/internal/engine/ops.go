package engine

import (
	"encoding/json"

	"studio-goose-engine/internal/ipc"
)

// Version and Capabilities let the backend refuse a stale engine binary.
const Version = "1"

var Capabilities = []string{"goose_engine_v1"}

// Handlers maps backend operations onto the engine.
func Handlers(e *Engine) map[string]ipc.Handler {
	return map[string]ipc.Handler{
		"hello": func(json.RawMessage) (any, error) {
			return map[string]any{"version": Version, "capabilities": Capabilities}, nil
		},
		"open": func(raw json.RawMessage) (any, error) {
			var request struct {
				IEDs []IED `json:"ieds"`
			}
			if err := json.Unmarshal(raw, &request); err != nil {
				return nil, err
			}
			return e.Open(request.IEDs)
		},
		"deliver": func(raw json.RawMessage) (any, error) {
			var request struct {
				Port int `json:"port"`
			}
			if err := json.Unmarshal(raw, &request); err != nil {
				return nil, err
			}
			return true, e.Deliver(request.Port)
		},
		"uplink": func(raw json.RawMessage) (any, error) {
			var request struct {
				AdapterGUID string `json:"adapter_guid"`
				Share       bool   `json:"share"`
			}
			if err := json.Unmarshal(raw, &request); err != nil {
				return nil, err
			}
			return e.SetUplink(request.AdapterGUID, request.Share), nil
		},
		"gate": func(raw json.RawMessage) (any, error) {
			var request struct {
				IED    string `json:"ied_id"`
				Choice string `json:"choice"`
			}
			if err := json.Unmarshal(raw, &request); err != nil {
				return nil, err
			}
			gate, err := e.Gate(request.IED, request.Choice)
			return map[string]any{"ied_id": request.IED, "gate": gate}, err
		},
		"stats": func(json.RawMessage) (any, error) { return e.Stats(), nil },
	}
}
