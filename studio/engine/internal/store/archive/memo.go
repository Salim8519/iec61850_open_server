package archive

import (
	"math/big"
	"strconv"
	"unsafe"

	"studio-engine/internal/pyjson"
)

// The encoder keeps json.dumps bytes of the containers near the top of a document by
// identity: a top-level value, its items and their items (for example the diagram, its
// node list and every node). Documents are immutable once handed over, so an object (by
// pointer) or a list (by first element and length) always encodes the same way; the entry
// holds the value, so its address cannot be reused while cached. A canvas edit therefore
// encodes one node and copies the bytes of the others. Entries not used by the last
// memoKeep encodings are dropped.

type identity struct {
	ptr  unsafe.Pointer
	size int // -1 for an object
}

type memoEntry struct {
	value any
	data  []byte
	gen   uint64
}

const (
	memoDepth = 2
	memoKeep  = 8 // documents (a commit and a background snapshot each encode one)
)

func identityOf(value any) (identity, bool) {
	switch v := value.(type) {
	case *pyjson.Object:
		return identity{unsafe.Pointer(v), -1}, true
	case []any:
		if len(v) > 0 {
			return identity{unsafe.Pointer(&v[0]), len(v)}, true
		}
	}
	return identity{}, false
}

// Dumps is pyjson.Dumps(value) after checkFinite (Python's save order: the ValueError
// for a non-finite float), memoised by identity for value and the two levels below it
// (the journal asks for the values the last Checksum/Encode just encoded); the bytes may
// be shared with the memo and must not be modified. value must not
// be mutated afterwards.
func (e *Encoder) Dumps(value any) ([]byte, error) {
	e.mu.Lock()
	gen := e.gen
	e.mu.Unlock()
	return e.appendMemo(nil, value, 0, gen)
}

// nextGeneration starts the encoding of one document.
func (e *Encoder) nextGeneration() uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.gen++
	return e.gen
}

func (e *Encoder) appendMemo(dst []byte, value any, depth int, gen uint64) ([]byte, error) {
	key, cacheable := identityOf(value)
	if !cacheable || depth > memoDepth {
		if err := checkFinite(value); err != nil {
			return nil, err
		}
		return appendPlain(dst, value)
	}
	e.mu.Lock()
	cached := e.memo[key]
	if cached != nil {
		cached.gen = gen
	}
	e.mu.Unlock()
	if cached != nil {
		if dst == nil {
			return cached.data[:len(cached.data):len(cached.data)], nil // shared: read-only
		}
		return append(dst, cached.data...), nil
	}
	var out []byte
	var err error
	switch v := value.(type) {
	case *pyjson.Object:
		out = append(out, '{')
		for index, name := range v.Keys() {
			if index > 0 {
				out = append(out, ", "...)
			}
			out = append(pyjson.AppendString(out, name), ": "...)
			item, _ := v.Get(name)
			if out, err = e.appendMemo(out, item, depth+1, gen); err != nil {
				return nil, err
			}
		}
		out = append(out, '}')
	case []any:
		out = append(out, '[')
		for index, item := range v {
			if index > 0 {
				out = append(out, ", "...)
			}
			if out, err = e.appendMemo(out, item, depth+1, gen); err != nil {
				return nil, err
			}
		}
		out = append(out, ']')
	}
	e.mu.Lock()
	e.memo[key] = &memoEntry{value: value, data: out, gen: gen}
	e.mu.Unlock()
	if dst == nil {
		return out[:len(out):len(out)], nil
	}
	return append(dst, out...), nil
}

// appendPlain is pyjson.Dumps for a value below the memoised levels or a scalar.
func appendPlain(dst []byte, value any) ([]byte, error) {
	switch v := value.(type) {
	case nil:
		return append(dst, "null"...), nil
	case bool:
		return strconv.AppendBool(dst, v), nil
	case int:
		return strconv.AppendInt(dst, int64(v), 10), nil
	case int64:
		return strconv.AppendInt(dst, v, 10), nil
	case *big.Int:
		return v.Append(dst, 10), nil
	case float64:
		return pyjson.AppendFloat(dst, v), nil
	case string:
		return pyjson.AppendString(dst, v), nil
	}
	encoded, err := pyjson.Dumps(value)
	if err != nil {
		return nil, err
	}
	return append(dst, encoded...), nil
}

// sweep drops entries no encoding used in the last memoKeep generations.
func (e *Encoder) sweep(gen uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if gen%memoKeep != 0 {
		return
	}
	for key, entry := range e.memo {
		if entry.gen+memoKeep < gen {
			delete(e.memo, key)
		}
	}
}
