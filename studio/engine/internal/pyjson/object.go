// Package pyjson reads and writes JSON byte-for-byte like CPython's json module and Flask's
// jsonify (ADR-0075), so the Go engine answers exactly as the Python engine did.
//
// Values are: nil, bool, int, int64, *big.Int, float64, string, []any and *Object.
package pyjson

// Object is a JSON object that keeps insertion order like a Python dict: setting an
// existing key keeps its place, deleting and setting again moves it to the end.
type Object struct {
	keys   []string
	values map[string]any
}

// NewObject returns an empty ordered object.
func NewObject() *Object {
	return &Object{values: map[string]any{}}
}

// Set stores value under key, keeping the key's first position.
func (o *Object) Set(key string, value any) *Object {
	if _, found := o.values[key]; !found {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
	return o
}

// Get returns the value stored under key.
func (o *Object) Get(key string) (any, bool) {
	value, found := o.values[key]
	return value, found
}

// Delete removes key; a later Set appends it at the end, as in Python.
func (o *Object) Delete(key string) {
	if _, found := o.values[key]; !found {
		return
	}
	delete(o.values, key)
	for index, existing := range o.keys {
		if existing == key {
			o.keys = append(o.keys[:index], o.keys[index+1:]...)
			return
		}
	}
}

// Keys returns the keys in insertion order (a copy).
func (o *Object) Keys() []string {
	return append([]string(nil), o.keys...)
}

// Len returns the number of keys.
func (o *Object) Len() int {
	return len(o.keys)
}
