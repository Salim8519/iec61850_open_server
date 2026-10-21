//go:build parity

package main

import (
	"fmt"
	"net/http"

	"studio-engine/internal/app"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Write-through of the in-place project edits Python tests make (`projects.project[...]
// [key] = value`, `ied.update(...)`, inspect_ops project_set/project_update). The Go project
// is immutable, so the edit is a copy along the path committed through Projects.Update.
// Python's in-place edit neither saved nor changed the revision; here the revision moves
// and the edit is journaled, which the tests using these operations do not observe.

func init() {
	parityOps["POST project/set"] = func(e *app.Engine, body *pyjson.Object, _ *http.Request) (any, error) {
		key, _ := field(body, "key").(string)
		value := field(body, "value")
		return nil, writeThrough(e, listOf(field(body, "path")), func(target *pyjson.Object) {
			target.Set(key, value)
		})
	}
	parityOps["POST project/update"] = func(e *app.Engine, body *pyjson.Object, _ *http.Request) (any, error) {
		values, _ := field(body, "values").(*pyjson.Object)
		return nil, writeThrough(e, listOf(field(body, "path")), func(target *pyjson.Object) {
			if values == nil {
				return
			}
			for _, key := range values.Keys() {
				item, _ := values.Get(key)
				target.Set(key, item)
			}
		})
	}
}

func writeThrough(e *app.Engine, path []any, edit func(*pyjson.Object)) error {
	_, err := e.Projects.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		var current any = draft
		for _, step := range path {
			next, err := copyStep(current, step)
			if err != nil {
				return nil, err
			}
			current = next
		}
		target, ok := current.(*pyjson.Object)
		if !ok {
			return nil, apperr.ValueError("project path does not lead to an object")
		}
		edit(target)
		return target, nil
	})
	return err
}

// copyStep replaces the child at step with a shallow copy in its (already copied) parent
// and returns the copy, so the committed project never shares an edited value.
func copyStep(parent any, step any) (any, error) {
	switch container := parent.(type) {
	case *pyjson.Object:
		key, ok := step.(string)
		if !ok {
			return nil, apperr.KeyError(fmt.Sprintf("%v", step))
		}
		child, found := container.Get(key)
		if !found {
			return nil, apperr.KeyError(key)
		}
		copied := shallowCopy(child)
		container.Set(key, copied)
		return copied, nil
	case []any:
		index := -1
		if position, ok := step.(int64); ok {
			index = int(position)
			if index < 0 {
				index += len(container)
			}
		} else if id := field(step, "id"); id != nil {
			for i, item := range container {
				if field(item, "id") == id {
					index = i
					break
				}
			}
		}
		if index < 0 || index >= len(container) {
			return nil, apperr.ValueError("project path step not found")
		}
		copied := shallowCopy(container[index])
		container[index] = copied
		return copied, nil
	}
	return nil, apperr.ValueError("project path walks into a value")
}

func shallowCopy(value any) any {
	switch v := value.(type) {
	case *pyjson.Object:
		out := pyjson.NewObject()
		for _, key := range v.Keys() {
			item, _ := v.Get(key)
			out.Set(key, item)
		}
		return out
	case []any:
		return append([]any(nil), v...)
	}
	return value
}
