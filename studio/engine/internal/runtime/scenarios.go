package runtime

import (
	"fmt"
	"math"
	"sync"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// scenarioRunner is simulation/scenario_runner.py: one-shot timelines on the run's clock.
type scenarioRunner struct {
	c     *Coordinator
	mu    sync.Mutex
	order []string
	runs  map[string]*scenarioRun
}

type scenarioRun struct {
	scenario *pyjson.Object
	start    float64
	next     int
	state    string
}

func newScenarioRunner(c *Coordinator) *scenarioRunner {
	return &scenarioRunner{c: c, runs: map[string]*scenarioRun{}}
}

func (s *scenarioRunner) reset(scenarios []any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.order, s.runs = nil, map[string]*scenarioRun{}
	for _, item := range scenarios {
		scenario, _ := item.(*pyjson.Object)
		if pyval.Truthy(pyval.Get(scenario, "enabled")) {
			_ = s.startLocked(scenario, 0)
		}
	}
}

func (s *scenarioRunner) startLocked(scenario *pyjson.Object, elapsed float64) error {
	id := pyval.Str(scenario, "id")
	if existing := s.runs[id]; existing != nil && existing.state == "running" {
		return apperr.ValueError("This scenario is already running")
	}
	if s.runs[id] == nil {
		s.order = append(s.order, id)
	}
	s.runs[id] = &scenarioRun{scenario: pyval.Copy(scenario).(*pyjson.Object), start: elapsed, state: "running"}
	s.c.emit("scenario_started", pyval.Obj("scenario_id", id, "name", pyval.Get(scenario, "name")))
	return nil
}

// StartScenario is POST /scenarios/<id>/start.
func (c *Coordinator) StartScenario(key string) error {
	c.mu.Lock()
	live, elapsed := c.live(), c.elapsed
	c.mu.Unlock()
	if !live {
		return apperr.ValueError("Run the IEDs before starting a scenario")
	}
	for _, item := range pyval.List(c.opt.Project.Current(), "scenarios") {
		scenario, _ := item.(*pyjson.Object)
		if pyval.Str(scenario, "id") == key {
			c.scenarios.mu.Lock()
			defer c.scenarios.mu.Unlock()
			return c.scenarios.startLocked(scenario, elapsed)
		}
	}
	return apperr.ValueError("Unknown scenario")
}

func (s *scenarioRunner) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range s.order {
		if run := s.runs[id]; run.state == "running" {
			run.state = "cancelled"
			s.c.emit("scenario_cancelled", pyval.Obj("scenario_id", id))
		}
	}
}

func (s *scenarioRunner) advance(elapsed float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range s.order {
		run := s.runs[id]
		steps := pyval.List(run.scenario, "steps")
		for run.state == "running" && run.next < len(steps) {
			step := steps[run.next].(*pyjson.Object)
			at, _ := pyval.Float(pyval.Get(step, "at"))
			if at > elapsed-run.start {
				break
			}
			actual, err := s.step(step)
			if err != nil {
				run.state = "failed"
				s.c.emit("scenario_failed", pyval.Obj("scenario_id", id, "step_id", pyval.Get(step, "id"),
					"message", errorText(err)))
				continue
			}
			run.next++
			s.c.emit("scenario_step_applied", pyval.Obj("scenario_id", id, "step_id", pyval.Get(step, "id"),
				"ied_id", pyval.Get(step, "ied_id"), "ref", pyval.Get(step, "ref"), "action", pyval.Get(step, "action"),
				"value", actual, "scheduled_seconds", pyval.Get(step, "at"), "observed_seconds", elapsed-run.start))
			if run.next == len(steps) {
				run.state = "completed"
				s.c.emit("scenario_completed", pyval.Obj("scenario_id", id))
			}
		}
	}
}

func (s *scenarioRunner) step(step *pyjson.Object) (any, error) {
	iedID, ref := pyval.Str(step, "ied_id"), pyval.Str(step, "ref")
	if pyval.Get(step, "action") == "set" {
		return s.c.Update(iedID, ref, pyval.Get(step, "value"), nil, "scenario")
	}
	actual, err := s.c.readNative(iedID, ref)
	if err != nil {
		return nil, err
	}
	expected := pyval.Get(step, "value")
	var equal bool
	if expectedFloat, isFloat := expected.(float64); isFloat {
		actualFloat, ok := pyval.Float(actual)
		if !ok {
			return nil, fmt.Errorf("must be real number, not %s", pyval.TypeName(actual))
		}
		equal = isClose(actualFloat, expectedFloat)
	} else {
		equal = pyval.Equal(actual, expected)
	}
	if !equal {
		return nil, apperr.ValueError(fmt.Sprintf("Assertion failed: expected %s, observed %s",
			pyval.Repr(expected), pyval.Repr(actual)))
	}
	return actual, nil
}

// isClose is math.isclose(a, b, rel_tol=1e-6, abs_tol=1e-6).
func isClose(a, b float64) bool {
	if a == b {
		return true
	}
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return false
	}
	diff := math.Abs(b - a)
	return diff <= math.Abs(1e-6*b) || diff <= math.Abs(1e-6*a) || diff <= 1e-6
}

// Snapshot is GET /scenarios/runs: [{"id", "state", "completed_steps"}].
func (c *Coordinator) ScenarioRuns() []any {
	s := c.scenarios
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]any, 0, len(s.order))
	for _, id := range s.order {
		run := s.runs[id]
		items = append(items, pyval.Obj("id", id, "state", run.state, "completed_steps", int64(run.next)))
	}
	return items
}

// readNative is Supervisor.read: one native read of a simulated IED.
func (c *Coordinator) readNative(iedID, ref string) (any, error) {
	target := c.simulatedRun(iedID)
	if target == nil {
		return nil, errNotRunning
	}
	var value any
	err := target.do(func() error {
		var err error
		value, err = target.worker.Read(ref)
		return err
	})
	return value, err
}
