package mib

import (
	"sort"
	"strconv"
	"strings"
)

// index is the resolved view of every loaded module. It is rebuilt after each Load or
// Remove and never changed afterwards, so readers can share it.
type index struct {
	nodes    map[string]*Node
	rank     map[string]int
	byName   map[string]string
	nameRank map[string]int
	byModule map[string]map[string]string
	children map[string][]*Node
	info     map[string]*ModuleInfo
}

type resolver struct {
	mods       map[string]*module
	order      []*module
	owners     map[string][]*module
	typeOwners map[string][]*module
	memo       map[*def]string
	busy       map[*def]bool
	idx        *index
}

// build resolves every module: OIDs across modules in any order, types, kinds and the tree.
func build(mods map[string]*module) *index {
	r := &resolver{mods: mods, owners: map[string][]*module{}, typeOwners: map[string][]*module{},
		memo: map[*def]string{}, busy: map[*def]bool{}}
	r.idx = &index{nodes: map[string]*Node{}, rank: map[string]int{}, byName: map[string]string{},
		nameRank: map[string]int{}, byModule: map[string]map[string]string{},
		children: map[string][]*Node{}, info: map[string]*ModuleInfo{}}
	for _, m := range mods {
		r.order = append(r.order, m)
	}
	sort.Slice(r.order, func(i, j int) bool { return r.order[i].name < r.order[j].name })
	for _, m := range r.order {
		for _, d := range m.defs {
			r.owners[d.name] = append(r.owners[d.name], m)
		}
		for name := range m.types {
			r.typeOwners[name] = append(r.typeOwners[name], m)
		}
	}
	for name, oid := range roots {
		r.place("", name, -1, &Node{OID: oid, Name: name, Kind: "node"})
	}
	var placed []placedDef
	for _, m := range r.order {
		r.idx.info[m.name] = r.moduleInfo(m)
		for _, d := range m.defs {
			if n := r.node(m, d); n != nil {
				r.place(m.name, d.name, rankOf(m), n)
				r.idx.info[m.name].Nodes++
				placed = append(placed, placedDef{m, d, n})
			}
		}
	}
	r.finish(placed)
	return r.idx
}

type placedDef struct {
	m *module
	d *def
	n *Node
}

func rankOf(m *module) int {
	if m.v1 {
		return 0
	}
	return 1
}

func (r *resolver) moduleInfo(m *module) *ModuleInfo {
	info := &ModuleInfo{Name: m.name, File: m.file, Builtin: m.builtin}
	info.Problems = append(info.Problems, m.problems...)
	seen := map[string]bool{}
	for _, from := range m.from {
		if r.mods[from] == nil && !seen[from] {
			seen[from] = true
			info.Missing = append(info.Missing, from)
		}
	}
	return info
}

// place records a node; an SMIv2 definition replaces an SMIv1 one of the same OID.
func (r *resolver) place(mod, name string, rank int, n *Node) {
	if old, ok := r.idx.nodes[n.OID]; !ok || rank > r.idx.rank[n.OID] {
		r.idx.nodes[n.OID], r.idx.rank[n.OID] = n, rank
	} else if old.Kind == "node" && n.Kind != "node" && rank == r.idx.rank[n.OID] {
		r.idx.nodes[n.OID] = n
	}
	if mod != "" {
		if r.idx.byModule[mod] == nil {
			r.idx.byModule[mod] = map[string]string{}
		}
		r.idx.byModule[mod][name] = n.OID
	}
	if _, ok := r.idx.byName[name]; !ok || rank > r.idx.nameRank[name] {
		r.idx.byName[name], r.idx.nameRank[name] = n.OID, rank
	}
}

// oidOf resolves a symbolic name as seen from module m.
func (r *resolver) oidOf(m *module, name string, depth int) (string, bool) {
	if depth > 64 {
		return "", false
	}
	if d := m.byName[name]; d != nil {
		return r.defOID(m, d, depth)
	}
	if from, ok := m.imports[name]; ok {
		if fm := r.mods[from]; fm != nil && fm.byName[name] != nil {
			return r.defOID(fm, fm.byName[name], depth)
		}
	}
	if oid, ok := roots[name]; ok {
		return oid, true
	}
	for _, om := range r.owners[name] {
		return r.defOID(om, om.byName[name], depth)
	}
	return "", false
}

func (r *resolver) defOID(m *module, d *def, depth int) (string, bool) {
	if oid, ok := r.memo[d]; ok {
		return oid, oid != ""
	}
	if r.busy[d] {
		return "", false
	}
	r.busy[d] = true
	defer delete(r.busy, d)
	parts := append([]string(nil), d.value.arcs...)
	if d.value.base != "" {
		base, ok := r.oidOf(m, d.value.base, depth+1)
		if !ok {
			r.memo[d] = ""
			return "", false
		}
		parts = append([]string{base}, parts...)
	}
	oid := strings.Join(parts, ".")
	r.memo[d] = oid
	return oid, oid != ""
}

// node builds the public node of a definition, or records why it cannot be placed.
func (r *resolver) node(m *module, d *def) *Node {
	oid, ok := r.defOID(m, d, 0)
	if !ok {
		if owner, base := r.unresolved(m, d); !r.fromMissing(owner, base) {
			r.problem(m, d.line, "cannot place "+d.name+": \""+base+"\" is not defined in any loaded MIB")
		}
		return nil
	}
	n := &Node{OID: oid, Name: d.name, Module: m.name, Kind: d.kind, Access: d.access,
		Units: d.units, Description: d.descr, Indexes: d.indexes, Objects: d.objects}
	if d.kind != "object" {
		return n
	}
	switch {
	case d.table:
		n.Kind = "table"
	case len(d.indexes) > 0 || d.augments != "":
		n.Kind = "row"
	case d.syntax != nil:
		info, ok := r.typeOf(m, *d.syntax, 0)
		if !ok && !r.fromMissing(m, d.syntax.name) {
			r.problem(m, d.line, "type "+d.syntax.name+" of "+d.name+" is not defined in any loaded MIB")
		}
		if info.base == sequenceType {
			n.Kind = "row"
		} else {
			n.Syntax, n.Hex, n.Enums = info.base, info.hex, info.enums
		}
	}
	return n
}

// unresolved follows a definition's parents within its module to the first name that is
// not defined there, so a missing import is not reported twice.
func (r *resolver) unresolved(m *module, d *def) (*module, string) {
	base := d.value.base
	for i := 0; i < 64 && m.byName[base] != nil; i++ {
		base = m.byName[base].value.base
	}
	return m, base
}

func (r *resolver) problem(m *module, line int, text string) {
	info := r.idx.info[m.name]
	info.Problems = append(info.Problems, "line "+strconv.Itoa(line)+": "+text)
}

// finish sets columns, AUGMENTS indexes and the parent/child tree.
func (r *resolver) finish(placed []placedDef) {
	for _, p := range placed {
		if p.d.augments != "" {
			if oid, ok := r.oidOf(p.m, p.d.augments, 0); ok && r.idx.nodes[oid] != nil {
				p.n.Indexes = r.idx.nodes[oid].Indexes
			}
		}
	}
	for oid, n := range r.idx.nodes {
		if n.Kind == "object" {
			if parent := r.idx.nodes[parentOID(oid)]; parent != nil && parent.Kind == "row" {
				n.Kind = "column"
			}
		}
		for p := parentOID(oid); p != ""; p = parentOID(p) {
			if r.idx.nodes[p] != nil {
				r.idx.children[p] = append(r.idx.children[p], n)
				break
			}
		}
	}
	for _, list := range r.idx.children {
		sort.Slice(list, func(i, j int) bool { return CompareOID(list[i].OID, list[j].OID) < 0 })
	}
}

// parentOID drops the last arc ("" for a single arc).
func parentOID(oid string) string {
	if i := strings.LastIndexByte(oid, '.'); i > 0 {
		return oid[:i]
	}
	return ""
}
