package pyjson

import (
	"fmt"
	"math/rand"
	"testing"
)

// referenceObject is the former map-backed Object (insertion order plus map): the compact
// entries must behave exactly like it (ADR-0116).
type referenceObject struct {
	keys   []string
	values map[string]any
}

func (r *referenceObject) set(key string, value any) {
	if _, found := r.values[key]; !found {
		r.keys = append(r.keys, key)
	}
	r.values[key] = value
}

func (r *referenceObject) remove(key string) {
	if _, found := r.values[key]; !found {
		return
	}
	delete(r.values, key)
	for index, existing := range r.keys {
		if existing == key {
			r.keys = append(r.keys[:index], r.keys[index+1:]...)
			return
		}
	}
}

func sameAsReference(t *testing.T, step int, object *Object, reference *referenceObject) {
	t.Helper()
	keys := object.Keys()
	if len(keys) != len(reference.keys) || object.Len() != len(reference.keys) {
		t.Fatalf("step %d: keys %q, want %q", step, keys, reference.keys)
	}
	for index, key := range reference.keys {
		value, found := object.Get(key)
		if keys[index] != key || !found || value != reference.values[key] {
			t.Fatalf("step %d: key %d %q=%v found=%v, want %q=%v", step, index, keys[index], value, found, key, reference.values[key])
		}
	}
	if _, found := object.Get("absent"); found {
		t.Fatalf("step %d: absent key found", step)
	}
	if got, want := string(mustDumps(t, object)), string(mustDumps(t, referenceDump(reference))); got != want {
		t.Fatalf("step %d: dumps %s, want %s", step, got, want)
	}
}

// referenceDump builds the expected JSON from the reference with a fresh object per key order.
func referenceDump(reference *referenceObject) *Object {
	object := NewObject()
	for _, key := range reference.keys {
		object.entries = append(object.entries, entry{key, reference.values[key]})
	}
	return object
}

func mustDumps(t *testing.T, value any) []byte {
	t.Helper()
	data, err := Dumps(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestObjectMatchesTheMapBackedObject(t *testing.T) {
	random := rand.New(rand.NewSource(61850))
	for round := 0; round < 400; round++ {
		object := NewObject()
		if round%2 == 1 {
			object = NewObjectSize(random.Intn(12))
		}
		reference := &referenceObject{values: map[string]any{}}
		size := 1 + random.Intn(3*indexAfter) // crosses the index threshold both ways
		for step := 0; step < 300; step++ {
			key := fmt.Sprintf("k%d", random.Intn(size))
			switch random.Intn(10) {
			case 0, 1, 2:
				object.Delete(key)
				reference.remove(key)
			default:
				value := int64(random.Intn(1000))
				object.Set(key, value)
				reference.set(key, value)
			}
			sameAsReference(t, step, object, reference)
		}
	}
}

func TestDecodedObjectsKeepPythonDictRules(t *testing.T) {
	members := ""
	for index := 0; index < 3*indexAfter; index++ {
		members += fmt.Sprintf(`"k%d": %d, `, index%(2*indexAfter), index)
	}
	for _, text := range []string{`{"a": 1, "b": 2, "a": 3}`, "{" + members + `"z": 0}`, `{}`,
		`{"x": {"a": 1, "a": {"b": 2, "b": 3}}, "y": [ {"c": 1} ], "x": 4}`} {
		value, err := Decode([]byte(text))
		if err != nil {
			t.Fatal(err)
		}
		object := value.(*Object)
		reference := &referenceObject{values: map[string]any{}}
		parsed, _ := Decode([]byte(text))
		for _, key := range parsed.(*Object).Keys() {
			item, _ := parsed.(*Object).Get(key)
			reference.set(key, item)
		}
		if got := string(mustDumps(t, object)); got != string(mustDumps(t, referenceDump(reference))) {
			t.Fatalf("%s: %s", text, got)
		}
		if text == `{"a": 1, "b": 2, "a": 3}` && string(mustDumps(t, object)) != `{"a": 3, "b": 2}` {
			t.Fatalf("repeated key: %s", mustDumps(t, object))
		}
		object.Set("new", int64(1))
		if keys := object.Keys(); keys[len(keys)-1] != "new" {
			t.Fatalf("set after decode: %q", keys)
		}
	}
}
