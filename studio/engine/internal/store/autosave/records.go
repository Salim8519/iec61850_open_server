package autosave

import (
	"bytes"
	"errors"
	"math/big"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
)

// Journal line kinds (one pyjson.Compact object per line, "\n" terminated):
//
//	{"journal":1,"base":"<sha256 hex of autosave.scadabench, or "" when there was none>"}
//	{"rev":N,"doc":{...}}                                  full compact document
//	{"rev":N,"keys":[...],"set":{...},"scl":{"order":[...],"add":{...}}}   change
//	{"snap":N,"sha256":"<hex>"}                            snapshot N is being renamed in
//
// A change record lists every top-level key of the new compact document in order ("keys"),
// the values of the keys that changed ("set"; absent keys keep the previous value), and,
// only when the scl_sources table changed, its digest order and the texts of new digests
// ("scl"): SCL text is journaled only when a source is added.
const sclKey = "scl_sources"

func headerRecord(base string) *pyjson.Object {
	return pyjson.NewObject().Set("journal", int64(1)).Set("base", base)
}

func fullRecord(rev uint64, doc *pyjson.Object) *pyjson.Object {
	return pyjson.NewObject().Set("rev", int64(rev)).Set("doc", doc)
}

func snapRecord(rev uint64, sha string) *pyjson.Object {
	return pyjson.NewObject().Set("snap", int64(rev)).Set("sha256", sha)
}

// changeLine is the journal line of a change record from prev to next. Values in "set"
// are the encoder's json.dumps bytes (memoised by identity, so an unchanged subtree is
// never encoded again; Fold decodes either spacing). A key is set when its encoded value
// differs from prev's; an identical value (same object) is never encoded.
func changeLine(enc *archive.Encoder, rev uint64, prev, next *pyjson.Object) ([]byte, error) {
	keys := next.Keys()
	listed := make([]any, len(keys))
	for index, key := range keys {
		listed[index] = key
	}
	head, _ := pyjson.Compact(pyjson.NewObject().Set("rev", int64(rev)).Set("keys", listed))
	line := append(head[:len(head)-1], `,"set":{`...)
	var scl *pyjson.Object
	written := 0
	for _, key := range keys {
		value, _ := next.Get(key)
		old, found := prev.Get(key)
		if key == sclKey {
			if oldTable, newTable, ok := tables(old, value); ok {
				scl = tableChange(oldTable, newTable)
				continue
			}
		}
		if found && identical(old, value) {
			continue
		}
		encoded, err := enc.Dumps(value)
		if err != nil {
			return nil, err
		}
		if found {
			if before, err := enc.Dumps(old); err == nil && bytes.Equal(before, encoded) {
				continue
			}
		}
		if written > 0 {
			line = append(line, ',')
		}
		line = append(append(pyjson.AppendString(line, key), ':'), encoded...)
		written++
	}
	line = append(line, '}')
	if scl != nil {
		encoded, _ := pyjson.Compact(scl) // digests and SCL texts only
		line = append(append(line, `,"scl":`...), encoded...)
	}
	return append(line, '}', '\n'), nil
}

func tables(old, value any) (*pyjson.Object, *pyjson.Object, bool) {
	oldTable, okOld := old.(*pyjson.Object)
	newTable, okNew := value.(*pyjson.Object)
	if !okOld || !okNew || !textsOnly(oldTable) || !textsOnly(newTable) {
		return nil, nil, false
	}
	return oldTable, newTable, true
}

func textsOnly(table *pyjson.Object) bool {
	for _, key := range table.Keys() {
		if value, _ := table.Get(key); !isText(value) {
			return false
		}
	}
	return true
}

func isText(value any) bool {
	_, ok := value.(string)
	return ok
}

// tableChange returns {"order", "add"} or nil when the digest order is unchanged.
func tableChange(old, next *pyjson.Object) *pyjson.Object {
	oldKeys, newKeys := old.Keys(), next.Keys()
	if equalStrings(oldKeys, newKeys) {
		return nil
	}
	order := make([]any, len(newKeys))
	added := pyjson.NewObject()
	for index, key := range newKeys {
		order[index] = key
		if _, known := old.Get(key); !known {
			text, _ := next.Get(key)
			added.Set(key, text)
		}
	}
	return pyjson.NewObject().Set("order", order).Set("add", added)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

// identical reports whether two values are the same immutable value without encoding:
// the same object or list (shared subtrees of copy-on-write snapshots) or equal scalars.
func identical(a, b any) bool {
	switch x := a.(type) {
	case *pyjson.Object:
		y, ok := b.(*pyjson.Object)
		return ok && x == y
	case []any:
		y, ok := b.([]any)
		return ok && len(x) == len(y) && (len(x) == 0 || &x[0] == &y[0])
	case *big.Int:
		y, ok := b.(*big.Int)
		return ok && x.Cmp(y) == 0
	case nil, bool, string, int, int64:
		return a == b
	}
	return false
}

var errMissing = errors.New("autosave journal refers to data that is not in the autosave")

// apply returns the document after record; pool holds every SCL text seen so far.
func apply(doc, record *pyjson.Object, pool map[string]string) (*pyjson.Object, error) {
	if full, ok := record.Get("doc"); ok {
		next, isDoc := full.(*pyjson.Object)
		if !isDoc {
			return nil, errMissing
		}
		addTexts(pool, next)
		return next, nil
	}
	if doc == nil {
		return nil, errMissing
	}
	keys, _ := record.Get("keys")
	listed, _ := keys.([]any)
	setValue, _ := record.Get("set")
	set, _ := setValue.(*pyjson.Object)
	sclValue, hasSCL := record.Get("scl")
	if set == nil || listed == nil {
		return nil, errMissing
	}
	next := pyjson.NewObject()
	for _, item := range listed {
		key, _ := item.(string)
		if value, changed := set.Get(key); changed {
			next.Set(key, value)
		} else if key == sclKey && hasSCL {
			table, err := rebuildTable(sclValue, pool)
			if err != nil {
				return nil, err
			}
			next.Set(key, table)
		} else if value, kept := doc.Get(key); kept {
			next.Set(key, value)
		} else {
			return nil, errMissing
		}
	}
	addTexts(pool, next)
	return next, nil
}

func rebuildTable(change any, pool map[string]string) (*pyjson.Object, error) {
	description, _ := change.(*pyjson.Object)
	if description == nil {
		return nil, errMissing
	}
	if added, _ := description.Get("add"); added != nil {
		if texts, ok := added.(*pyjson.Object); ok {
			addTable(pool, texts)
		}
	}
	orderValue, _ := description.Get("order")
	order, _ := orderValue.([]any)
	table := pyjson.NewObject()
	for _, item := range order {
		key, _ := item.(string)
		text, found := pool[key]
		if !found {
			return nil, errMissing
		}
		table.Set(key, text)
	}
	return table, nil
}

func addTexts(pool map[string]string, doc *pyjson.Object) {
	if value, _ := doc.Get(sclKey); value != nil {
		if table, ok := value.(*pyjson.Object); ok {
			addTable(pool, table)
		}
	}
}

func addTable(pool map[string]string, table *pyjson.Object) {
	for _, key := range table.Keys() {
		if value, _ := table.Get(key); isText(value) {
			pool[key] = value.(string)
		}
	}
}
