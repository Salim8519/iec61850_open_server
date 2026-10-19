package xmldoc

import "studio-engine/internal/scl/pyrules"

// DOCTYPE handling. The tree never uses a DTD; what matters is accepting what expat
// accepts and failing like defusedxml: any ENTITY declaration (general or parameter)
// raises EntitiesForbidden when it is complete. Malformed internal subsets are reported
// as expat's syntax error at the offending token (a close approximation of its role
// machine; the SCL corpus has no other DTD content).

type ptok struct {
	kind       byte // 's' space, 'n' name, 'l' literal, '[', ']', '>', '%', '<' markup, 'x' other
	start, end int
	text       string
}

// ptoken reads one prolog token at p; ok=false with err for invalid input, EOF gives
// kind 0.
func (s *scanner) ptoken(p int) (ptok, error) {
	r, w, cls := s.at(p)
	switch {
	case cls == clsEOF:
		return ptok{start: p, end: p}, nil
	case cls == clsPartial:
		return ptok{}, s.fail(errPartial, p)
	case cls == clsInvalid:
		return ptok{}, s.fail(errInvalid, p)
	case r < 0x80 && isSpace(byte(r)):
		end := p
		for end < len(s.buf) && isSpace(s.buf[end]) {
			end++
		}
		return ptok{kind: 's', start: p, end: end}, nil
	case isNameChar(r) || r == ':':
		end := p + w
		for {
			c, cw, ccls := s.at(end)
			if ccls == clsOK && (isNameChar(c) || c == ':') {
				end += cw
				continue
			}
			if ccls == clsEOF || ccls == clsOK && c < 0x80 && (isSpace(byte(c)) || isTokenEnd(byte(c))) {
				return ptok{kind: 'n', start: p, end: end, text: string(s.buf[p:end])}, nil
			}
			if ccls == clsPartial {
				return ptok{}, s.fail(errPartial, end)
			}
			return ptok{}, s.fail(errInvalid, end)
		}
	case r == '"' || r == '\'':
		end, err := s.literal(p)
		if err != nil {
			return ptok{}, err
		}
		return ptok{kind: 'l', start: p, end: end, text: string(s.buf[p+1 : end-1])}, nil
	case r == '[' || r == ']' || r == '>' || r == '%':
		return ptok{kind: byte(r), start: p, end: p + 1}, nil
	case r == '<':
		return ptok{kind: '<', start: p, end: p + 1}, nil
	}
	return ptok{kind: 'x', start: p, end: p + w}, nil
}

// nextTok skips white space and returns the next token.
func (s *scanner) nextTok(p int) (ptok, error) {
	t, err := s.ptoken(p)
	if err == nil && t.kind == 's' {
		return s.ptoken(t.end)
	}
	return t, err
}

// doctype reads the rest of "<!DOCTYPE" from p and returns the position after its ">".
func (s *scanner) doctype(p int) (int, error) {
	expect := func(t ptok, kinds string) error {
		if t.kind == 0 {
			return s.eof(errNoElements)
		}
		for i := 0; i < len(kinds); i++ {
			if t.kind == kinds[i] {
				return nil
			}
		}
		return s.fail(errSyntax, t.start)
	}
	t, err := s.nextTok(p)
	if err != nil {
		return 0, err
	}
	if err = expect(t, "n"); err != nil {
		return 0, err
	}
	if t, err = s.nextTok(t.end); err != nil {
		return 0, err
	}
	if t.kind == 'n' && (t.text == "SYSTEM" || t.text == "PUBLIC") {
		literals := 1
		if t.text == "PUBLIC" {
			literals = 2
		}
		for i := 0; i < literals; i++ {
			if t, err = s.nextTok(t.end); err != nil {
				return 0, err
			}
			if err = expect(t, "l"); err != nil {
				return 0, err
			}
		}
		if t, err = s.nextTok(t.end); err != nil {
			return 0, err
		}
	}
	if err = expect(t, "[>"); err != nil {
		return 0, err
	}
	if t.kind == '[' {
		end, err := s.subset(t.end)
		if err != nil {
			return 0, err
		}
		if t, err = s.nextTok(end); err != nil {
			return 0, err
		}
		if err = expect(t, ">"); err != nil {
			return 0, err
		}
	}
	return t.end, nil
}

// subset reads the internal subset up to and including its "]".
func (s *scanner) subset(p int) (int, error) {
	for {
		t, err := s.nextTok(p)
		if err != nil {
			return 0, err
		}
		switch t.kind {
		case 0:
			return 0, s.eof(errNoElements)
		case ']':
			return t.end, nil
		case '%':
			name, err := s.ptoken(t.end)
			if err != nil {
				return 0, err
			}
			if name.kind != 'n' || name.end >= len(s.buf) || s.buf[name.end] != ';' {
				return 0, s.fail(errSyntax, t.start)
			}
			p = name.end + 1
		case '<':
			if p, err = s.subsetMarkup(t.start); err != nil {
				return 0, err
			}
		default:
			return 0, s.fail(errSyntax, t.start)
		}
	}
}

// subsetMarkup reads a comment, PI or markup declaration inside the internal subset.
func (s *scanner) subsetMarkup(lt int) (int, error) {
	r, _, cls := s.at(lt + 1)
	if cls == clsOK && r == '?' {
		end, _, _, err := s.pi(lt)
		return end, err
	}
	if cls != clsOK || r != '!' {
		return 0, s.fail(errInvalid, lt+1)
	}
	if next, _, _ := s.at(lt + 2); next == '-' {
		end, _, err := s.comment(lt)
		return end, err
	}
	keyword, err := s.ptoken(lt + 2)
	if err != nil {
		return 0, err
	}
	if keyword.kind != 'n' {
		return 0, s.fail(errInvalid, lt+2)
	}
	switch keyword.text {
	case "ENTITY":
		return s.entity(keyword.end)
	case "ELEMENT", "ATTLIST", "NOTATION":
		return s.skipDecl(keyword.end)
	}
	return 0, s.fail(errSyntax, lt)
}

func (s *scanner) skipDecl(p int) (int, error) {
	for {
		t, err := s.ptoken(p)
		if err != nil {
			return 0, err
		}
		switch t.kind {
		case 0:
			return 0, s.eof(errNoElements)
		case '>':
			return t.end, nil
		}
		p = t.end
	}
}

// entity parses an ENTITY declaration and raises defusedxml's EntitiesForbidden.
func (s *scanner) entity(p int) (int, error) {
	var words []ptok
	for {
		t, err := s.nextTok(p)
		if err != nil {
			return 0, err
		}
		if t.kind == 0 {
			return 0, s.eof(errNoElements)
		}
		if t.kind == '>' {
			break
		}
		words = append(words, t)
		p = t.end
	}
	if len(words) > 0 && words[0].kind == '%' {
		words = words[1:]
	}
	if len(words) < 2 || words[0].kind != 'n' {
		return 0, s.fail(errSyntax, p)
	}
	system, public := "", ""
	hasSystem, hasPublic := false, false
	switch {
	case words[1].kind == 'n' && words[1].text == "SYSTEM" && len(words) >= 3:
		system, hasSystem = words[2].text, true
	case words[1].kind == 'n' && words[1].text == "PUBLIC" && len(words) >= 4:
		public, hasPublic = words[2].text, true
		system, hasSystem = words[3].text, true
	}
	return 0, &Error{Text: "EntitiesForbidden(name='" + words[0].text + "', system_id=" +
		pyrules.ReprOrNone(system, hasSystem) + ", public_id=" + pyrules.ReprOrNone(public, hasPublic) + ")"}
}
