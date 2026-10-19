package netcfg

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"studio-engine/internal/pyjson"
)

// sortedCopy returns value with every object's keys in sorted order, so pyjson.Dumps writes
// what json.dumps(value, sort_keys=True) writes.
func sortedCopy(value any) any {
	switch v := value.(type) {
	case []any:
		items := make([]any, len(v))
		for index, item := range v {
			items[index] = sortedCopy(item)
		}
		return items
	case *pyjson.Object:
		keys := v.Keys()
		sort.Strings(keys)
		object := pyjson.NewObject()
		for _, key := range keys {
			item, _ := v.Get(key)
			object.Set(key, sortedCopy(item))
		}
		return object
	}
	return value
}

// dumpsSorted is json.dumps(value, sort_keys=True) with default separators.
func dumpsSorted(value any) ([]byte, error) {
	return pyjson.Dumps(sortedCopy(value))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// sortedFingerprint is hashlib.sha256(json.dumps(value, sort_keys=True).encode()).hexdigest().
func sortedFingerprint(value any) (string, error) {
	data, err := dumpsSorted(value)
	if err != nil {
		return "", err
	}
	return sha256Hex(data), nil
}
