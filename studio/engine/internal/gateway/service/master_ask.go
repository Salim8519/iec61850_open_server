package service

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/numlog"
	"studio-engine/internal/tools/pytext"
	"studio-engine/internal/tools/worker"
)

func nowSeconds() float64 { return numlog.Now() }

// askAll is _ask_all: GI, integrity poll or Modbus reads; errors become the message.
func (m *Master) askAll() *pyjson.Object {
	answer, err := m.Interrogate()
	if err != nil {
		m.setMessage(askWords[text(m.Config(), "protocol")] + " failed: " + err.Error())
		return nil
	}
	return answer
}

// Interrogate asks the device for every value (Modbus: reads the user's ranges).
func (m *Master) Interrogate() (*pyjson.Object, error) {
	process, err := m.activeProcess()
	if err != nil {
		return nil, err
	}
	if text(m.Config(), "protocol") == "modbus" {
		return m.poll(process)
	}
	// GI confirmation (t1) + its end + the counter interrogation (t1) can take about 50 s.
	reply, err := process.Request("interrogate", 60*time.Second, nil)
	if err != nil {
		return nil, err
	}
	answer, _ := reply.(*pyjson.Object)
	if answer == nil {
		answer = pyjson.NewObject()
	}
	m.mu.Lock()
	m.lastAsked = nowSeconds()
	m.mu.Unlock()
	said, found := answer.Get("text")
	if !found {
		said = ""
	}
	m.Traffic.Add(info("Asked for all values: " + pytext.Str(said)))
	return answer, nil
}

func (m *Master) poll(process worker.Process) (*pyjson.Object, error) {
	ranges, _ := get(m.Config(), "modbus_ranges").([]any)
	if len(ranges) == 0 {
		message := "Add the Modbus ranges to read, or use Find registers (Modbus has no discovery)"
		m.setMessage(message)
		return pyjson.NewObject().Set("confirmed", false).Set("text", message), nil
	}
	reply, err := process.Request("poll", 30*time.Second, pyjson.NewObject().Set("ranges", ranges))
	if err != nil {
		return nil, err
	}
	answer, _ := reply.(*pyjson.Object)
	failed, _ := get(answer, "failed").([]any)
	message := ""
	if len(failed) > 0 {
		message = strconv.Itoa(len(failed)) + " of " + strconv.Itoa(len(ranges)) + " ranges refused"
	}
	m.mu.Lock()
	m.failures, m.lastAsked, m.message = orList(get(answer, "failed")), nowSeconds(), message
	m.mu.Unlock()
	return pyjson.NewObject().Set("confirmed", true).Set("text", "read "+pytext.Str(get(answer, "values"))+" values"), nil
}

func orList(value any) []any {
	if items, ok := value.([]any); ok {
		return items
	}
	return []any{}
}

// Command sends a confirmed command (or one SCADA sent through the converter); it is
// logged as sent, then as answered. A native refusal is an answer, a timeout an error.
func (m *Master) Command(request *pyjson.Object, who string) (*pyjson.Object, error) {
	process, err := m.activeProcess()
	if err != nil {
		return nil, err
	}
	selected, found := request.Get("select")
	if !found {
		selected = false
	}
	entry := m.Commands.Add(pyjson.NewObject().Set("who", who).Set("kind", get(request, "kind")).
		Set("address", get(request, "address")).Set("value", get(request, "value")).Set("select", selected).
		Set("state", "sent").Set("text", "Sent, waiting for the device"))
	seq := get(entry, "seq").(int64)
	reply, err := process.Request("command", 40*time.Second, request)
	var timeout *worker.Timeout
	var native *apperr.Error
	if err != nil && !errors.As(err, &timeout) && errors.As(err, &native) && native.Kind == apperr.Runtime {
		m.Commands.Update(seq, pyjson.NewObject().Set("state", "refused").Set("text", err.Error()))
		return pyjson.NewObject().Set("ok", false).Set("steps", []any{}).Set("text", err.Error()), nil
	}
	if err != nil {
		return nil, err
	}
	answer, _ := reply.(*pyjson.Object)
	if answer == nil {
		answer = pyjson.NewObject()
	}
	var parts []string
	for _, raw := range orList(get(answer, "steps")) {
		step, _ := raw.(*pyjson.Object)
		parts = append(parts, pytext.Str(get(step, "step"))+": "+pytext.Str(get(step, "text")))
	}
	said := strings.Join(parts, "; ")
	state := "refused"
	if pytext.Truthy(get(answer, "ok")) {
		state = "confirmed"
	}
	m.Commands.Update(seq, pyjson.NewObject().Set("state", state).Set("text", said))
	return with(answer, "text", said), nil
}

// FindRegisters starts the bounded Modbus probe in the background.
func (m *Master) FindRegisters(table string, start, end int64) (*pyjson.Object, error) {
	process, err := m.activeProcess()
	if err != nil {
		return nil, err
	}
	m.Traffic.Add(info("Find registers: " + table + " " + strconv.FormatInt(start, 10) + " to " +
		strconv.FormatInt(end, 10)))
	return m.Probe.Start(process, table, start, end, func(result *pyjson.Object) {
		found, _ := get(result, "found").([]any)
		m.Traffic.Add(info("Find registers " + pytext.Str(get(result, "state")) + ": " +
			strconv.Itoa(len(found)) + " ranges answer"))
	})
}
