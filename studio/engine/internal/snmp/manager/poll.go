package manager

import (
	"sync"
	"time"

	"studio-engine/internal/snmp/proto"
)

// pollChunk is how many values one poll request asks for.
const pollChunk = 20

// Poller re-reads chosen OIDs at a fixed interval and reports only what changed.
type Poller struct {
	stop chan struct{}
	done sync.WaitGroup
}

// Changed receives the values that changed since the last round (every value on the first
// round) and the round's problem ("" when the device answered).
type Changed func(values []any, problem string)

// StartPoller polls oids every interval until Stop. A failed round is reported once and
// retried at the next interval.
func StartPoller(s *Session, oids []proto.OID, every time.Duration, changed Changed) *Poller {
	p := &Poller{stop: make(chan struct{})}
	p.done.Add(1)
	go func() {
		defer p.done.Done()
		last := map[string]string{}
		problem := ""
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			binds, err := s.GetMany(oids, pollChunk)
			text := ""
			if err != nil {
				text = err.Error()
			}
			fresh := []proto.VarBind{}
			for _, bind := range binds {
				kind, value, _ := proto.ValueText(bind.Value)
				key := bind.OID.String()
				if last[key] != kind+"\x00"+value {
					last[key] = kind + "\x00" + value
					fresh = append(fresh, bind)
				}
			}
			if len(fresh) > 0 || text != problem {
				problem = text
				changed(Points(fresh), text)
			}
			select {
			case <-p.stop:
				return
			case <-ticker.C:
			}
		}
	}()
	return p
}

// Stop ends polling and waits for the current round.
func (p *Poller) Stop() {
	close(p.stop)
	p.done.Wait()
}
