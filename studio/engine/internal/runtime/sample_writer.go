package runtime

import (
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/runtime/signal"
)

// item is one (reference, value, quality) write; quality nil means None.
type item struct {
	ref     string
	value   any
	quality any
}

// overriding are the write sources that pause an automatic source (sample_writer.py:48).
var overriding = map[string]bool{"manual": true, "scenario": true, "pulse": true, "pulse_return": true,
	"control_feedback": true, "equipment_button": true, "equipment_test": true}

// simulatedRun returns the running simulated IED (Python: sample_writer.locks[ied_id]).
func (c *Coordinator) simulatedRun(iedID string) *running {
	c.mu.Lock()
	defer c.mu.Unlock()
	item := c.runs[iedID]
	if item == nil || !item.simulated() || item.worker == nil {
		return nil
	}
	return item
}

// apply is SampleWriter.apply on the IED's actor; it returns {ref: value read back}.
func (c *Coordinator) apply(iedID string, items []item, source string) (*pyjson.Object, error) {
	target := c.simulatedRun(iedID)
	if target == nil {
		return nil, errNotRunning
	}
	var result *pyjson.Object
	err := target.do(func() error {
		var err error
		result, _, err = c.writeSample(target, items, source, nil)
		return err
	})
	return result, err
}

type sample struct {
	signal  signal.Signal
	related signal.Companions
}

// writeSample is the body of SampleWriter.apply and must run on target's actor. With a guard
// it sends update_guarded and also returns the native {values, revision}.
func (c *Coordinator) writeSample(target *running, items []item, source string, guard *pyjson.Object) (*pyjson.Object, *pyjson.Object, error) {
	id := target.model.id
	c.mu.Lock()
	live := c.live()
	c.mu.Unlock()
	if !live {
		return nil, nil, apperr.ValueError("Run the IED before updating a live value")
	}
	updates := pyjson.NewObject()
	var samples []sample
	timestamp := time.Now().UnixMilli()
	for _, entry := range items {
		info, ok := target.model.lookup(entry.ref)
		if !ok || !info.Writable || info.FC == "CF" {
			return nil, nil, apperr.ValueError("This attribute is not runtime-editable")
		}
		if source == "generator" {
			c.mu.Lock()
			spec := c.srcs[sourceKey{id, entry.ref}]
			skip := spec == nil || spec.overridden
			c.mu.Unlock()
			if skip {
				continue
			}
		}
		batch, related, err := signal.MakeSample(info, target.model.lookup, entry.value, timestamp, entry.quality)
		if err != nil {
			return nil, nil, err
		}
		for _, update := range batch {
			updates.Set(update.Ref, update.Value)
		}
		samples = append(samples, sample{info, related})
	}
	if updates.Len() == 0 {
		return pyjson.NewObject(), nil, nil
	}
	if updates.Len() > 256 {
		return nil, nil, apperr.ValueError("Sample batch exceeds the native 256-attribute limit")
	}
	batch := make([]any, 0, updates.Len())
	for _, ref := range updates.Keys() {
		value, _ := updates.Get(ref)
		batch = append(batch, pyval.Obj("ref", ref, "value", value))
	}
	var result, guarded *pyjson.Object
	var err error
	if guard != nil {
		guarded, err = target.worker.UpdateGuarded(batch, guard)
		result = pyval.Map(guarded, "values")
		if result == nil {
			result = pyjson.NewObject()
		}
	} else {
		result, err = target.worker.UpdateBatch(batch)
	}
	if err != nil {
		return nil, nil, err
	}
	for _, ref := range result.Keys() {
		value, _ := result.Get(ref)
		target.table.set(ref, value)
	}
	companion := func(present bool, ref string) any {
		if !present {
			return int64(0)
		}
		value, _ := target.table.get(ref)
		return value
	}
	c.mu.Lock()
	for _, entry := range samples {
		target.table.setTQ(entry.signal.Ref, companion(entry.related.HasT, entry.related.T.Ref),
			companion(entry.related.HasQ, entry.related.Q.Ref))
		if spec := c.srcs[sourceKey{id, entry.signal.Ref}]; spec != nil && overriding[source] {
			spec.overridden = true
		}
	}
	c.mu.Unlock()
	for _, entry := range samples {
		if source != "pulse" && source != "pulse_return" {
			c.pulses.cancel(id, entry.signal.Ref, true, "Newer signal write")
		}
		changes := pyjson.NewObject()
		for _, ref := range append([]string{entry.signal.Ref}, entry.related.Refs()...) {
			if value, ok := result.Get(ref); ok {
				changes.Set(ref, value)
			}
		}
		value, _ := result.Get(entry.signal.Ref)
		quality, timestamp := companion(entry.related.HasQ, entry.related.Q.Ref), companion(entry.related.HasT, entry.related.T.Ref)
		c.emit("signal", pyval.Obj("ied_id", id, "ref", entry.signal.Ref, "value", value, "source", source,
			"quality", quality, "source_timestamp", timestamp, "updates", changes))
	}
	keys := make([]bindingKey, 0, result.Len())
	for _, ref := range result.Keys() {
		keys = append(keys, bindingKey{id, ref})
	}
	c.plant.publishChanged(keys)
	return result, guarded, nil
}
