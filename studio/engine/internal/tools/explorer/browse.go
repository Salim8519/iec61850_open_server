package explorer

import (
	"errors"
	"strings"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/numlog"
	"studio-engine/internal/tools/pytext"
)

const requestTimeout = 30 * time.Second

// Devices lists the logical devices (cached until Disconnect).
func (s *Service) Devices() (any, error) {
	return s.cached("devices", func() (any, error) { return s.names(nil) })
}

// Nodes lists the logical nodes of device (cached).
func (s *Service) Nodes(device string) (any, error) {
	return s.cached("nodes\x00"+device, func() (any, error) {
		return s.names(pyjson.NewObject().Set("device", device))
	})
}

func (s *Service) names(fields *pyjson.Object) (any, error) {
	answer, err := s.request("explore", requestTimeout, fields)
	if err != nil {
		return nil, err
	}
	object, _ := answer.(*pyjson.Object)
	if object == nil {
		object = pyjson.NewObject()
	}
	names, found := object.Get("names")
	if !found {
		return nil, errors.New("KeyError: 'names'") // Python's subscript fails: HTTP 500
	}
	return names, nil
}

// Node is the shaped view of one logical node "LD/LN" (cached).
func (s *Service) Node(reference string) (*pyjson.Object, error) {
	if strings.Count(reference, "/") != 1 {
		return nil, apperr.ValueError("A logical node reference looks like LD/LN, for example IED1CTRL/XCBR1")
	}
	view, err := s.cached("node\x00"+reference, func() (any, error) {
		answer, err := s.request("explore", 60*time.Second, pyjson.NewObject().Set("node", reference))
		if err != nil {
			return nil, err
		}
		object, _ := answer.(*pyjson.Object)
		if object == nil {
			object = pyjson.NewObject()
		}
		return NodeView(reference, object), nil
	})
	if err != nil {
		return nil, err
	}
	return view.(*pyjson.Object), nil
}

func (s *Service) cached(key string, load func() (any, error)) (any, error) {
	s.mu.Lock()
	value, found := s.cache[key]
	s.mu.Unlock()
	if found {
		return value, nil
	}
	value, err := load()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cache[key] = value
	s.mu.Unlock()
	return value, nil
}

// Read reads 1..200 attributes once; values are kept for the watch list.
func (s *Service) Read(items []*pyjson.Object) (*pyjson.Object, error) {
	if len(items) == 0 || len(items) > MaxWatch {
		return nil, apperr.ValueError("Read between 1 and 200 attributes at a time")
	}
	readable, skipped := splitSBO(items) // reading <DO>.SBO would Select the control (change 0177)
	var answer any
	if len(readable) > 0 {
		refs := make([]any, len(readable))
		for index, item := range readable {
			refs[index] = pyjson.NewObject().Set("ref", get(item, "ref")).Set("fc", get(item, "fc"))
		}
		var err error
		if answer, err = s.request("read", requestTimeout, pyjson.NewObject().Set("refs", refs)); err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.reads++
		s.mu.Unlock()
	}
	now := numlog.Now()
	results := pyjson.NewObject()
	for _, ref := range skipped {
		results.Set(ref, notRead(now))
	}
	object, _ := answer.(*pyjson.Object)
	values, _ := get(object, "values").([]any)
	for _, raw := range values {
		entry, _ := raw.(*pyjson.Object)
		if entry == nil {
			continue // the native reader only sends objects
		}
		if structure, found := entry.Get("structure"); found {
			members, _ := structure.(*pyjson.Object)
			if members == nil {
				members = pyjson.NewObject()
			}
			for _, ref := range members.Keys() {
				results.Set(ref, described(ref, get(members, ref), now))
			}
		} else if value, found := entry.Get("value"); found {
			ref := pytext.Str(get(entry, "ref"))
			results.Set(ref, described(ref, value, now))
		} else {
			code := get(entry, "error")
			text := "read failed (MMS error " + pytext.Str(code) + ")"
			if code == "access_error" {
				text = "device refused the read (access error " + pytext.Str(get(entry, "access_error")) + ")"
			}
			results.Set(pytext.Str(get(entry, "ref")), pyjson.NewObject().Set("raw", nil).Set("text", text).
				Set("status", "error").Set("at", now))
		}
	}
	s.mu.Lock()
	for _, ref := range results.Keys() {
		s.values[ref] = get(results, ref).(*pyjson.Object)
	}
	s.mu.Unlock()
	s.changed() // watch values and the read counter
	return results, nil
}

func described(ref string, raw any, now float64) *pyjson.Object {
	result := pyjson.NewObject().Set("raw", raw)
	description := Describe(ref, raw)
	return result.Set("text", get(description, "text")).Set("status", get(description, "status")).Set("at", now)
}
