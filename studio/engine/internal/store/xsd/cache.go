package xsd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// cache holds verdicts by cacheKey: in memory (LRU) and as <dir>/<sha256(key)>.json
// files, so a document validated once is never sent to the helper again, even after a
// restart. A file that cannot be read back exactly is a miss, never an error.
type cache struct {
	dir    string
	mu     sync.Mutex
	memory *lru
}

func newCache(dir string, memoryEntries int) *cache {
	return &cache{dir: dir, memory: newLRU(memoryEntries)}
}

// cacheKey = sha256(source) + NUL + schema + NUL + libxml2 version + NUL + schema digest:
// a new libxml2 or changed XSD files never reuse old verdicts.
func cacheKey(sourceSHA256, schema, schemaDigest string) string {
	return sourceSHA256 + "\x00" + schema + "\x00" + Libxml2Version + "\x00" + schemaDigest
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (c *cache) file(key string) string {
	return filepath.Join(c.dir, sha256Hex([]byte(key))+".json")
}

func (c *cache) get(key string) (scl.SchemaVerdict, bool) {
	c.mu.Lock()
	verdict, found := c.memory.get(key)
	c.mu.Unlock()
	if found {
		return verdict, true
	}
	data, err := os.ReadFile(c.file(key))
	if err != nil {
		return scl.SchemaVerdict{}, false
	}
	verdict, err = decodeVerdict(data)
	if err != nil {
		return scl.SchemaVerdict{}, false
	}
	c.mu.Lock()
	c.memory.put(key, verdict)
	c.mu.Unlock()
	return copyVerdict(verdict), true
}

// put stores the verdict in memory and on disk (temporary file + rename).
func (c *cache) put(key string, verdict scl.SchemaVerdict) error {
	c.mu.Lock()
	c.memory.put(key, verdict)
	c.mu.Unlock()
	data, err := encodeVerdict(verdict)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(c.dir, "verdict-*.tmp")
	if err != nil {
		return err
	}
	_, err = temp.Write(data)
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(temp.Name(), c.file(key))
	}
	if err != nil {
		_ = os.Remove(temp.Name())
	}
	return err
}

// encodeVerdict writes json.dumps({"status": ..., "errors": [{"line", "message"}]}).
func encodeVerdict(verdict scl.SchemaVerdict) ([]byte, error) {
	errors := make([]any, 0, len(verdict.Errors))
	for _, item := range verdict.Errors {
		errors = append(errors, pyjson.NewObject().Set("line", item.Line).Set("message", item.Message))
	}
	return pyjson.Dumps(pyjson.NewObject().Set("status", verdict.Status).Set("errors", errors))
}

func decodeVerdict(data []byte) (scl.SchemaVerdict, error) {
	value, err := pyjson.Decode(data)
	if err != nil {
		return scl.SchemaVerdict{}, err
	}
	object, ok := value.(*pyjson.Object)
	if !ok || object.Len() != 2 {
		return scl.SchemaVerdict{}, fmt.Errorf("verdict is not {status, errors}")
	}
	status, _ := object.Get("status")
	items, _ := object.Get("errors")
	return verdictFrom(status, items)
}

// verdictFrom checks decoded JSON values for a verdict: used by cache files and seeds.
func verdictFrom(status, items any) (scl.SchemaVerdict, error) {
	text, ok := status.(string)
	if !ok || (text != "valid" && text != "invalid") {
		return scl.SchemaVerdict{}, fmt.Errorf("status must be \"valid\" or \"invalid\"")
	}
	list, ok := items.([]any)
	if !ok || len(list) > maxErrors {
		return scl.SchemaVerdict{}, fmt.Errorf("errors must be a list of at most %d items", maxErrors)
	}
	verdict := scl.SchemaVerdict{Status: text, Errors: make([]scl.SchemaError, 0, len(list))}
	for _, item := range list {
		entry, ok := item.(*pyjson.Object)
		if !ok || entry.Len() != 2 {
			return scl.SchemaVerdict{}, fmt.Errorf("each error must be {line, message}")
		}
		lineValue, _ := entry.Get("line")
		messageValue, _ := entry.Get("message")
		line, okLine := intValue(lineValue)
		message, okMessage := messageValue.(string)
		if !okLine || !okMessage {
			return scl.SchemaVerdict{}, fmt.Errorf("each error must have an integer line and a text message")
		}
		verdict.Errors = append(verdict.Errors, scl.SchemaError{Line: line, Message: message})
	}
	return verdict, nil
}

func intValue(value any) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true
	case int64:
		return int(number), true
	case *big.Int:
		if number.IsInt64() {
			return int(number.Int64()), true
		}
	}
	return 0, false
}
