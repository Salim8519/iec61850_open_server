package mib

import (
	"strconv"
	"strings"
)

// macroKeywords start a definition: "name KEYWORD clauses ::= value".
var macroKeywords = map[string]string{
	"MODULE-IDENTITY": "node", "OBJECT-IDENTITY": "node", "OBJECT-TYPE": "object",
	"NOTIFICATION-TYPE": "notification", "TRAP-TYPE": "notification",
	"OBJECT-GROUP": "node", "NOTIFICATION-GROUP": "node", "MODULE-COMPLIANCE": "node",
	"AGENT-CAPABILITIES": "node",
}

// clauseKeywords may follow an object name inside a definition without starting one.
var clauseKeywords = map[string]bool{
	"MIN-ACCESS": true, "WRITE-SYNTAX": true, "MAX-ACCESS": true, "DISPLAY-HINT": true,
	"LAST-UPDATED": true, "CONTACT-INFO": true, "MANDATORY-GROUPS": true,
	"PRODUCT-RELEASE": true, "CREATION-REQUIRES": true,
}

type parser struct {
	toks []token
	pos  int
	mod  *module
}

// parseModules finds every "NAME DEFINITIONS ::= BEGIN ... END" module in a file.
func parseModules(file, text string, builtin bool) []*module {
	p := &parser{toks: lex(text)}
	var mods []*module
	for p.pos < len(p.toks) {
		if !p.atHeader() {
			p.pos++
			continue
		}
		p.mod = newModule(p.toks[p.pos].text, file, builtin)
		for p.pos < len(p.toks) && p.peek(0) != "BEGIN" {
			p.pos++
		}
		p.pos++
		p.body()
		p.mod.v1 = p.mod.name == "RFC1155-SMI" || p.mod.name == "RFC-1212" || p.mod.name == "RFC-1215"
		for _, from := range p.mod.from {
			p.mod.v1 = p.mod.v1 || from == "RFC1155-SMI"
		}
		mods = append(mods, p.mod)
	}
	return mods
}

// atHeader reports whether the parser stands on "NAME DEFINITIONS ... ::= BEGIN".
func (p *parser) atHeader() bool {
	if p.toks[p.pos].kind != tokIdent || p.peek(1) != "DEFINITIONS" {
		return false
	}
	for i := p.pos + 2; i+1 < len(p.toks) && i < p.pos+8; i++ {
		if p.toks[i].text == "::=" {
			return p.toks[i+1].text == "BEGIN"
		}
	}
	return false
}

// peek returns the text of a token for keyword matching; quoted strings read as "".
func (p *parser) peek(n int) string {
	if i := p.pos + n; i >= 0 && i < len(p.toks) && p.toks[i].kind != tokString {
		return p.toks[i].text
	}
	return ""
}

// text returns the text of any token, quoted strings included.
func (p *parser) text(n int) string {
	if i := p.pos + n; i >= 0 && i < len(p.toks) {
		return p.toks[i].text
	}
	return ""
}

func (p *parser) line() int {
	if p.pos < len(p.toks) {
		return p.toks[p.pos].line
	}
	if len(p.toks) > 0 {
		return p.toks[len(p.toks)-1].line
	}
	return 1
}

// body parses assignments until the module's END.
func (p *parser) body() {
	for p.pos < len(p.toks) {
		switch t := p.peek(0); {
		case t == "END":
			p.pos++
			return
		case t == "IMPORTS":
			p.pos++
			p.imports()
		case t == "EXPORTS":
			for p.pos < len(p.toks) && p.peek(0) != ";" {
				p.pos++
			}
			p.pos++
		case p.isStart(p.pos, false):
			p.statement()
		case p.isUnknownStart(p.pos):
			p.mod.problem(p.line(), "\""+t+" "+p.peek(1)+"\" is not a construct this reader knows; skipped")
			p.skipToStart()
		default:
			p.mod.problem(p.line(), "unexpected \""+t+"\"; skipped to the next definition")
			p.skipToStart()
		}
	}
	p.mod.problem(p.line(), "module "+p.mod.name+" has no END")
}

// isStart reports whether token i begins a top-level assignment. Inside a definition
// (clause true) "Name ::=" is not a start, because the definition ends with "::=" itself.
func (p *parser) isStart(i int, clause bool) bool {
	if i+1 >= len(p.toks) || p.toks[i].kind != tokIdent || p.toks[i+1].kind == tokString {
		return false
	}
	next := p.toks[i+1].text
	switch {
	case next == "MACRO" || macroKeywords[next] != "":
		return true
	case next == "::=":
		return !clause && isUpperIdent(p.toks[i].text)
	case next == "OBJECT":
		return i+3 < len(p.toks) && p.toks[i+2].text == "IDENTIFIER" && p.toks[i+3].text == "::="
	}
	return false
}

// isUnknownStart spots "name SOME-MACRO" constructs this parser does not know.
func (p *parser) isUnknownStart(i int) bool {
	if i+1 >= len(p.toks) || p.toks[i].kind != tokIdent || isUpperIdent(p.toks[i].text) {
		return false
	}
	next := p.toks[i+1].text
	if p.toks[i+1].kind != tokIdent || clauseKeywords[next] || !isUpperIdent(next) {
		return false
	}
	for j := 0; j < len(next); j++ {
		if next[j] >= 'a' && next[j] <= 'z' {
			return false
		}
	}
	return strings.IndexByte(next, '-') > 0
}

// skipToStart moves past broken input to the next definition or the module END.
func (p *parser) skipToStart() {
	p.pos++
	for p.pos < len(p.toks) {
		if p.peek(0) == "END" || p.isStart(p.pos, false) || p.isUnknownStart(p.pos) {
			return
		}
		p.pos++
	}
}

func (p *parser) imports() {
	var pending []string
	for p.pos < len(p.toks) {
		t := p.toks[p.pos]
		switch {
		case t.text == ";":
			p.pos++
			return
		case t.text == "FROM":
			from := p.peek(1)
			p.pos += 2
			for _, s := range pending {
				p.mod.imports[s] = from
			}
			pending = pending[:0]
			p.mod.from = append(p.mod.from, from)
			if p.peek(0) == "{" {
				p.skipGroup()
			}
		case p.isStart(p.pos, false):
			p.mod.problem(t.line, "IMPORTS has no closing \";\"")
			return
		case t.kind == tokIdent:
			pending = append(pending, t.text)
			p.pos++
		default:
			p.pos++
		}
	}
}

// statement parses one assignment starting at a name.
func (p *parser) statement() {
	name, line := p.peek(0), p.line()
	kw := p.peek(1)
	p.pos += 2
	switch {
	case kw == "MACRO":
		p.skipMacro()
	case kw == "::=":
		p.typeAssignment(name, line)
	case kw == "OBJECT":
		p.pos += 2
		if v, ok := p.oidValue(); ok {
			p.mod.addDef(&def{name: name, kind: "node", line: line, value: v})
		}
	default:
		p.macroValue(name, kw, line)
	}
}

// skipMacro skips "MACRO ::= BEGIN ... END" (nested BEGIN/END pairs included).
func (p *parser) skipMacro() {
	depth := 0
	for p.pos < len(p.toks) {
		switch p.peek(0) {
		case "BEGIN":
			depth++
		case "END":
			depth--
			if depth <= 0 {
				p.pos++
				return
			}
		}
		p.pos++
	}
}

// skipGroup skips a balanced {..}, (..) or [..] group starting at the current token.
func (p *parser) skipGroup() {
	depth := 0
	for p.pos < len(p.toks) {
		switch p.peek(0) {
		case "{", "(", "[":
			depth++
		case "}", ")", "]":
			depth--
		}
		p.pos++
		if depth <= 0 {
			return
		}
	}
}

// oidValue parses "{ base 1 name(2) }".
func (p *parser) oidValue() (oidValue, bool) {
	line := p.line()
	var v oidValue
	if p.peek(0) != "{" {
		p.mod.problem(line, "expected an OID value \"{ parent number }\" but found \""+p.peek(0)+"\"")
		p.skipToStart()
		return v, false
	}
	p.pos++
	for first := true; p.pos < len(p.toks) && p.peek(0) != "}"; first = false {
		t := p.toks[p.pos]
		switch {
		case t.kind == tokNumber:
			v.arcs = append(v.arcs, t.text)
			p.pos++
		case t.kind == tokIdent && p.peek(1) == "(" && p.peek(3) == ")":
			v.arcs = append(v.arcs, p.peek(2))
			p.pos += 4
		case t.kind == tokIdent && first:
			v.base = t.text
			p.pos++
		default:
			p.mod.problem(t.line, "cannot read OID value part \""+t.text+"\"")
			p.skipToStart()
			return v, false
		}
	}
	p.pos++
	for _, a := range v.arcs {
		if _, err := strconv.ParseUint(a, 10, 32); err != nil {
			p.mod.problem(line, "OID arc \""+a+"\" is not a number from 0 to 4294967295")
			return v, false
		}
	}
	return v, true
}
