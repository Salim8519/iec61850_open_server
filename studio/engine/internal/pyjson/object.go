// Package pyjson reads and writes JSON byte-for-byte like CPython's json module and Flask's
// jsonify (ADR-0075), so the Go engine answers exactly as the Python engine did.
//
// Values are: nil, bool, int, int64, *big.Int, float64, string, []any and *Object.
package pyjson

// indexAfter is the size above which an object also keeps a key index. Smaller objects (an
// IED signal has 9 keys) search their entries directly: fewer bytes and no map (ADR-0116).
const indexAfter = 16

// entry is one key and its value.
type entry struct {
	key   string
	value any
}

// Object is a JSON object that keeps insertion order like a Python dict: setting an
// existing key keeps its place, deleting and setting again moves it to the end.
type Object struct {
	entries []entry
	index   map[string]int // key -> position, only above indexAfter entries
}

// NewObject returns an empty ordered object.
func NewObject() *Object {
	return &Object{}
}

// NewObjectSize returns an empty ordered object with room for size keys, for builders that
// know how many keys they set (no growth, no spare room).
func NewObjectSize(size int) *Object {
	return &Object{entries: make([]entry, 0, size)}
}

// objectOf owns entries that hold no repeated key (the decoder's exact-size slice).
func objectOf(entries []entry) *Object {
	object := &Object{entries: entries}
	if len(entries) > indexAfter {
		object.reindex()
	}
	return object
}

func (o *Object) reindex() {
	o.index = make(map[string]int, len(o.entries))
	for position, item := range o.entries {
		o.index[item.key] = position
	}
}

// position is where key is stored, or -1.
func (o *Object) position(key string) int {
	if o.index != nil {
		if position, found := o.index[key]; found {
			return position
		}
		return -1
	}
	for position := range o.entries {
		if o.entries[position].key == key {
			return position
		}
	}
	return -1
}

// Set stores value under key, keeping the key's first position.
func (o *Object) Set(key string, value any) *Object {
	if position := o.position(key); position >= 0 {
		o.entries[position].value = value
		return o
	}
	if cap(o.entries) == 0 {
		o.entries = make([]entry, 0, 4)
	}
	o.entries = append(o.entries, entry{key, value})
	if o.index != nil {
		o.index[key] = len(o.entries) - 1
	} else if len(o.entries) > indexAfter {
		o.reindex()
	}
	return o
}

// Get returns the value stored under key.
func (o *Object) Get(key string) (any, bool) {
	if position := o.position(key); position >= 0 {
		return o.entries[position].value, true
	}
	return nil, false
}

// Delete removes key; a later Set appends it at the end, as in Python.
func (o *Object) Delete(key string) {
	position := o.position(key)
	if position < 0 {
		return
	}
	o.entries = append(o.entries[:position], o.entries[position+1:]...)
	o.entries[:cap(o.entries)][len(o.entries)] = entry{} // drop the moved-out reference
	if o.index != nil {
		delete(o.index, key)
		for later := position; later < len(o.entries); later++ {
			o.index[o.entries[later].key] = later
		}
	}
}

// Keys returns the keys in insertion order (a copy).
func (o *Object) Keys() []string {
	if len(o.entries) == 0 {
		return nil // as append([]string(nil)) did
	}
	keys := make([]string, len(o.entries))
	for position := range o.entries {
		keys[position] = o.entries[position].key
	}
	return keys
}

// Len returns the number of keys.
func (o *Object) Len() int {
	return len(o.entries)
}
