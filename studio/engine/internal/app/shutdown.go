package app

import "time"

// resumeWait bounds how long Close waits for a gateway resume that is still starting.
const resumeWait = 15 * time.Second

// Close stops the engine after the HTTP server stopped, in the order that keeps every
// consumer ahead of what it consumes: background loops, gateway and master (their
// auto-start choice is kept, as when Python's process ended), live acquisition, the
// coordinator (Stop, then its own goroutines), Explorer and Find IEDs, the command client
// and the XSD helper, and last the project service (final snapshot, journal removed). The
// caller releases the data-folder lease afterwards. Close is safe to call twice.
func (e *Engine) Close() error {
	var err error
	e.closeOnce.Do(func() {
		if e.cancel != nil {
			e.cancel()
		}
		if e.resumed != nil && e.App != nil {
			select {
			case <-e.resumed:
			case <-time.After(resumeWait):
				e.options.Logf("Gateway resume did not finish within %s; stopping anyway", resumeWait)
			}
		}
		if e.Gateway != nil {
			_, _ = e.Gateway.Stop(false)
		}
		if e.Master != nil {
			_, _ = e.Master.Disconnect(false)
		}
		if e.Live != nil {
			e.Live.Stop()
		}
		if e.Runtime != nil {
			e.Runtime.Close()
		}
		if e.Explorer != nil {
			e.Explorer.Close()
		}
		if e.Scanner != nil {
			e.Scanner.Stop()
		}
		if e.Control != nil {
			e.Control.Close()
		}
		e.closeValidator()
		if e.Recovery != nil {
			e.Recovery.Wait(-1) // a restore still running must not race the final snapshot
		}
		if e.Projects != nil {
			err = e.Projects.Close()
		}
	})
	return err
}

func (e *Engine) closeValidator() {
	if e.validator != nil {
		e.validator.Close()
	}
}
