package runtime

import (
	"os"
	"sync"
	"time"

	"studio-engine/internal/native"
	"studio-engine/internal/pyjson"
)

// Spare IED processes (ADR-0142). Each Run started one studio_ied per simulated IED, and each
// process did its own start-up and licence check (the packaged build also checks the program's
// signature) in the middle of the busy Run. While nothing runs, the coordinator keeps one spare
// process per simulated IED of the project, started and greeted ("hello" runs the licence check)
// one after another; a Run takes them instead of starting processes. A spare holds no model and
// no address: about 1 MB of private memory and no processor time while it waits. It is a new
// process each Run, as before; only the moment differs. A spare is never used when the program
// file changed after it started, and every spare is closed at shutdown.

// spareIed is one waiting process; route delivers its events to the IED that takes it.
type spareIed struct {
	worker *native.Ied
	route  *eventRoute
}

// eventRoute forwards a worker's events once the process belongs to an IED (none before).
type eventRoute struct {
	mu      sync.Mutex
	deliver func(*pyjson.Object)
}

func (r *eventRoute) event(event *pyjson.Object) {
	r.mu.Lock()
	deliver := r.deliver
	r.mu.Unlock()
	if deliver != nil {
		deliver(event)
	}
}

func (r *eventRoute) set(deliver func(*pyjson.Object)) {
	r.mu.Lock()
	r.deliver = deliver
	r.mu.Unlock()
}

type spareIeds struct {
	executable string
	mu         sync.Mutex
	ready      []*spareIed
	stamp      fileStamp
	filling    chan struct{} // open while spares are being started; nil otherwise
	closed     bool
}

// fileStamp identifies the program file a spare was started from.
type fileStamp struct {
	size     int64
	modified time.Time
}

func stampOf(path string) (fileStamp, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return fileStamp{}, false
	}
	return fileStamp{info.Size(), info.ModTime()}, true
}

func newSpareIeds(executable string) *spareIeds { return &spareIeds{executable: executable} }

// PrepareSpares readies one spare process per simulated IED of the current project while the
// simulation is stopped: after Stop, after a failed Run and once the project opened or changed.
func (c *Coordinator) PrepareSpares() {
	c.mu.Lock()
	stopped := c.state == "stopped"
	c.mu.Unlock()
	if !stopped {
		return
	}
	count := 0
	for _, ied := range projectIeds(c.opt.Project.Current()) {
		if roleOf(ied) == "simulated" {
			count++
		}
	}
	c.spares.fill(count)
}

// fill starts spares in the background until want are waiting. A process that does not start
// or whose hello fails is closed: the Run then starts that IED's process as before.
func (s *spareIeds) fill(want int) {
	s.mu.Lock()
	if s.closed || s.filling != nil || len(s.ready) >= want {
		s.mu.Unlock()
		return
	}
	filling := make(chan struct{})
	s.filling = filling
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			s.filling = nil
			s.mu.Unlock()
			close(filling)
		}()
		for {
			s.mu.Lock()
			enough := s.closed || len(s.ready) >= want
			s.mu.Unlock()
			if enough {
				return
			}
			stamp, found := stampOf(s.executable)
			if !found {
				return
			}
			spare, err := s.start()
			if err != nil {
				return
			}
			s.mu.Lock()
			keep := !s.closed && (len(s.ready) == 0 || stamp == s.stamp)
			if keep {
				s.ready, s.stamp = append(s.ready, spare), stamp
			}
			s.mu.Unlock()
			if !keep {
				spare.worker.Close()
				return
			}
		}
	}()
}

func (s *spareIeds) start() (*spareIed, error) {
	route := &eventRoute{}
	worker, err := native.StartIed(s.executable, route.event)
	if err != nil {
		return nil, err
	}
	if _, err := worker.Request("hello", native.DefaultTimeout, nil); err != nil {
		worker.Close()
		return nil, err
	}
	return &spareIed{worker: worker, route: route}, nil
}

// take hands one waiting process to the IED (its events go to deliver), or nil when none is
// waiting or the program file changed since the spares started (they are closed then).
func (s *spareIeds) take(deliver func(*pyjson.Object)) *native.Ied {
	for {
		s.mu.Lock()
		if len(s.ready) == 0 {
			s.mu.Unlock()
			return nil
		}
		spare, stamp := s.ready[0], s.stamp
		s.ready = s.ready[1:]
		s.mu.Unlock()
		if now, found := stampOf(s.executable); !found || now != stamp {
			s.discard()
			spare.worker.Close()
			return nil
		}
		if exited, _ := spare.worker.Exited(); exited {
			spare.worker.Close()
			continue
		}
		spare.route.set(deliver)
		return spare.worker
	}
}

// discard closes every waiting process.
func (s *spareIeds) discard() {
	s.mu.Lock()
	ready := s.ready
	s.ready = nil
	s.mu.Unlock()
	closeSpares(ready)
}

// close discards the spares and starts no others (shutdown); a process being started is
// closed by its starter.
func (s *spareIeds) close() {
	s.mu.Lock()
	s.closed = true
	filling := s.filling
	s.mu.Unlock()
	if filling != nil {
		<-filling
	}
	s.discard()
}

func closeSpares(spares []*spareIed) {
	var group sync.WaitGroup
	for _, spare := range spares {
		group.Add(1)
		go func(spare *spareIed) {
			defer group.Done()
			spare.worker.Close()
		}(spare)
	}
	group.Wait()
}
