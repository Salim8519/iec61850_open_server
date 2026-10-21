package netcfg

import (
	"sync"
	"time"

	"studio-engine/internal/pyjson"
)

// Live progress of the running address check (change 0150, ADR-0093). Network setup shows it
// as a step list while POST /network/plan/preview runs: which step is running, how many
// addresses have answered, and each row's address turning free or in use as answers arrive.
// It only reports; the preview itself and its result are unchanged.

// Preview steps in the order they run.
var previewSteps = []struct{ key, label string }{
	{"cards", "Read the network card"},
	{"addresses", "Check each address on the network"},
	{"firewall", "Read Windows Firewall rules"},
	{"plan", "Prepare the exact Windows changes"},
}

type stepTiming struct {
	state   string // waiting, running, done
	started time.Time
	took    time.Duration
}

type plannedRow struct{ id, name, address string }

// PreviewProgress is the live state of the latest address check.
type PreviewProgress struct {
	mu        sync.Mutex
	run       int64
	running   bool
	steps     map[string]*stepTiming
	planned   []plannedRow
	addresses map[string]string // checking, free, in_use
	asked     int
	answered  int
}

func (p *PreviewProgress) begin() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.run++
	p.running, p.planned, p.asked, p.answered = true, nil, 0, 0
	p.steps, p.addresses = map[string]*stepTiming{}, map[string]string{}
	for _, step := range previewSteps {
		p.steps[step.key] = &stepTiming{state: "waiting"}
	}
}

// step marks key running and every earlier step done.
func (p *PreviewProgress) step(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.steps == nil {
		return
	}
	now := time.Now()
	for _, step := range previewSteps {
		timing := p.steps[step.key]
		if step.key == key {
			if timing.state == "waiting" {
				timing.state, timing.started = "running", now
			}
			return
		}
		if timing.state == "running" {
			timing.state, timing.took = "done", now.Sub(timing.started)
		} else if timing.state == "waiting" {
			timing.state = "done"
		}
	}
}

func (p *PreviewProgress) finish() {
	p.step("") // no step has this key, so every step ends
	p.mu.Lock()
	p.running = false
	p.mu.Unlock()
}

// plan records the rows being checked; a later round replaces a row's address.
func (p *PreviewProgress) plan(rows []Row) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, row := range rows {
		entry := plannedRow{id: row.IEDID, name: row.Name, address: row.Proposed}
		replaced := false
		for index := range p.planned {
			if p.planned[index].id == row.IEDID {
				p.planned[index], replaced = entry, true
			}
		}
		if !replaced {
			p.planned = append(p.planned, entry)
		}
	}
}

func (p *PreviewProgress) asking(addresses []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, address := range addresses {
		if _, known := p.addresses[address]; !known {
			p.asked++
		}
		p.addresses[address] = "checking"
	}
}

func (p *PreviewProgress) answer(address string, used bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.addresses[address] == "checking" {
		p.answered++
	}
	p.addresses[address] = map[bool]string{true: "in_use", false: "free"}[used]
}

// Object is the progress as JSON.
func (p *PreviewProgress) Object() *pyjson.Object {
	p.mu.Lock()
	defer p.mu.Unlock()
	steps := []any{}
	for _, step := range previewSteps {
		state, took := "waiting", int64(0)
		if timing := p.steps[step.key]; timing != nil {
			state, took = timing.state, timing.took.Milliseconds()
			if state == "running" {
				took = time.Since(timing.started).Milliseconds()
			}
		}
		steps = append(steps, pyjson.NewObject().Set("key", step.key).Set("label", step.label).
			Set("state", state).Set("ms", took))
	}
	rows := []any{}
	for _, row := range p.planned {
		state := p.addresses[row.address]
		if state == "" {
			state = "not_checked"
		}
		rows = append(rows, pyjson.NewObject().Set("id", row.id).Set("name", row.name).
			Set("address", row.address).Set("state", state))
	}
	return pyjson.NewObject().Set("run", p.run).Set("running", p.running).Set("steps", steps).
		Set("asked", int64(p.asked)).Set("answered", int64(p.answered)).Set("rows", rows)
}

// PreviewProgress is GET /network/plan/progress.
func (s *NetworkService) PreviewProgress() *pyjson.Object { return s.Progress.Object() }

// reportingProber passes answers to the progress as each one arrives.
type reportingProber struct {
	inner    Prober
	progress *PreviewProgress
}

func (r reportingProber) InUse(addresses []string) (map[string]bool, error) {
	r.progress.asking(addresses)
	if arp, ok := r.inner.(ARPProber); ok {
		arp.Report = r.progress.answer
		return arp.InUse(addresses)
	}
	used, err := r.inner.InUse(addresses)
	if err == nil {
		for _, address := range addresses {
			r.progress.answer(address, used[address])
		}
	}
	return used, err
}
