package runtime

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/native"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// probeTimeout is the 15 s subprocess timeout of runtime/mms_client.operate.
const probeTimeout = 15 * time.Second

// errProbeTimeout stands for subprocess.TimeoutExpired, which Flask answered with its generic
// 500 (it is deliberately not an apperr).
var errProbeTimeout = fmt.Errorf("Command '[studio_probe]' timed out after 15 seconds")

// commandClient sends canvas commands through one warm studio_control process in its
// multiplexed mode (control_mux_v1) instead of one studio_probe process per click; it is
// started on first use and again after it died.
type commandClient struct {
	executable string
	keep       time.Duration
	mu         sync.Mutex
	process    *native.Control
	warming    sync.WaitGroup // a warm-up start close waits for
}

func newCommandClient(executable string, keep time.Duration) *commandClient {
	return &commandClient{executable: executable, keep: keep}
}

func (m *commandClient) client() (*native.Control, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.process != nil {
		if exited, _ := m.process.Exited(); !exited {
			return m.process, nil
		}
		m.process = nil
	}
	if _, err := os.Stat(m.executable); err != nil {
		return nil, apperr.RuntimeError("Bundled MMS client is unavailable")
	}
	process, err := native.StartControl(m.executable)
	if err != nil {
		return nil, apperr.RuntimeError("Bundled MMS client is unavailable")
	}
	hello, err := process.Hello()
	if err != nil || !native.HasCapabilities(hello, []string{native.MuxCapability}) {
		process.Close()
		return nil, apperr.RuntimeError("Bundled MMS client is unavailable")
	}
	m.process = process
	return process, nil
}

// warm starts the studio_control process in the background (Run start), so the first
// canvas command does not pay its start (~0.1-0.5 s). Nothing is shown or emitted: a start
// that fails here fails again, with its message, on the command that needs the process.
func (m *commandClient) warm() {
	m.warming.Add(1)
	go func() {
		defer m.warming.Done()
		_, _ = m.client()
	}()
}

// operate is runtime/mms_client.operate with Python's checks and texts; the result is the
// probe's {"ok": true}.
func (m *commandClient) operate(ied *pyjson.Object, state string, ref string, value any, host string, port int64) (*pyjson.Object, error) {
	flag, ok := value.(bool)
	if !ok {
		return nil, apperr.ValueError("Control value must be boolean")
	}
	known := false
	for _, entry := range pyval.List(ied, "controls") {
		known = known || pyval.Str(entry.(*pyjson.Object), "ref") == ref
	}
	if !known {
		return nil, apperr.ValueError("Unknown control object")
	}
	if state != "running" {
		return nil, apperr.ValueError("IED must be running to operate equipment")
	}
	process, err := m.client()
	if err != nil {
		return nil, err
	}
	result, err := process.CanvasOperate(host, port, ref, flag, m.keep, probeTimeout)
	failed := apperr.RuntimeError(fmt.Sprintf("MMS control failed or command termination was not received from %s:%d", host, port))
	if err != nil {
		var timeout *native.TimeoutError
		if errors.As(err, &timeout) && timeout.Op == "canvas_operate" {
			return nil, errProbeTimeout
		}
		return nil, failed
	}
	if okValue, _ := result.Get("ok"); okValue != true {
		return nil, failed
	}
	return result, nil
}

func (m *commandClient) close() {
	m.warming.Wait()
	m.mu.Lock()
	process := m.process
	m.process = nil
	m.mu.Unlock()
	if process != nil {
		process.Close()
	}
}
