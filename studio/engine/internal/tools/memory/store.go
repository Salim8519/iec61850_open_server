package memory

import (
	"os"
	"path/filepath"
	"sync"
	"unicode/utf8"

	"studio-engine/internal/pyjson"
)

// FileName is the tool memory file in the data folder.
const FileName = "tool-settings.json"

// Store reads and writes the tool memory; a damaged file only resets the memory.
type Store struct {
	path string
	mu   sync.Mutex // serialises writers, never held by readers
}

// New returns the store of the data folder directory.
func New(directory string) *Store { return &Store{path: filepath.Join(directory, FileName)} }

// Get is ToolMemory.get: the cleaned file, or the defaults when it is missing or invalid.
func (s *Store) Get() *pyjson.Object {
	data, err := os.ReadFile(s.path)
	if err != nil || !utf8.Valid(data) {
		return Clean(nil)
	}
	value, err := pyjson.Decode(data)
	if err != nil {
		return Clean(nil)
	}
	return Clean(value)
}

// Update is ToolMemory.update(tool, **changes): merge, clean, write atomically, and
// return the cleaned section of tool ("explorer" or "find").
func (s *Store) Update(tool string, changes *pyjson.Object) (*pyjson.Object, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.Get()
	section, _ := current.Get(tool)
	merged := section.(*pyjson.Object)
	for _, key := range changes.Keys() {
		value, _ := changes.Get(key)
		merged.Set(key, value)
	}
	cleaned := Clean(current)
	encoded, err := Indent(cleaned)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return nil, err
	}
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, crlf(encoded), 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(temporary, s.path); err != nil {
		return nil, err
	}
	result, _ := cleaned.Get(tool)
	return result.(*pyjson.Object), nil
}
