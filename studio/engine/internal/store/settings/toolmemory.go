package settings

import (
	"path/filepath"
	"sync"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/jsonfile"
)

// ToolSettingsName is the tool memory file inside the data folder.
const ToolSettingsName = "tool-settings.json"

// ToolMemory is tool_memory.ToolMemory: what the user last did in each tool, checked on
// read so a damaged file only resets the tool memory.
type ToolMemory struct {
	path string
	mu   sync.Mutex
}

// NewToolMemory returns the store of <dataDir>/tool-settings.json.
func NewToolMemory(dataDir string) *ToolMemory {
	return &ToolMemory{path: filepath.Join(dataDir, ToolSettingsName)}
}

// Get returns the cleaned settings {"explorer": {...}, "find": {...}}; a missing or
// damaged file gives the defaults, never an error.
func (m *ToolMemory) Get() *pyjson.Object {
	value, ok := jsonfile.Read(m.path)
	if !ok {
		return clean(nil)
	}
	return clean(value)
}

// Update is update(tool, **changes): changes are merged into the tool's section, the whole
// memory is cleaned and written (indent=2, CRLF, tool-settings.json.tmp then replace), and
// the cleaned section is returned. An unknown tool is Python's KeyError.
func (m *ToolMemory) Update(tool string, changes *pyjson.Object) (*pyjson.Object, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current := m.Get()
	section, ok := current.Get(tool)
	if !ok {
		return nil, apperr.KeyError(tool)
	}
	merged := section.(*pyjson.Object)
	for _, key := range changes.Keys() {
		value, _ := changes.Get(key)
		merged.Set(key, value)
	}
	cleaned := clean(current)
	if err := jsonfile.Write(m.path, cleaned); err != nil {
		return nil, err
	}
	result, _ := cleaned.Get(tool)
	return result.(*pyjson.Object), nil
}
