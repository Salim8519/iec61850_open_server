package service

import (
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/worker"
)

// Plain reasons a SCADA command is refused before it runs.
const (
	reasonStopped   = "the gateway stopped before this command ran"
	reasonQueueFull = "too many commands are waiting; the gateway refused this one"
)

type job struct {
	process worker.Process
	event   *pyjson.Object
}

// executor is CommandExecutor: one goroutine runs SCADA commands in arrival order.
type executor struct {
	manager *Manager
	jobs    chan job
}

func newExecutor(manager *Manager) *executor {
	e := &executor{manager: manager, jobs: make(chan job, 1024)}
	go e.loop()
	return e
}

// submit queues a SCADA command. It runs on the worker's reader, so it never blocks: a
// command that does not fit is refused, never run out of order.
func (e *executor) submit(process worker.Process, event *pyjson.Object) {
	if process == nil {
		e.refuseUnrun(event, reasonStopped)
		return
	}
	select {
	case e.jobs <- job{process, event}:
	default:
		e.refuseUnrun(event, reasonQueueFull)
		go answer(process, event, false, reasonQueueFull, worker.DefaultTimeout) // not on the reader
	}
}

func (e *executor) loop() {
	for next := range e.jobs {
		if !e.manager.serving(next.process) { // Stop came first: the command is never sent
			e.refuseUnrun(next.event, reasonStopped)
			answer(next.process, next.event, false, reasonStopped, time.Second)
			continue
		}
		ok, reason := e.run(next.event)
		answer(next.process, next.event, ok, reason, worker.DefaultTimeout)
	}
}

// drain refuses every command still waiting; Stop calls it after it took the process away.
// A worker that does not take an answer gets no more of them, so Stop never waits long.
func (e *executor) drain() {
	listening := true
	for {
		select {
		case next := <-e.jobs:
			e.refuseUnrun(next.event, reasonStopped)
			if listening {
				listening = answer(next.process, next.event, false, reasonStopped, time.Second) == nil
			}
		default:
			return
		}
	}
}

// refuseUnrun logs a command that was not run, like the other refusals.
func (e *executor) refuseUnrun(event *pyjson.Object, reason string) {
	m := e.manager
	label := pytext.Str(get(event, "point_id"))
	m.Commands.Add(pyjson.NewObject().Set("point_id", get(event, "point_id")).Set("label", label).
		Set("protocol", get(event, "protocol")).Set("peer", get(event, "peer")).Set("request", "received").
		Set("applied", "refused: "+reason).Set("observed", ""))
	m.emit("gateway_command", pyjson.NewObject().Set("point", label).Set("result", "refused").Set("reason", reason))
}

// answer tells studio_gateway the outcome so it can answer the SCADA master.
func answer(process worker.Process, event *pyjson.Object, ok bool, reason string, timeout time.Duration) error {
	if process == nil {
		return nil
	}
	_, err := process.Request("command_result", timeout, pyjson.NewObject().
		Set("token", get(event, "token")).Set("ok", ok).Set("reason", reason))
	return err
}

// serving reports whether process is still the running gateway's worker.
func (m *Manager) serving(process worker.Process) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return process != nil && m.process == process
}
