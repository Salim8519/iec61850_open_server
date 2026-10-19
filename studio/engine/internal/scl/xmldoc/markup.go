package xmldoc

// piEnd scans the rest of a processing instruction whose target ends at p (lt is its "<")
// and returns the position after "?>". Errors follow expat: a bad character is an invalid
// token at its position, the end of the input is an unclosed token (or partial character)
// at lt.
func (s *scanner) piEnd(lt, p int) (int, error) {
	r, w, cls := s.at(p)
	switch {
	case cls == clsEOF:
		return 0, s.fail(errUnclosed, lt)
	case cls == clsPartial:
		return 0, s.fail(errPartial, lt)
	case cls == clsOK && r == '?':
		next, _, nextCls := s.at(p + 1)
		if nextCls == clsEOF {
			return 0, s.fail(errUnclosed, lt)
		}
		if nextCls == clsOK && next == '>' {
			return p + 2, nil
		}
		return 0, s.fail(errInvalid, p+1)
	case cls == clsOK && (r == ' ' || r == '\t' || r == '\n' || r == '\r'):
		p += w
	default:
		return 0, s.fail(errInvalid, p)
	}
	for {
		r, w, cls = s.at(p)
		switch {
		case cls == clsEOF:
			return 0, s.fail(errUnclosed, lt)
		case cls == clsPartial:
			return 0, s.fail(errPartial, lt)
		case cls == clsInvalid:
			return 0, s.fail(errInvalid, p)
		case r == '?' && p+1 < len(s.buf) && s.buf[p+1] == '>':
			return p + 2, nil
		}
		p += w
	}
}

// pi scans "<?target ...?>" at lt and returns its end, the target and the data. A target
// spelled like "xml" in other letter cases is an invalid token after the name; "xml"
// itself is reported by the caller (misplaced declaration).
func (s *scanner) pi(lt int) (end int, target, data string, err error) {
	r, _, cls := s.at(lt + 2)
	switch {
	case cls == clsEOF:
		return 0, "", "", s.fail(errUnclosed, lt)
	case cls == clsPartial:
		return 0, "", "", s.fail(errPartial, lt)
	case cls != clsOK || !isNameStart(r):
		return 0, "", "", s.fail(errInvalid, lt+2)
	}
	nameEnd, ok, partial := s.name(lt + 2)
	if partial {
		return 0, "", "", s.fail(errPartial, lt)
	}
	if !ok {
		return 0, "", "", s.fail(errInvalid, nameEnd)
	}
	target = string(s.buf[lt+2 : nameEnd])
	if len(target) == 3 && (target[0]|0x20) == 'x' && (target[1]|0x20) == 'm' && (target[2]|0x20) == 'l' &&
		target != "xml" {
		return 0, "", "", s.fail(errInvalid, nameEnd)
	}
	end, err = s.piEnd(lt, nameEnd)
	if err != nil {
		return 0, "", "", err
	}
	if target == "xml" {
		return end, target, "", nil
	}
	dataStart := nameEnd
	for dataStart < end-2 && isSpace(s.buf[dataStart]) {
		dataStart++
	}
	return end, target, string(s.buf[dataStart : end-2]), nil
}

// comment scans "<!--...-->" at lt (the "<!" is checked) and returns its end and text.
func (s *scanner) comment(lt int) (int, string, error) {
	p := lt + 2
	for i := 0; i < 2; i++ {
		r, _, cls := s.at(p + i)
		switch {
		case cls == clsEOF:
			return 0, "", s.fail(errUnclosed, lt)
		case cls == clsPartial:
			return 0, "", s.fail(errPartial, lt)
		case r != '-' || cls != clsOK:
			return 0, "", s.fail(errInvalid, p+i)
		}
	}
	p += 2
	textStart := p
	for {
		r, w, cls := s.at(p)
		switch {
		case cls == clsEOF:
			return 0, "", s.fail(errUnclosed, lt)
		case cls == clsPartial:
			return 0, "", s.fail(errPartial, lt)
		case cls == clsInvalid:
			return 0, "", s.fail(errInvalid, p)
		case r == '-' && p+1 < len(s.buf) && s.buf[p+1] == '-':
			after, _, afterCls := s.at(p + 2)
			if afterCls == clsEOF {
				return 0, "", s.fail(errUnclosed, lt)
			}
			if afterCls != clsOK || after != '>' {
				return 0, "", s.fail(errInvalid, p+2)
			}
			return p + 3, string(s.buf[textStart:p]), nil
		}
		p += w
	}
}

// cdata scans "<![CDATA[...]]>" at lt and returns its end and raw text (CRs normalised).
func (s *scanner) cdata(lt int) (int, string, error) {
	const open = "<![CDATA["
	for i := 3; i < len(open); i++ {
		r, _, cls := s.at(lt + i)
		switch {
		case cls == clsEOF:
			return 0, "", s.fail(errUnclosed, lt)
		case cls == clsPartial:
			return 0, "", s.fail(errPartial, lt)
		case cls != clsOK || r != rune(open[i]):
			return 0, "", s.fail(errInvalid, lt+i)
		}
	}
	p := lt + len(open)
	textStart := p
	for {
		r, w, cls := s.at(p)
		switch {
		case cls == clsEOF:
			// expat reports the start of a pending "]" or "]]" token at the end.
			at := p
			for at > textStart && p-at < 2 && s.buf[at-1] == ']' {
				at--
			}
			return 0, "", s.fail(errUnclosedCD, at)
		case cls == clsPartial:
			return 0, "", s.fail(errPartial, p)
		case cls == clsInvalid:
			return 0, "", s.fail(errInvalid, p)
		case r == ']' && p+2 < len(s.buf) && s.buf[p+1] == ']' && s.buf[p+2] == '>':
			return p + 3, normalizeNewlines(s.buf[textStart:p]), nil
		}
		p += w
	}
}

// normalizeNewlines turns CRLF and lone CR into LF (XML end-of-line handling).
func normalizeNewlines(b []byte) string {
	cr := false
	for _, c := range b {
		if c == '\r' {
			cr = true
			break
		}
	}
	if !cr {
		return string(b)
	}
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == '\r' {
			out = append(out, '\n')
			if i+1 < len(b) && b[i+1] == '\n' {
				i++
			}
			continue
		}
		out = append(out, b[i])
	}
	return string(out)
}
