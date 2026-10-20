package mib

import (
	"strconv"
	"strings"
)

// module is one parsed MIB module. It is never changed after parsing, so several
// libraries can share it.
type module struct {
	name     string
	file     string
	builtin  bool
	v1       bool              // SMIv1 module; SMIv2 definitions of the same OID win
	imports  map[string]string // symbol -> module it comes from
	from     []string          // imported module names, in order
	defs     []*def            // definitions that carry an OID, in order
	byName   map[string]*def
	types    map[string]*typeDef
	problems []string
}

// oidValue is "{ base 1 2 }": a symbolic base (empty for an absolute value) and arcs.
type oidValue struct {
	base string
	arcs []string
}

// syntaxRef is the SYNTAX of an object or type: a type name and any named numbers.
type syntaxRef struct {
	name  string
	enums map[int64]string
}

// def is one definition with an OID value.
type def struct {
	name     string
	kind     string // node, object or notification
	line     int
	value    oidValue
	syntax   *syntaxRef
	table    bool // SYNTAX SEQUENCE OF
	access   string
	units    string
	descr    string
	indexes  []string
	augments string
	objects  []string
}

// typeDef is a TEXTUAL-CONVENTION or a plain type assignment.
type typeDef struct {
	syntax syntaxRef
	hint   string
	line   int
}

func newModule(name, file string, builtin bool) *module {
	return &module{name: name, file: file, builtin: builtin, imports: map[string]string{},
		byName: map[string]*def{}, types: map[string]*typeDef{}}
}

func (m *module) addDef(d *def) {
	if _, dup := m.byName[d.name]; dup {
		m.problem(d.line, d.name+" is defined twice; the later definition is used")
		for i, old := range m.defs {
			if old.name == d.name {
				m.defs = append(m.defs[:i], m.defs[i+1:]...)
				break
			}
		}
	}
	m.defs = append(m.defs, d)
	m.byName[d.name] = d
}

func (m *module) problem(line int, text string) {
	m.problems = append(m.problems, "line "+strconv.Itoa(line)+": "+text)
}

// collapse joins white space and keeps a description to at most 300 characters.
func collapse(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > 300 {
		return string(r[:297]) + "..."
	}
	return s
}
