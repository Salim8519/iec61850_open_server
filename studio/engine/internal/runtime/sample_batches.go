package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// nativeBatchLimit is studio_ied's update_batch maximum (worker/updates/values.cpp). One batch
// is applied under one data-model lock, so it is one change for reports and GOOSE. A mode
// change on a large LD (Mod, every LN's Beh, their times) fits one batch; only a write above
// the limit is sent as several batches, each still whole per point (ADR-0115).
var nativeBatchLimit = 4096

// sendBatches writes updates in as few native batches as the limit allows, never splitting
// one point's value, quality and time (groups), and returns {ref: value read back}. A guarded
// write (a pulse) must fit one batch: its revision check covers the whole write.
func sendBatches(target *running, updates *pyjson.Object, groups [][]string, guard *pyjson.Object) (
	*pyjson.Object, *pyjson.Object, error) {
	if guard != nil {
		if updates.Len() > nativeBatchLimit {
			return nil, nil, apperr.ValueError("A pulse writes too many attributes at once for one native batch")
		}
		guarded, err := target.worker.UpdateGuarded(batchOf(updates, updates.Keys()), guard)
		if err != nil {
			return nil, nil, err
		}
		result := pyval.Map(guarded, "values")
		if result == nil {
			result = pyjson.NewObject()
		}
		return result, guarded, nil
	}
	result := pyjson.NewObject()
	var chunk []string
	send := func() error {
		if len(chunk) == 0 {
			return nil
		}
		part, err := target.worker.UpdateBatch(batchOf(updates, chunk))
		if err != nil {
			return err
		}
		for _, ref := range part.Keys() {
			value, _ := part.Get(ref)
			result.Set(ref, value)
		}
		chunk = chunk[:0]
		return nil
	}
	for _, refs := range groups {
		if len(chunk)+len(refs) > nativeBatchLimit {
			if err := send(); err != nil {
				return nil, nil, err
			}
		}
		chunk = append(chunk, refs...)
	}
	if err := send(); err != nil {
		return nil, nil, err
	}
	return result, nil, nil
}

func batchOf(updates *pyjson.Object, refs []string) []any {
	batch := make([]any, 0, len(refs))
	for _, ref := range refs {
		value, _ := updates.Get(ref)
		batch = append(batch, pyval.Obj("ref", ref, "value", value))
	}
	return batch
}
