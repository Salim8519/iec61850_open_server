package mib

import "strconv"

// macroValue parses "name KEYWORD clauses ::= value" for OBJECT-TYPE, NOTIFICATION-TYPE,
// TRAP-TYPE, MODULE-IDENTITY and the other macros that register an OID.
func (p *parser) macroValue(name, kw string, line int) {
	d := &def{name: name, kind: macroKeywords[kw], line: line}
	enterprise := ""
	for {
		if p.pos >= len(p.toks) || p.isStart(p.pos, true) || p.isUnknownStart(p.pos) || p.peek(0) == "END" {
			p.mod.problem(line, name+" "+kw+" has no \"::=\" value; skipped")
			return
		}
		t := p.peek(0)
		if t == "::=" {
			p.pos++
			break
		}
		switch t {
		case "SYNTAX":
			p.pos++
			s, table := p.parseType()
			if d.syntax == nil {
				d.syntax, d.table = &s, table
			}
		case "MAX-ACCESS", "ACCESS":
			if d.access == "" {
				d.access = p.peek(1)
			}
			p.pos += 2
		case "UNITS":
			d.units = p.text(1)
			p.pos += 2
		case "DESCRIPTION":
			if d.descr == "" {
				d.descr = collapse(p.text(1))
			}
			p.pos += 2
		case "INDEX":
			p.pos++
			d.indexes = p.nameList()
		case "AUGMENTS":
			p.pos++
			if l := p.nameList(); len(l) > 0 {
				d.augments = l[0]
			}
		case "OBJECTS", "VARIABLES":
			p.pos++
			d.objects = p.nameList()
		case "ENTERPRISE":
			enterprise = p.peek(1)
			p.pos += 2
		case "{", "(", "[":
			p.skipGroup()
		default:
			p.pos++
		}
	}
	if kw == "TRAP-TYPE" {
		num := p.peek(0)
		if _, err := strconv.ParseUint(num, 10, 32); err != nil || enterprise == "" {
			p.mod.problem(line, "TRAP-TYPE "+name+" needs an ENTERPRISE and a trap number")
			p.skipToStart()
			return
		}
		p.pos++
		d.value = oidValue{base: enterprise, arcs: []string{"0", num}}
	} else {
		v, ok := p.oidValue()
		if !ok {
			return
		}
		d.value = v
	}
	p.mod.addDef(d)
}

// nameList reads "{ a, IMPLIED b }" and returns the names.
func (p *parser) nameList() []string {
	if p.peek(0) != "{" {
		if p.pos < len(p.toks) && p.toks[p.pos].kind == tokIdent {
			p.pos++
			return []string{p.peek(-1)}
		}
		return nil
	}
	var names []string
	for p.pos++; p.pos < len(p.toks) && p.peek(0) != "}"; p.pos++ {
		if p.isStart(p.pos, true) {
			p.mod.problem(p.line(), "list is not closed with \"}\"")
			return names
		}
		t := p.toks[p.pos]
		if t.kind == tokIdent && t.text != "IMPLIED" {
			names = append(names, t.text)
		}
	}
	p.pos++
	return names
}

// typeAssignment parses "Name ::= TEXTUAL-CONVENTION ... SYNTAX type" or "Name ::= type".
func (p *parser) typeAssignment(name string, line int) {
	td := &typeDef{line: line}
	if p.peek(0) == "TEXTUAL-CONVENTION" {
		p.pos++
		for p.peek(0) != "SYNTAX" {
			if p.pos >= len(p.toks) || p.isStart(p.pos, false) || p.peek(0) == "END" {
				p.mod.problem(line, "textual convention "+name+" has no SYNTAX; skipped")
				return
			}
			if p.peek(0) == "DISPLAY-HINT" {
				td.hint = p.text(1)
				p.pos++
			}
			p.pos++
		}
		p.pos++
	}
	td.syntax, _ = p.parseType()
	if td.syntax.name == "" {
		p.mod.problem(line, "type "+name+" has no readable syntax")
		p.skipToStart()
		return
	}
	p.mod.types[name] = td
}

// parseType reads a type with optional named numbers and constraints. It reports whether
// the type is "SEQUENCE OF Entry" (a table).
func (p *parser) parseType() (syntaxRef, bool) {
	var s syntaxRef
	for p.peek(0) == "[" {
		p.skipGroup()
	}
	if p.peek(0) == "IMPLICIT" || p.peek(0) == "EXPLICIT" {
		p.pos++
	}
	switch t := p.peek(0); {
	case t == "SEQUENCE" && p.peek(1) == "OF":
		s.name = p.peek(2)
		p.pos += 3
		return s, true
	case t == "SEQUENCE" || t == "CHOICE":
		s.name = t
		p.pos++
		if p.peek(0) == "{" {
			p.skipGroup()
		}
		return s, false
	case t == "OCTET" && p.peek(1) == "STRING", t == "OBJECT" && p.peek(1) == "IDENTIFIER":
		s.name = t + " " + p.peek(1)
		p.pos += 2
	case p.pos < len(p.toks) && p.toks[p.pos].kind == tokIdent:
		s.name = t
		p.pos++
		if p.peek(0) == "." && isUpperIdent(p.peek(1)) {
			s.name = p.peek(1)
			p.pos += 2
		}
	default:
		p.mod.problem(p.line(), "expected a type but found \""+t+"\"")
		return s, false
	}
	if p.peek(0) == "{" {
		s.enums = p.namedNumbers()
	}
	for p.peek(0) == "(" {
		p.skipGroup()
	}
	return s, false
}

// namedNumbers reads "{ up(1), down(2) }".
func (p *parser) namedNumbers() map[int64]string {
	enums := map[int64]string{}
	for p.pos++; p.pos < len(p.toks) && p.peek(0) != "}"; {
		if p.isStart(p.pos, true) {
			p.mod.problem(p.line(), "named numbers are not closed with \"}\"")
			return enums
		}
		if p.toks[p.pos].kind == tokIdent && p.peek(1) == "(" && p.peek(3) == ")" {
			if n, err := strconv.ParseInt(p.peek(2), 10, 64); err == nil {
				enums[n] = p.peek(0)
			}
			p.pos += 4
			continue
		}
		p.pos++
	}
	p.pos++
	return enums
}
