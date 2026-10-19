package native

import (
	"time"

	"studio-engine/internal/pyjson"
)

// Live is a typed client for studio_live.exe (read-only MMS client, GOOSE and SV listener).
// Timeouts are the Python callers' (processes-2.md §1); one process per role, because the
// native loop answers one request at a time.
type Live struct{ *Worker }

// LiveRequired are the capabilities a real-IED session needs (acquisition/real_session.py).
var LiveRequired = []string{"live_discover", "live_reports", "live_poll", "read_only_client"}

// StartLive launches studio_live.exe.
func StartLive(executable string, onEvent EventFunc) (*Live, error) {
	worker, err := Start(executable, onEvent)
	if err != nil {
		return nil, err
	}
	return &Live{worker}, nil
}

// Hello returns {capabilities}.
func (c *Live) Hello() (*pyjson.Object, error) {
	return objectResult(c.call("hello", nil, DefaultTimeout))
}

// Connect opens the session's association (15 s, ip and port only, as real_session.py).
func (c *Live) Connect(ip string, port int64) (*pyjson.Object, error) {
	return objectResult(c.call("connect", pyjson.NewObject().Set("ip", ip).Set("port", port), 15*time.Second))
}

// Discover browses the device model (120 s).
func (c *Live) Discover() (*pyjson.Object, error) {
	return objectResult(c.call("discover", nil, 120*time.Second))
}

// Read reads [{ref, fc}] (at most 2000; 30 s).
func (c *Live) Read(refs []any) (*pyjson.Object, error) {
	return objectResult(c.call("read", pyjson.NewObject().Set("refs", refs), 30*time.Second))
}

// RcbStatus reads report control blocks (30 s).
func (c *Live) RcbStatus(rcbs []any) (*pyjson.Object, error) {
	return objectResult(c.call("rcb_status", pyjson.NewObject().Set("rcbs", rcbs), 30*time.Second))
}

// RcbReserve reserves and enables one report block (30 s).
func (c *Live) RcbReserve(rcb string) (*pyjson.Object, error) {
	return objectResult(c.call("rcb_reserve", pyjson.NewObject().Set("rcb", rcb), 30*time.Second))
}

// Disconnect gives every report block back and closes (15 s).
func (c *Live) Disconnect() (any, error) {
	return c.call("disconnect", nil, 15*time.Second)
}

// GooseStart opens the loopback GOOSE inbox for subscriptions (30 s).
func (c *Live) GooseStart(subscriptions []any) (*pyjson.Object, error) {
	return objectResult(c.call("goose_start", pyjson.NewObject().Set("subscriptions", subscriptions), 30*time.Second))
}

// SvStart listens for Sampled Values on an adapter (30 s).
func (c *Live) SvStart(adapterGUID string, streams []any) (*pyjson.Object, error) {
	return objectResult(c.call("sv_start", pyjson.NewObject().Set("adapter_guid", adapterGUID).
		Set("streams", streams), 30*time.Second))
}

// Call sends any other operation (explore, identify) with an explicit timeout.
func (c *Live) Call(op string, fields *pyjson.Object, timeout time.Duration) (any, error) {
	return c.call(op, fields, timeout)
}
