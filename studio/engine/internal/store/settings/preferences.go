// Package settings owns <data>/preferences.json and <data>/tool-settings.json: the file
// part and locking of studio_app.projects.preferences.PreferenceStore (validation lives in
// internal/project and is passed in) and a full port of studio_app.tools.tool_memory.
package settings

import (
	"path/filepath"
	"sync"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/jsonfile"
)

// PreferencesName is the preferences file inside the data folder.
const PreferencesName = "preferences.json"

// UpdateFunc checks changes against the stored document (nil when missing or unreadable)
// and returns the document to write; project.UpdatePreferences has this shape.
type UpdateFunc func(stored, changes any) (*pyjson.Object, error)

// Preferences is the preferences.json file store.
type Preferences struct {
	path string
	mu   sync.Mutex
}

// NewPreferences returns the store of <dataDir>/preferences.json.
func NewPreferences(dataDir string) *Preferences {
	return &Preferences{path: filepath.Join(dataDir, PreferencesName)}
}

// Load returns the decoded file, or nil when it is missing or unreadable (Python's
// `stored = {}` path); project.Preferences turns it into the GET answer.
func (p *Preferences) Load() any {
	value, ok := jsonfile.Read(p.path)
	if !ok {
		return nil
	}
	return value
}

// Update runs update on the stored document and writes its result under the store lock
// (indent=2, CRLF, preferences.json.tmp then replace). It returns the document read back
// afterwards, as PreferenceStore.update returns self.get(). A validation error writes nothing.
func (p *Preferences) Update(changes any, update UpdateFunc) (any, error) {
	p.mu.Lock()
	next, err := update(p.Load(), changes)
	if err == nil {
		err = jsonfile.Write(p.path, next)
	}
	p.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return p.Load(), nil
}
