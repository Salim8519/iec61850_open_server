package hostkey

import (
	"errors"
	"os"
	"time"
)

// startWait is how long turning sharing on waits for a just-started helper to take its lock, so
// Settings shows "On" and not "not running yet" for a helper that is merely still starting.
const startWait = 3 * time.Second

// Process is the helper's process seam: whether it runs in this Windows session, and starting it.
type Process interface {
	Running() bool
	Start(executable string) error
}

// Sharing is the switch "Let my virtual machines use this license": a Run value that starts the
// helper at logon, plus starting the helper right away. The helper watches the same value and
// ends itself soon after it is removed, so turning sharing off needs no process killing.
type Sharing struct {
	Run        RunKey
	Process    Process
	Executable string            // scadabench_hostkey.exe beside the native workers
	Exists     func(string) bool // nil means "the file exists on disk"
}

// State is what Settings shows.
type State struct {
	Enabled     bool // the Run value exists
	Running     bool // a helper holds the mutex
	HelperFound bool // this copy of ScadaBench ships the helper
}

// ErrNoHelper is returned when the helper program is not part of this copy of ScadaBench.
var ErrNoHelper = errors.New("the virtual machine helper is not part of this copy of ScadaBench")

func (s Sharing) found() bool {
	if s.Exists != nil {
		return s.Exists(s.Executable)
	}
	info, err := os.Stat(s.Executable)
	return err == nil && !info.IsDir()
}

// command is the Run value: the helper's path in quotes, so spaces in the path are safe.
func (s Sharing) command() string { return `"` + s.Executable + `"` }

// Wanted reports whether sharing is switched on (the helper's own question).
func (s Sharing) Wanted() bool {
	_, set := s.Run.Get()
	return set
}

// State reads the current state.
func (s Sharing) State() State {
	return State{Enabled: s.Wanted(), Running: s.Process.Running(), HelperFound: s.found()}
}

// Enable turns sharing on: the Run value, then the helper itself (it refuses to serve until this
// PC's license, key and certificate all agree, so starting it is always safe).
func (s Sharing) Enable() error {
	if !s.found() {
		return ErrNoHelper
	}
	if err := s.Run.Set(s.command()); err != nil {
		return err
	}
	if s.Process.Running() {
		return nil
	}
	return s.start()
}

// Disable turns sharing off: the Run value goes; the helper ends itself within seconds.
func (s Sharing) Disable() error { return s.Run.Delete() }

// Refresh runs when the app starts: with sharing on, the Run value follows the app's current
// folder (an updated or portable copy moves) and a missing helper is started.
func (s Sharing) Refresh() error {
	if !s.Wanted() || !s.found() {
		return nil
	}
	if current, _ := s.Run.Get(); current != s.command() {
		if err := s.Run.Set(s.command()); err != nil {
			return err
		}
	}
	if s.Process.Running() {
		return nil
	}
	return s.start()
}

// start runs the helper and waits briefly until it holds its lock. A helper that refuses to serve
// (for example sharing was switched off meanwhile) simply never takes it; that is not an error.
func (s Sharing) start() error {
	if err := s.Process.Start(s.Executable); err != nil {
		return err
	}
	for deadline := time.Now().Add(startWait); !s.Process.Running() && time.Now().Before(deadline); {
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}
