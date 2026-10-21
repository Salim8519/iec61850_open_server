package uplink

// device is an open adapter as the pump sees it: *npcap.Handle, or a fake in tests.
type device interface {
	// Next returns a waiting frame without blocking, or nil, nil when none is waiting.
	Next() ([]byte, error)
	Send(frame []byte) error
	// Event is the handle signalled when frames are waiting (0: none).
	Event() uintptr
	Close()
}

// batch bounds one pass over received or queued frames, so a flood in one
// direction can never starve the other.
const batch = 256

// pump sleeps in the kernel until a frame arrives, a frame is queued to send or
// the uplink stops; there is no polling. The handle is only ever used from this
// goroutine, which run locked to its OS thread.
func (u *Uplink) pump(dev device) error {
	for {
		select {
		case <-u.stop:
			return nil
		default:
		}
		// Clear before draining: a Send after this point always wakes the next wait.
		u.pending.Store(false)
		moreOut := u.flush(dev)
		moreIn, err := u.receive(dev)
		if err != nil {
			return err
		}
		if moreOut || moreIn {
			continue
		}
		if err := u.sig.wait(dev.Event()); err != nil {
			return err
		}
		u.wakes.Add(1)
	}
}

// receive hands every waiting frame to onFrame. It reports true when frames may
// still be waiting after a full batch.
func (u *Uplink) receive(dev device) (bool, error) {
	for i := 0; i < batch; i++ {
		frame, err := dev.Next()
		if err != nil {
			return false, err
		}
		if frame == nil {
			return false, nil
		}
		u.onFrame(frame)
	}
	return true, nil
}

// flush sends queued frames in order. It reports true when frames are still queued.
func (u *Uplink) flush(dev device) bool {
	for i := 0; i < batch; i++ {
		select {
		case frame := <-u.out:
			if !u.allow.Load() {
				continue
			}
			if err := dev.Send(frame); err != nil {
				if !u.failing {
					u.failing = true
					u.onState("send_failed", err.Error())
				}
			} else if u.failing {
				u.failing = false
				u.onState("open", "")
			}
		default:
			return false
		}
	}
	return len(u.out) > 0
}
