package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"studio-engine/internal/apperr"
	"studio-engine/internal/snmp/mib"
)

// maxMIBFile is the largest MIB file accepted.
const maxMIBFile = 5 << 20

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// Library is the MIB library of this PC (ADR-0092): the built-in modules minus the ones the
// user removed, plus the files the user imported, kept in <data>/snmp/mibs.
type Library struct {
	dir  string
	logf func(string, ...any)

	mu     sync.Mutex
	loaded *mib.Library
}

// NewLibrary returns the library; it is read from disk on first use.
func NewLibrary(dataDir string, logf func(string, ...any)) *Library {
	return &Library{dir: filepath.Join(dataDir, "snmp", "mibs"), logf: logf}
}

func (l *Library) removedPath() string { return filepath.Join(l.dir, "removed.json") }

func (l *Library) removed() map[string]bool {
	names := []string{}
	if data, err := os.ReadFile(l.removedPath()); err == nil {
		_ = json.Unmarshal(data, &names)
	}
	set := map[string]bool{}
	for _, name := range names {
		set[name] = true
	}
	return set
}

func (l *Library) saveRemoved(set map[string]bool) error {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	data, _ := json.Marshal(names)
	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(l.removedPath(), data, 0o644)
}

// Lib is the loaded library (loading it the first time).
func (l *Library) Lib() *mib.Library {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.loaded == nil {
		l.loaded = l.load()
	}
	return l.loaded
}

func (l *Library) load() *mib.Library {
	library, err := mib.Standard()
	if err != nil {
		l.logf("Built-in MIBs: %v", err)
		library = mib.NewLibrary()
	}
	for name := range l.removed() {
		library.Remove(name)
	}
	files, _ := os.ReadDir(l.dir)
	for _, file := range files {
		if file.IsDir() || file.Name() == "removed.json" {
			continue
		}
		text, err := os.ReadFile(filepath.Join(l.dir, file.Name()))
		if err == nil {
			_, err = library.Load(file.Name(), text, false)
		}
		if err != nil {
			l.logf("MIB file %s: %v", file.Name(), err)
		}
	}
	return library
}

// ModuleRow is one module in the page's MIB library list.
type ModuleRow struct {
	Name     string   `json:"name"`
	File     string   `json:"file"`
	Builtin  bool     `json:"builtin"`
	Nodes    int      `json:"nodes"`
	Missing  []string `json:"missing"`
	Problems []string `json:"problems"`
}

// Modules lists every loaded module.
func (l *Library) Modules() []ModuleRow {
	modules := l.Lib().Modules()
	rows := make([]ModuleRow, len(modules))
	for index, info := range modules {
		rows[index] = ModuleRow{Name: info.Name, File: info.File, Builtin: info.Builtin, Nodes: info.Nodes,
			Missing: append([]string{}, info.Missing...), Problems: append([]string{}, info.Problems...)}
	}
	return rows
}

// Import adds a MIB file; it is kept for every project on this PC.
func (l *Library) Import(name string, text []byte) ([]string, error) {
	if len(text) > maxMIBFile {
		return nil, apperr.ValueError("The MIB file is larger than 5 MB")
	}
	file := unsafeName.ReplaceAllString(filepath.Base(name), "_")
	if file == "" || file == "." || file == "removed.json" {
		file = "imported.mib"
	}
	library := l.Lib()
	modules, err := library.Load(file, text, false)
	if err != nil {
		return nil, apperr.ValueError(err.Error())
	}
	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(l.dir, file), text, 0o644); err != nil {
		return nil, fmt.Errorf("the MIB file could not be saved: %v", err)
	}
	set := l.removed()
	for _, module := range modules {
		delete(set, module)
	}
	return modules, l.saveRemoved(set)
}

// Remove takes a module out: a built-in one is remembered as removed, an imported file is deleted.
func (l *Library) Remove(module string) error {
	library := l.Lib()
	var info *mib.ModuleInfo
	for _, item := range library.Modules() {
		if item.Name == module {
			info = &item
			break
		}
	}
	if info == nil {
		return apperr.ValueError(fmt.Sprintf("There is no MIB module %s", module))
	}
	if info.Builtin {
		set := l.removed()
		set[module] = true
		if err := l.saveRemoved(set); err != nil {
			return err
		}
	} else if info.File != "" && !strings.ContainsAny(info.File, `/\`) {
		if err := os.Remove(filepath.Join(l.dir, info.File)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	l.mu.Lock()
	l.loaded = l.load()
	l.mu.Unlock()
	return nil
}

// Restore brings back every removed built-in module.
func (l *Library) Restore() error {
	if err := l.saveRemoved(map[string]bool{}); err != nil {
		return err
	}
	l.mu.Lock()
	l.loaded = l.load()
	l.mu.Unlock()
	return nil
}

// Name is "MODULE::name.instance", or the dotted OID when no MIB knows it.
func (l *Library) Name(oid string) string {
	if oid == "" {
		return ""
	}
	return l.Lib().Name(oid)
}
