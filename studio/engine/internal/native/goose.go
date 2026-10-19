package native

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Goose texts from acquisition/goose_engine.py.
const (
	GooseMissing = "The GOOSE engine (studio_goose.exe) is not built yet; run studio/tools/build_goose_engine.py"
	GooseStale   = "The GOOSE engine is outdated. Rebuild it with studio/tools/build_goose_engine.py before Run"
)

// Goose is a typed client for studio_goose.exe (ADR-0073); every call uses the 10 s default.
type Goose struct{ *Worker }

// StartGoose launches the engine and checks goose_engine_v1, closing an outdated binary.
func StartGoose(executable string, onEvent EventFunc) (*Goose, error) {
	worker, err := Start(executable, onEvent)
	if err != nil {
		return nil, err
	}
	client := &Goose{worker}
	hello, err := objectResult(client.Request("hello", nil, DefaultTimeout))
	if err != nil {
		return nil, err
	}
	if !HasCapabilities(hello, []string{"goose_engine_v1"}) {
		client.Close()
		return nil, apperr.ValueError(GooseStale)
	}
	return client, nil
}

// Open registers publishing IEDs [{id, gocb_refs}]: {port, sources:{id: MAC}}.
func (c *Goose) Open(ieds []any) (*pyjson.Object, error) {
	return objectResult(c.Request("open", pyjson.NewObject().Set("ieds", ieds), DefaultTimeout))
}

// Deliver points the engine at studio_live's GOOSE inbox port.
func (c *Goose) Deliver(port int64) error {
	_, err := c.Request("deliver", pyjson.NewObject().Set("port", port), DefaultTimeout)
	return err
}

// Uplink listens on an adapter ("" = none) and shares outside only when share is true.
func (c *Goose) Uplink(adapterGUID string, share bool) (*pyjson.Object, error) {
	return objectResult(c.Request("uplink", pyjson.NewObject().Set("adapter_guid", adapterGUID).
		Set("share", share), DefaultTimeout))
}

// Gate answers a GOOSE conflict hold for one IED (stop or continue).
func (c *Goose) Gate(iedID, choice string) (*pyjson.Object, error) {
	return objectResult(c.Request("gate", pyjson.NewObject().Set("ied_id", iedID).Set("choice", choice), DefaultTimeout))
}

// Stats returns the engine counters.
func (c *Goose) Stats() (*pyjson.Object, error) {
	return objectResult(c.Request("stats", nil, DefaultTimeout))
}
