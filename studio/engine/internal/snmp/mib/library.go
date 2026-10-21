// Package mib reads SMIv1/SMIv2 MIB modules and turns OIDs into names and back (ADR-0092).
// Standard IETF/IANA modules are built in; users can load vendor MIB files on top.
package mib

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Node is one named OID of a loaded module. Nodes are shared: callers must not change them.
type Node struct {
	OID         string           `json:"oid"`                   // dotted numeric, "1.3.6.1.2.1.2.2.1.8"
	Name        string           `json:"name"`                  // "ifOperStatus"
	Module      string           `json:"module"`                // "IF-MIB"; "" for a well-known root
	Kind        string           `json:"kind"`                  // node, object, table, row, column, notification
	Syntax      string           `json:"syntax,omitempty"`      // base point type (model.Type*) of objects
	Hex         bool             `json:"hex,omitempty"`         // the value is best shown as hex
	Enums       map[int64]string `json:"enums,omitempty"`       // named numbers (and named bits of BITS)
	Access      string           `json:"access,omitempty"`      // read-only, read-write, read-create, ...
	Units       string           `json:"units,omitempty"`       // UNITS clause
	Description string           `json:"description,omitempty"` // white space collapsed, at most 300 characters
	Indexes     []string         `json:"indexes,omitempty"`     // for rows: index object names
	Objects     []string         `json:"objects,omitempty"`     // for notifications: OBJECTS names
}

// ModuleInfo describes one loaded module.
type ModuleInfo struct {
	Name     string   `json:"name"`
	File     string   `json:"file"`
	Builtin  bool     `json:"builtin"`
	Nodes    int      `json:"nodes"`
	Missing  []string `json:"missing,omitempty"`  // imported modules that are not loaded
	Problems []string `json:"problems,omitempty"` // definitions that could not be read or placed
}

// Library holds loaded modules and their resolved names. It is safe for concurrent use.
type Library struct {
	mu      sync.RWMutex
	modules map[string]*module
	idx     *index
}

// NewLibrary returns an empty library that still knows the well-known roots (iso, internet,
// enterprises, ...).
func NewLibrary() *Library {
	l := &Library{modules: map[string]*module{}}
	l.idx = build(l.modules)
	return l
}

// Load parses a MIB file, which may hold several modules, and adds (or replaces) its
// modules. It returns the module names. It fails only when the file holds no module;
// problems inside a module are listed in Modules().
func (l *Library) Load(file string, text []byte, builtin bool) ([]string, error) {
	src := strings.TrimPrefix(string(text), "\ufeff")
	mods := parseModules(file, src, builtin)
	if len(mods) == 0 {
		return nil, errors.New(strconv.Quote(file) +
			` holds no MIB module (expected "NAME DEFINITIONS ::= BEGIN")`)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	names := make([]string, 0, len(mods))
	for _, m := range mods {
		l.modules[m.name] = m
		names = append(names, m.name)
	}
	l.idx = build(l.modules)
	return names, nil
}

// Remove unloads a module. It reports whether the module was loaded.
func (l *Library) Remove(module string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.modules[module]; !ok {
		return false
	}
	delete(l.modules, module)
	l.idx = build(l.modules)
	return true
}

// Modules lists the loaded modules sorted by name.
func (l *Library) Modules() []ModuleInfo {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]ModuleInfo, 0, len(l.idx.info))
	for _, info := range l.idx.info {
		c := *info
		c.Missing = append([]string(nil), info.Missing...)
		c.Problems = append([]string(nil), info.Problems...)
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// current returns the resolved view of the loaded modules.
func (l *Library) current() *index {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.idx
}
