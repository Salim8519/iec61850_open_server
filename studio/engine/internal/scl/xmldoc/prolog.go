package xmldoc

// prolog reads what precedes the root element (after an XML declaration) and returns the
// position of the root's "<". Unexpected tokens are a syntax error at their start.
func (s *scanner) prolog(p int) (int, error) {
	doctype := false
	for {
		r, w, cls := s.at(p)
		switch {
		case cls == clsEOF:
			return 0, s.eof(errNoElements)
		case cls == clsPartial:
			return 0, s.fail(errPartial, p)
		case cls == clsInvalid:
			return 0, s.fail(errInvalid, p)
		case isSpace(byte(r)) && r < 0x80:
			p += w
			continue
		case r != '<':
			return 0, s.prologJunk(p, errSyntax)
		}
		next, _, nextCls := s.at(p + 1)
		switch {
		case nextCls == clsEOF:
			return 0, s.fail(errUnclosed, p)
		case nextCls == clsPartial:
			return 0, s.fail(errPartial, p)
		case nextCls == clsOK && next == '?':
			end, target, data, err := s.pi(p)
			if err != nil {
				return 0, err
			}
			if target == "xml" {
				return 0, s.fail(errMisplacedPI, p)
			}
			s.keepOutside(Node{Kind: PINode, Text: target, Data: data}, true)
			p = end
		case nextCls == clsOK && next == '!':
			end, isDoctype, err := s.declaration(p, doctype)
			if err != nil {
				return 0, err
			}
			doctype = doctype || isDoctype
			p = end
		case nextCls == clsOK && isNameStart(next):
			return p, nil
		default:
			return 0, s.fail(errInvalid, p+1)
		}
	}
}

// declaration handles "<!--" comments and the DOCTYPE in the prolog.
func (s *scanner) declaration(lt int, seenDoctype bool) (int, bool, error) {
	r, _, cls := s.at(lt + 2)
	switch {
	case cls == clsEOF:
		return 0, false, s.fail(errUnclosed, lt)
	case cls == clsOK && r == '-':
		end, text, err := s.comment(lt)
		if err == nil {
			s.keepOutside(Node{Kind: CommentNode, Text: text}, true)
		}
		return end, false, err
	case cls == clsOK && isNameStart(r):
		nameEnd, _, _ := s.name(lt + 2)
		after, _, afterCls := s.at(nameEnd)
		if afterCls == clsEOF {
			return 0, false, s.eof(errNoElements)
		}
		if afterCls != clsOK || !(after < 0x80 && isSpace(byte(after))) && after != '%' {
			return 0, false, s.fail(errInvalid, nameEnd)
		}
		if seenDoctype || string(s.buf[lt+2:nameEnd]) != "DOCTYPE" {
			return 0, false, s.fail(errSyntax, lt)
		}
		end, err := s.doctype(nameEnd)
		s.doc.Doctype = true
		return end, true, err
	}
	return 0, false, s.fail(errInvalid, lt+2)
}

// prologJunk classifies text in the prolog or epilog the way expat's prolog tokenizer
// does: a well-formed token gives message at its start, a character that cannot start or
// end such a token is an invalid token at its position.
func (s *scanner) prologJunk(p int, message string) error {
	r, w, _ := s.at(p)
	switch {
	case isNameChar(r) || r == ':':
		end := p + w
		for {
			c, cw, cls := s.at(end)
			if cls == clsEOF {
				return s.fail(message, p)
			}
			if cls == clsPartial {
				return s.fail(errPartial, end)
			}
			if cls == clsOK && (isNameChar(c) || c == ':') {
				end += cw
				continue
			}
			if cls == clsOK && c < 0x80 && (isSpace(byte(c)) || isTokenEnd(byte(c))) {
				return s.fail(message, p)
			}
			return s.fail(errInvalid, end)
		}
	case r == '"' || r == '\'':
		if _, err := s.literal(p); err != nil {
			return err
		}
		return s.fail(message, p)
	case r < 0x80 && isTokenStart(byte(r)):
		return s.fail(message, p)
	}
	return s.fail(errInvalid, p)
}

// literal scans a quoted literal at p (expat scanLit) and returns the position after it:
// it must be followed by white space, '>', '%' or '[', or end the input.
func (s *scanner) literal(p int) (int, error) {
	quote := s.buf[p]
	q := p + 1
	for {
		r, w, cls := s.at(q)
		switch {
		case cls == clsEOF:
			return 0, s.fail(errUnclosed, p)
		case cls == clsPartial:
			return 0, s.fail(errPartial, p)
		case cls == clsInvalid:
			return 0, s.fail(errInvalid, q)
		case r == rune(quote):
			after, _, afterCls := s.at(q + 1)
			if afterCls == clsEOF || afterCls == clsOK && after < 0x80 &&
				(isSpace(byte(after)) || after == '>' || after == '%' || after == '[') {
				return q + 1, nil
			}
			return 0, s.fail(errInvalid, q+1)
		}
		q += w
	}
}

func isTokenEnd(c byte) bool {
	switch c {
	case '>', '%', ')', '|', ',', '+', '*', '?', '[', ']':
		return true
	}
	return false
}

func isTokenStart(c byte) bool {
	switch c {
	case '[', ']', '(', ')', '|', ',', '>', '%', '#':
		return true
	}
	return false
}

// epilog reads what follows the root element: only white space, comments and PIs.
func (s *scanner) epilog(p int) error {
	for {
		r, w, cls := s.at(p)
		switch {
		case cls == clsEOF:
			if s.oddTail {
				return s.fail(errUnclosed, len(s.buf))
			}
			return nil
		case cls == clsPartial:
			return s.fail(errPartial, p)
		case cls == clsInvalid:
			return s.fail(errInvalid, p)
		case r < 0x80 && isSpace(byte(r)):
			p += w
			continue
		case r != '<':
			return s.prologJunk(p, errJunk)
		}
		next, _, nextCls := s.at(p + 1)
		switch {
		case nextCls == clsEOF:
			return s.fail(errUnclosed, p)
		case nextCls == clsOK && next == '?':
			end, target, data, err := s.pi(p)
			if err != nil {
				return err
			}
			if target == "xml" {
				return s.fail(errJunk, p)
			}
			s.keepOutside(Node{Kind: PINode, Text: target, Data: data}, false)
			p = end
		case nextCls == clsOK && next == '!':
			after, _, afterCls := s.at(p + 2)
			if afterCls == clsOK && after == '-' {
				end, text, err := s.comment(p)
				if err != nil {
					return err
				}
				s.keepOutside(Node{Kind: CommentNode, Text: text}, false)
				p = end
				continue
			}
			if afterCls == clsOK && isNameStart(after) {
				return s.fail(errJunk, p)
			}
			if afterCls == clsEOF {
				return s.fail(errUnclosed, p)
			}
			return s.fail(errInvalid, p+2)
		case nextCls == clsOK && isNameStart(next):
			return s.fail(errJunk, p)
		default:
			return s.fail(errInvalid, p+1)
		}
	}
}

func (s *scanner) keepOutside(node Node, before bool) {
	if !s.keep {
		return
	}
	if before {
		s.doc.Before = append(s.doc.Before, node)
	} else {
		s.doc.After = append(s.doc.After, node)
	}
}
