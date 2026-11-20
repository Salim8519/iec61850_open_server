package service

import (
	"net"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/worker"
)

// Watching the listen address (change 0221, trial N8): a gateway that listens on one address of
// this PC checks every watchEvery that Windows still has that address. When it is gone (taken off
// its card, the card removed) the gateway keeps running but SCADA cannot reach it, so the page
// says "Address gone" until the address is back. Nothing is watched for all networks or 127.0.0.1.

// watchEvery is how often a running gateway asks Windows for its address (one table read).
var watchEvery = 2 * time.Second

// addressWatch is one running watch; close stops it and waits for it.
type addressWatch struct {
	stop, done chan struct{}
}

func (w *addressWatch) close() {
	if w != nil {
		close(w.stop)
		<-w.done
	}
}

// watched says whether a bind address can disappear: one IPv4 address of this PC.
func watched(bind string) bool {
	address := net.ParseIP(bind).To4()
	return address != nil && !address.IsUnspecified() && !address.Equal(net.IPv4(127, 0, 0, 1))
}

// watchAddress starts watching bind for the gateway process; nil when there is nothing to watch.
func (m *Manager) watchAddress(process worker.Process, bind, endpoint string) *addressWatch {
	if m.HasAddress == nil || !watched(bind) {
		return nil
	}
	watch := &addressWatch{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(watch.done)
		ticker := time.NewTicker(watchEvery)
		defer ticker.Stop()
		present := true
		for {
			select {
			case <-watch.stop:
				return
			case <-ticker.C:
			}
			now, err := m.HasAddress(bind)
			if err == nil && now != present {
				present = now
				m.addressChanged(process, bind, endpoint, now)
			}
		}
	}()
	return watch
}

// addressChanged shows that the address went or came back, while this process still runs.
func (m *Manager) addressChanged(process worker.Process, bind, endpoint string, present bool) {
	text := "Listening on " + endpoint
	if !present {
		text = "Address gone: " + bind + " is no longer on this PC (it was taken off its network card), so SCADA " +
			"cannot connect to the gateway. "
		if isOwn(m.Config(), bind) {
			text += "Press Stop, then Start: ScadaBench puts the address back on the lab card."
		} else {
			text += "Give the card its address back in Windows, then press Stop and Start."
		}
	}
	m.mu.Lock()
	if m.process != process || m.state != "running" {
		m.mu.Unlock()
		return
	}
	m.message, m.gone = text, !present
	m.mu.Unlock()
	line := "Address " + bind + " is back on this PC"
	kind := "gateway_address_back"
	if !present {
		line, kind = "Address "+bind+" is gone from this PC; SCADA cannot connect", "gateway_address_gone"
	}
	m.Traffic.Add(pyjson.NewObject().Set("dir", "info").Set("peer", "").Set("text", line))
	m.emit(kind, pyjson.NewObject().Set("address", bind).Set("message", text))
	m.changed(ChangeGatewayStatus)
}
