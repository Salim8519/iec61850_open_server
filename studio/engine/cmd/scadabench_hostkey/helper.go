package main

import (
	"context"
	"sync"
	"time"

	"studio-engine/internal/licensing/hostkey"
)

// helper is the host helper's behaviour with every outside world behind a function, so each rule is
// tested without Windows: it serves only while sharing is switched on, only while this PC's license,
// machine ID and key agree, and only on the adapters of virtual switches.
type helper struct {
	responder hostkey.Responder
	// check says whether this PC may serve at all (nil error) or why not, in plain words.
	check func() error
	// adapters lists this PC's adapters.
	adapters func() ([]hostkey.Adapter, error)
	// wanted is whether "Let my virtual machines use this license" is still on.
	wanted func() bool
	// extra adds endpoints beyond the adapters; only the test build (seam_test.go) sets it.
	extra func() []hostkey.Endpoint
	logf  func(string, ...any)

	pollEvery time.Duration // how often "wanted" is read (default 5 s)
	rescan    time.Duration // how often adapters and the license are read again (default 10 s)

	mu       sync.Mutex
	lastNote string
}

// note logs a refusal once, not on every rescan.
func (h *helper) note(text string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.lastNote != text {
		h.lastNote = text
		h.logf("%s", text)
	}
}

// endpoints is what to serve on right now: nothing unless the license allows, else the addresses
// of the virtual-switch adapters (plus the test seam's, in the test build only).
func (h *helper) endpoints() []hostkey.Endpoint {
	if err := h.check(); err != nil {
		h.note("Not serving virtual machines: " + err.Error())
		return nil
	}
	found := []hostkey.Endpoint{}
	adapters, err := h.adapters()
	if err != nil {
		h.note("Not serving virtual machines: the adapters could not be read: " + err.Error())
	}
	for _, listen := range hostkey.ListenAddresses(adapters) {
		subnet := listen.Subnet
		found = append(found, hostkey.Endpoint{Addr: hostkey.ListenerAddress(listen.IP, hostkey.Port), Subnet: &subnet})
	}
	if h.extra != nil {
		found = append(found, h.extra()...)
	}
	if len(found) == 0 && err == nil {
		h.note("This PC has no virtual switch to serve yet.")
	} else if len(found) > 0 {
		h.note("Serving virtual machines.")
	}
	return found
}

// run serves until ctx ends or sharing is switched off, then returns.
func (h *helper) run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	poll := h.pollEvery
	if poll <= 0 {
		poll = 5 * time.Second
	}
	go func() {
		ticker := time.NewTicker(poll)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !h.wanted() {
					h.logf("Sharing was switched off; the helper ends.")
					cancel()
					return
				}
			}
		}
	}()
	server := &hostkey.Server{Responder: h.responder, Listeners: h.endpoints, Limiter: hostkey.DefaultLimiter(),
		Rescan: h.rescan, Logf: h.logf}
	if h.rescan <= 0 {
		server.Rescan = 10 * time.Second
	}
	server.Run(ctx)
}
