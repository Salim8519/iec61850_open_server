package service

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project"
	"studio-engine/internal/project/history"
	"studio-engine/internal/pyjson"
)

// Undo and Redo over the project history (ADR-0135). Every change committed while the IEDs
// are stopped is a step; changes made while they run (operations, test values, addresses
// given at Start) are not, and Undo is refused until they stop. A step is put back through
// commit like any edit: journaled, saved, announced with project_changed.

// runningWords is the answer while IEDs run.
const runningWords = "Stop the IEDs first. Undo and Redo work on changes made while they are stopped."

// SetRunning tells the service whether IEDs run (the runtime's state hook; never blocks).
func (s *Service) SetRunning(running bool) { s.running.Store(running) }

// Undo puts the last step back.
func (s *Service) Undo() error { return s.stepHistory(true) }

// Redo applies the step Undo put back last.
func (s *Service) Redo() error { return s.stepHistory(false) }

func (s *Service) stepHistory(undo bool) error {
	if s.running.Load() {
		return apperr.New(runningWords)
	}
	s.mutate.Lock()
	defer s.mutate.Unlock()
	st := s.current.Load()
	plan, ok := s.history.Next(undo, st.project)
	if !ok {
		if undo {
			return apperr.New("There is nothing to undo.")
		}
		return apperr.New("There is nothing to redo.")
	}
	next := shallow(st.project)
	for index, key := range plan.Keys {
		if plan.Values[index] == history.Missing {
			next.Delete(key)
		} else {
			next.Set(key, plan.Values[index])
		}
	}
	if err := s.commitUnrecorded(next, st.notice); err != nil {
		return err
	}
	s.history.Done(undo)
	return nil
}

// HistoryStatus is GET /project/history: whether Undo and Redo can run now and what they
// would put back, in words for the button tips.
func (s *Service) HistoryStatus() *pyjson.Object {
	s.mutate.Lock()
	defer s.mutate.Unlock()
	st := s.current.Load()
	_, canUndo := s.history.Next(true, st.project)
	_, canRedo := s.history.Next(false, st.project)
	running := s.running.Load()
	return pyjson.NewObject().Set("can_undo", canUndo && !running).Set("can_redo", canRedo && !running).
		Set("undo", stepLabel(s.history.Keys(true))).Set("redo", stepLabel(s.history.Keys(false))).
		Set("running", running).Set("limit", int64(history.Limit))
}

// NewProject closes the open project and starts an empty one named name ("" keeps the
// default name). This PC's network choice is kept: it belongs to the PC, not the project.
// The history starts again.
func (s *Service) NewProject(name string) error {
	if s.running.Load() {
		return apperr.New("Stop the IEDs before you start a new project or close this one.")
	}
	s.mutate.Lock()
	defer s.mutate.Unlock()
	st := s.current.Load()
	fresh := project.NewProject(s.config.NewID())
	if name != "" {
		fresh.Set("name", name)
	}
	if network, ok := st.project.Get("network"); ok {
		fresh.Set("network", network)
	}
	if err := s.commitUnrecorded(fresh, nil); err != nil {
		return err
	}
	s.history.Clear()
	return nil
}

// commitUnrecorded commits a change that is not itself an Undo step. mutate is held.
func (s *Service) commitUnrecorded(next, notice *pyjson.Object) error {
	s.replaying = true
	defer func() { s.replaying = false }()
	return s.commit(next, notice, true, nil)
}
