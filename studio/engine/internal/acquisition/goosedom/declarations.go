package goosedom

import (
	"sort"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// Declarations are the merged GOOSE declarations of every SCL file stored with the
// project (scl/goose_sources.read_goose_sources, parsed by another stream).
//
// IEDs maps an IED name to {desc, manufacturer, type, loaded}; Subscriptions, Problems
// and Remote map a subscriber name to its subscriptions, unresolved-input messages and
// remote inputs. Blocks are every GSEControl keyed by (ied, ldInst, cbName).
type Declarations struct {
	IEDs          *pyjson.Object
	Blocks        []BlockEntry
	Subscriptions *pyjson.Object
	Problems      *pyjson.Object
	Remote        *pyjson.Object
}

// BlockEntry is one item of the Python blocks dict: the key tuple and the block.
type BlockEntry struct {
	Key   []string
	Block *pyjson.Object
}

// DeclarationsFromJSON reads the JSON layout of declarations: the Python dict with
// "blocks" written as [[key list, block], ...] because JSON has no tuple keys.
// Missing parts read as empty.
func DeclarationsFromJSON(value *pyjson.Object) Declarations {
	result := Declarations{IEDs: pyv.OrObj(value, "ieds"), Subscriptions: pyv.OrObj(value, "subscriptions"),
		Problems: pyv.OrObj(value, "problems"), Remote: pyv.OrObj(value, "remote")}
	for _, item := range pyv.GetList(value, "blocks") {
		pair := pyv.List(item)
		if len(pair) != 2 {
			continue
		}
		var key []string
		for _, part := range pyv.List(pair[0]) {
			key = append(key, pyv.PyStr(part))
		}
		result.Blocks = append(result.Blocks, BlockEntry{Key: key, Block: pyv.Obj(pair[1])})
	}
	return result
}

// sortedBlocks is sorted(declarations["blocks"].items()): by key tuple, lexicographic.
func sortedBlocks(blocks []BlockEntry) []BlockEntry {
	result := append([]BlockEntry(nil), blocks...)
	sort.SliceStable(result, func(i, j int) bool { return tupleLess(result[i].Key, result[j].Key) })
	return result
}

// tupleLess compares string tuples like Python: element by element, shorter first on a tie.
func tupleLess(left, right []string) bool {
	for index := 0; index < len(left) && index < len(right); index++ {
		if left[index] != right[index] {
			return left[index] < right[index]
		}
	}
	return len(left) < len(right)
}

// sortedKeys returns the keys of an object sorted by code point, like sorted(dict).
func sortedKeys(object *pyjson.Object) []string {
	if object == nil {
		return nil
	}
	keys := object.Keys()
	sort.Strings(keys)
	return keys
}

// stringSet is a Python set of names.
type stringSet map[string]bool

func (s stringSet) sorted() []string {
	names := make([]string, 0, len(s))
	for name := range s {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
