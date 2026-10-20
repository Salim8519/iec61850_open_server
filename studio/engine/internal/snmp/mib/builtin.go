package mib

import (
	"embed"
	"errors"
	"io/fs"
	"sort"
	"strings"
	"sync"
)

// builtinFS holds the verbatim IETF/IANA modules, one module per file (see builtin/README.md).
//
//go:embed builtin/*.txt
var builtinFS embed.FS

var (
	builtinOnce    sync.Once
	builtinModules map[string]*module
	builtinIndex   *index
	builtinErr     error
)

// BuiltinNames lists the module names shipped built in, sorted.
func BuiltinNames() []string {
	entries, err := fs.ReadDir(builtinFS, "builtin")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".txt") {
			names = append(names, strings.TrimSuffix(e.Name(), ".txt"))
		}
	}
	sort.Strings(names)
	return names
}

// Standard returns a new library with every built-in module loaded and resolved. The
// built-in files are parsed once per process; each call returns an independent library.
func Standard() (*Library, error) {
	builtinOnce.Do(loadBuiltins)
	l := &Library{modules: make(map[string]*module, len(builtinModules)), idx: builtinIndex}
	for name, m := range builtinModules {
		l.modules[name] = m
	}
	return l, builtinErr
}

func loadBuiltins() {
	builtinModules = map[string]*module{}
	var errs []error
	for _, name := range BuiltinNames() {
		text, err := builtinFS.ReadFile("builtin/" + name + ".txt")
		if err != nil {
			errs = append(errs, err)
			continue
		}
		mods := parseModules(name+".txt", string(text), true)
		if len(mods) == 0 {
			errs = append(errs, errors.New("built-in "+name+".txt holds no MIB module"))
		}
		for _, m := range mods {
			builtinModules[m.name] = m
		}
	}
	builtinIndex = build(builtinModules)
	builtinErr = errors.Join(errs...)
}
