package xmldoc

type rawAttr struct {
	qname        string
	nameAt       int
	vStart, vEnd int
}

// startTag tokenises "<name attr='v' ...>" at lt like expat (every syntax error before
// any attribute processing) and returns its end, the raw name, attributes and whether
// it is an empty-element tag.
func (s *scanner) startTag(lt int) (end int, qname string, attrs []rawAttr, empty bool, err error) {
	nameEnd, ok, partial := s.name(lt + 1)
	if partial {
		return 0, "", nil, false, s.fail(errPartial, lt)
	}
	if !ok {
		return 0, "", nil, false, s.fail(errInvalid, nameEnd)
	}
	qname = string(s.buf[lt+1 : nameEnd])
	p := nameEnd
	spaced := false
	for {
		r, w, cls := s.at(p)
		switch {
		case cls == clsEOF:
			return 0, "", nil, false, s.fail(errUnclosed, lt)
		case cls == clsPartial:
			return 0, "", nil, false, s.fail(errPartial, lt)
		case cls == clsInvalid:
			return 0, "", nil, false, s.fail(errInvalid, p)
		case r == '>':
			return p + 1, qname, attrs, false, nil
		case r == '/':
			next, _, nextCls := s.at(p + 1)
			if nextCls == clsEOF {
				return 0, "", nil, false, s.fail(errUnclosed, lt)
			}
			if nextCls == clsPartial {
				return 0, "", nil, false, s.fail(errPartial, lt)
			}
			if next != '>' || nextCls != clsOK {
				return 0, "", nil, false, s.fail(errInvalid, p+1)
			}
			return p + 2, qname, attrs, true, nil
		case r < 0x80 && isSpace(byte(r)):
			spaced = true
			p += w
		case spaced && isNameStart(r):
			attr, next, err := s.attribute(lt, p)
			if err != nil {
				return 0, "", nil, false, err
			}
			attrs = append(attrs, attr)
			p, spaced = next, false
		default:
			return 0, "", nil, false, s.fail(errInvalid, p)
		}
	}
}

// attribute tokenises one attribute starting at p (a name start).
func (s *scanner) attribute(lt, p int) (rawAttr, int, error) {
	nameEnd, ok, partial := s.name(p)
	if partial {
		return rawAttr{}, 0, s.fail(errPartial, lt)
	}
	if !ok {
		return rawAttr{}, 0, s.fail(errInvalid, nameEnd)
	}
	attr := rawAttr{qname: string(s.buf[p:nameEnd]), nameAt: p}
	q := s.skipSpace(nameEnd)
	if err := s.expectByte(lt, q, '='); err != nil {
		return rawAttr{}, 0, err
	}
	q = s.skipSpace(q + 1)
	r, _, cls := s.at(q)
	if cls == clsEOF {
		return rawAttr{}, 0, s.fail(errUnclosed, lt)
	}
	if cls == clsPartial {
		return rawAttr{}, 0, s.fail(errPartial, lt)
	}
	if cls != clsOK || r != '"' && r != '\'' {
		return rawAttr{}, 0, s.fail(errInvalid, q)
	}
	quote := r
	q++
	attr.vStart = q
	for {
		r, w, cls := s.at(q)
		switch {
		case cls == clsEOF:
			return rawAttr{}, 0, s.fail(errUnclosed, lt)
		case cls == clsPartial:
			return rawAttr{}, 0, s.fail(errPartial, lt)
		case cls == clsInvalid || r == '<':
			return rawAttr{}, 0, s.fail(errInvalid, q)
		case r == quote:
			attr.vEnd = q
			after, _, afterCls := s.at(q + 1)
			switch {
			case afterCls == clsEOF:
				return rawAttr{}, 0, s.fail(errUnclosed, lt)
			case afterCls == clsPartial:
				return rawAttr{}, 0, s.fail(errPartial, lt)
			case afterCls == clsOK && (after == '>' || after == '/' || after < 0x80 && isSpace(byte(after))):
				return attr, q + 1, nil
			}
			return rawAttr{}, 0, s.fail(errInvalid, q+1)
		case r == '&':
			end, bad, eof, part := s.reference(q)
			switch {
			case eof:
				return rawAttr{}, 0, s.fail(errUnclosed, lt)
			case part:
				return rawAttr{}, 0, s.fail(errPartial, lt)
			case bad >= 0:
				return rawAttr{}, 0, s.fail(errInvalid, bad)
			}
			q = end
			continue
		}
		q += w
	}
}

func (s *scanner) skipSpace(p int) int {
	for p < len(s.buf) && isSpace(s.buf[p]) {
		p++
	}
	return p
}

func (s *scanner) expectByte(lt, p int, want rune) error {
	r, _, cls := s.at(p)
	switch {
	case cls == clsEOF:
		return s.fail(errUnclosed, lt)
	case cls == clsPartial:
		return s.fail(errPartial, lt)
	case cls != clsOK || r != want:
		return s.fail(errInvalid, p)
	}
	return nil
}

// attrValue normalises an attribute value: references resolved, CR LF / CR / LF / TAB
// become one space each (references to them are kept).
func (s *scanner) attrValue(lt int, a rawAttr) (string, error) {
	out := make([]byte, 0, a.vEnd-a.vStart)
	for p := a.vStart; p < a.vEnd; p++ {
		switch c := s.buf[p]; c {
		case '&':
			end, _, _, _ := s.reference(p)
			text, problem := s.resolveRef(p, end)
			switch problem {
			case errUndefined:
				return "", s.fail(errUndefined, lt)
			case errBadCharRef:
				return "", s.fail(errBadCharRef, p)
			}
			out = append(out, text...)
			p = end - 1
		case '\r':
			out = append(out, ' ')
			if p+1 < a.vEnd && s.buf[p+1] == '\n' {
				p++
			}
		case '\n', '\t':
			out = append(out, ' ')
		default:
			out = append(out, c)
		}
	}
	return string(out), nil
}
