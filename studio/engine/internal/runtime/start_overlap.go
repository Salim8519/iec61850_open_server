package runtime

import "studio-engine/internal/pyjson"

// The lab addresses and the GOOSE bus of a fresh Run are prepared together (ADR-0133). Each
// IED's model still starts only after both are ready, so the request each worker receives is
// the one it received before.

// busOpening is the GOOSE bus of a batch: opened at once for a fresh Run, or only when asked
// for an IED added to a running simulation (a refused addition must not register publishers
// in the running GOOSE engine).
type busOpening struct {
	c     *Coordinator
	done  chan struct{}
	buses map[string]string
	err   error
}

func (c *Coordinator) openBus(ieds []*pyjson.Object, early bool) *busOpening {
	opening := &busOpening{c: c}
	if early {
		opening.done = make(chan struct{})
		go func() {
			defer close(opening.done)
			opening.buses, opening.err = c.opt.Live.BusFor(ieds)
		}()
	}
	return opening
}

// wait lets an early opening finish before the batch gives up (a failed Run stops the engine).
func (b *busOpening) wait() {
	if b.done != nil {
		<-b.done
	}
}

// result is the bus of every publishing IED: the early answer, or BusFor now.
func (b *busOpening) result(ieds []*pyjson.Object) (map[string]string, error) {
	if b.done == nil {
		return b.c.opt.Live.BusFor(ieds)
	}
	<-b.done
	return b.buses, b.err
}
