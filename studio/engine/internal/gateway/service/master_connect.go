package service

import (
	"os"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/worker"
)

var notConnected = apperr.ValueError("Not connected to the device; press Connect first")

// connectTimeout is t0 + 10 s (t0 only exists for IEC 104; 10 otherwise).
func connectTimeout(settings *pyjson.Object) time.Duration {
	t0 := int64(10)
	if value, found := settings.Get("t0"); found {
		t0, _ = pytext.Int64(value)
	}
	return time.Duration(t0+10) * time.Second
}

// Connect starts studio_master toward the configured device and asks for every value.
func (m *Master) Connect() (*pyjson.Object, error) {
	defer m.changed(ChangeMasterPoints)
	defer m.changed(ChangeMasterStatus)
	m.lifecycle.Lock()
	config, protocol, host, port, err := m.open()
	m.lifecycle.Unlock()
	if err != nil {
		return nil, err
	}
	if !pytext.Truthy(get(config, "auto_connect")) {
		if err := m.Save(with(config, "auto_connect", true)); err != nil {
			return nil, err
		}
	}
	m.emit("master_connected", pyjson.NewObject().Set("protocol", protocol).Set("host", host).Set("port", port))
	m.askAll()
	return m.Status(), nil
}

func (m *Master) open() (*pyjson.Object, string, string, any, error) {
	if m.Busy() {
		return nil, "", "", nil, apperr.ValueError("Already connected; press Disconnect first")
	}
	if role := get(m.gatewaySection(), "role"); role != "master" && role != "converter" {
		return nil, "", "", nil, apperr.ValueError("Switch the tool to SCADA master first")
	}
	if _, err := os.Stat(m.Executable); err != nil {
		return nil, "", "", nil, apperr.RuntimeError("The master program (studio_master.exe) is not built yet; rebuild the native programs")
	}
	config := m.Config()
	protocol, stored := settingsOf(config)
	host := text(config, "host")
	if host == "" {
		return nil, "", "", nil, apperr.ValueError("Type the address of the gateway, RTU or outstation first")
	}
	settings, err := gateway.ValidateMasterSettings(protocol, stored)
	if err != nil {
		return nil, "", "", nil, err
	}
	process, err := m.spawn(m.Executable, m.event)
	if err != nil {
		return nil, "", "", nil, err
	}
	if err := m.handshake(process, protocol, host, settings); err != nil {
		process.Close()
		return nil, "", "", nil, err
	}
	port := get(settings, "port")
	device := []any{protocol, host, port}
	m.tableMu.Lock()
	if m.device == nil || !pytext.Equal(device, m.device) {
		m.table, m.device = pyjson.NewObject(), device
	}
	m.tableMu.Unlock()
	endpoint := host + ":" + pytext.Str(port)
	stopping := make(chan struct{})
	m.mu.Lock()
	m.process, m.linkOpen, m.state, m.message, m.stopping = process, true, "connected", "Connected to "+endpoint, stopping
	m.mu.Unlock()
	m.Traffic.Add(info(protocolNames[protocol] + " master connected to " + endpoint))
	go m.keep(stopping)
	return config, protocol, host, port, nil
}

func (m *Master) handshake(process worker.Process, protocol, host string, settings *pyjson.Object) error {
	hello, err := process.Request("hello", worker.DefaultTimeout, nil)
	if err != nil {
		return err
	}
	if !control.HasCapability(hello, capability[protocol]) {
		return apperr.RuntimeError("ScadaBench cannot act as " + protocolNames[protocol] + " master in this build yet")
	}
	_, err = process.Request("connect", connectTimeout(settings), pyjson.NewObject().Set("protocol", protocol).
		Set("settings", with(settings, "host", host)))
	return err
}

func info(line string) *pyjson.Object {
	return pyjson.NewObject().Set("dir", "info").Set("peer", "").Set("text", line)
}

// Autoconnect connects again after the app restarts if it was connected when it closed.
func (m *Master) Autoconnect() {
	if pytext.Truthy(get(m.Config(), "auto_connect")) && !m.Busy() {
		if _, err := m.Connect(); err != nil {
			m.mu.Lock()
			m.state, m.message = "failed", "Automatic connect failed: "+err.Error()
			m.mu.Unlock()
			m.changed(ChangeMasterStatus)
		}
	}
}

// keep repeats the interrogation or Modbus reads on the user's interval and retries a dropped link.
func (m *Master) keep(stopping chan struct{}) {
	for {
		config := m.Config()
		protocol, settings := settingsOf(config)
		wait := time.Second
		if protocol == "modbus" {
			ms, _ := pytext.Float(get(settings, "poll_ms"))
			wait = time.Duration(ms * float64(time.Millisecond))
		}
		if sleep(stopping, wait) {
			return
		}
		minutes := get(settings, "gi_minutes")
		if _, found := settings.Get("gi_minutes"); !found {
			minutes = get(settings, "integrity_minutes")
		}
		m.mu.Lock()
		open, state, last := m.linkOpen, m.state, m.lastAsked
		m.mu.Unlock()
		switch {
		case !open:
			if state == "lost" && !sleep(stopping, RetrySeconds*time.Second) {
				m.reconnect(config)
			}
		case protocol == "modbus":
			if !m.Probe.Running() {
				m.askAll()
			}
		case pytext.Truthy(minutes):
			asked, _ := pytext.Float(last)
			span, _ := pytext.Float(minutes)
			if nowSeconds()-asked >= span*60 {
				m.askAll()
			}
		}
	}
}

// sleep waits for d; true when stopping was signalled.
func sleep(stopping chan struct{}, d time.Duration) bool {
	select {
	case <-stopping:
		return true
	case <-time.After(d):
		return false
	}
}

func (m *Master) reconnect(config *pyjson.Object) {
	defer m.changed(ChangeMasterPoints)
	defer m.changed(ChangeMasterStatus)
	m.mu.Lock()
	process := m.process
	m.mu.Unlock()
	if process == nil {
		return
	}
	protocol, stored := settingsOf(config)
	settings, err := gateway.ValidateMasterSettings(protocol, stored)
	if err == nil {
		_, err = process.Request("disconnect", 10*time.Second, nil)
	}
	if err == nil {
		_, err = process.Request("connect", connectTimeout(settings), pyjson.NewObject().Set("protocol", protocol).
			Set("settings", with(settings, "host", get(config, "host"))))
	}
	if err != nil {
		m.setMessage("Still trying to reach the device: " + err.Error())
		return
	}
	m.mu.Lock()
	m.linkOpen, m.state, m.message = true, "connected", "Connected again"
	m.mu.Unlock()
	m.askAll()
}

// Disconnect stops the worker; remember=false keeps auto_connect for an app shutdown.
func (m *Master) Disconnect(remember bool) (*pyjson.Object, error) {
	defer m.changed(ChangeMasterPoints)
	defer m.changed(ChangeMasterStatus)
	m.cyclic.Stop() // never keep sending once the link is going away
	if config := m.Config(); remember && pytext.Truthy(get(config, "auto_connect")) {
		if err := m.Save(with(config, "auto_connect", false)); err != nil {
			return nil, err
		}
	}
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	m.mu.Lock()
	process, stopping := m.process, m.stopping
	m.process = nil
	m.mu.Unlock()
	closeChannel(stopping)
	if process != nil {
		process.Request("stop", 10*time.Second, nil)
		process.Close()
		m.Traffic.Add(info("Disconnected"))
		m.emit("master_disconnected", pyjson.NewObject())
	}
	m.mu.Lock()
	m.state, m.message, m.linkOpen = "disconnected", "", false
	m.mu.Unlock()
	return m.Status(), nil
}

func closeChannel(stop chan struct{}) {
	defer func() { recover() }()
	close(stop)
}
