package runtime

import (
	"sync"

	"studio-engine/internal/runtime/plant"
)

// snapshotInFlight bounds the reads one IED worker has outstanding at Run (ADR-0129). The worker
// answers requests in order as fast as it reads them, so overlapping them removes the round trip
// per point (1,704 points took 18.9 s one by one on a busy laptop).
const snapshotInFlight = 64

// snapshotRead is one binding's read at Run: the refs to read and, after readSnapshots, their
// values or the first error in ref order.
type snapshotRead struct {
	binding              plant.Binding
	target               *running
	qualityRef, stampRef string
	refs                 []string
	values               []any
	errs                 []error
}

// failure is the read's first error in ref order (nil when every ref was read).
func (r *snapshotRead) failure() error {
	for _, err := range r.errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// readSnapshots reads every planned snapshot: the IEDs read in parallel, and each IED's reads
// overlap inside its actor. The plan keeps its order, so everything that uses the values after
// this sees exactly what reading one by one gave.
func readSnapshots(plan []*snapshotRead) {
	byTarget := map[*running][]*snapshotRead{}
	var order []*running
	for _, read := range plan {
		read.values, read.errs = make([]any, len(read.refs)), make([]error, len(read.refs))
		if _, seen := byTarget[read.target]; !seen {
			order = append(order, read.target)
		}
		byTarget[read.target] = append(byTarget[read.target], read)
	}
	var ieds sync.WaitGroup
	for _, target := range order {
		ieds.Add(1)
		go func(target *running, reads []*snapshotRead) {
			defer ieds.Done()
			if err := target.do(func() error { readOverlapped(target, reads); return nil }); err != nil {
				for _, read := range reads {
					read.errs[0] = err
				}
			}
		}(target, byTarget[target])
	}
	ieds.Wait()
}

// readOverlapped sends one IED's reads with up to snapshotInFlight outstanding.
func readOverlapped(target *running, reads []*snapshotRead) {
	slots := make(chan struct{}, snapshotInFlight)
	var all sync.WaitGroup
	for _, read := range reads {
		for index, ref := range read.refs {
			slots <- struct{}{}
			all.Add(1)
			go func(read *snapshotRead, index int, ref string) {
				defer func() { <-slots; all.Done() }()
				read.values[index], read.errs[index] = target.worker.Read(ref)
			}(read, index, ref)
		}
	}
	all.Wait()
}
