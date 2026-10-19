package xsd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// schemaDigests computes, once per schema folder, the SHA-256 over its .xsd files:
// for each file in byte order of its name, "<name>\n<size>\n<bytes>". Other files of
// the folder (documentation zip, history, manifest) do not affect validation.
type schemaDigests struct {
	root   string
	mu     sync.Mutex
	values map[string]string
}

func newSchemaDigests(root string) *schemaDigests {
	return &schemaDigests{root: root, values: map[string]string{}}
}

func (d *schemaDigests) get(schema string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if value, found := d.values[schema]; found {
		return value, nil
	}
	value, err := digestFolder(filepath.Join(d.root, schema))
	if err != nil {
		return "", errSchemaFiles(schema, err)
	}
	d.values[schema] = value
	return value, nil
}

func digestFolder(folder string) (string, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return "", err
	}
	var names []string
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.EqualFold(filepath.Ext(entry.Name()), ".xsd") {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no .xsd files in %s", folder)
	}
	sort.Strings(names)
	hash := sha256.New()
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(folder, name))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(hash, "%s\n%d\n", name, len(data))
		hash.Write(data)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
