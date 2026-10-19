package xmldoc

// Options choose what the parse keeps.
type Options struct {
	// Nodes keeps comments, PIs and mixed content (needed by the exporter).
	Nodes bool
}

// Parse reads data like defusedxml.ElementTree.fromstring (expat with namespace
// processing): it returns the tree or an *Error whose text equals str(error) in Python
// ("mismatched tag: line 3, column 2", "EntitiesForbidden(name='e', ...)", ...).
func Parse(data []byte, options Options) (*Document, error) {
	buf, start, wide, odd := detect(data)
	s := &scanner{buf: buf, start: start, keep: options.Nodes, oddTail: odd, doc: &Document{Version: "1.0"}}
	if string(data) == "\xef" || string(data) == "\xef\xbb" {
		return nil, s.fail(errUnclosed, 0) // an incomplete byte order mark
	}
	s.ns.init()
	p, err := s.xmlDecl(start, wide)
	if err != nil {
		return nil, err
	}
	p, err = s.prolog(p)
	if err != nil {
		return nil, err
	}
	p, err = s.content(p)
	if err != nil {
		return nil, err
	}
	if err = s.epilog(p); err != nil {
		return nil, err
	}
	return s.doc, nil
}

// eof is the error at the end of the input in a state that needs more data: an odd
// trailing UTF-16 byte is an unclosed token there, as in expat.
func (s *scanner) eof(message string) error {
	if s.oddTail {
		return s.fail(errUnclosed, len(s.buf))
	}
	return s.fail(message, len(s.buf))
}

// xmlDecl handles an XML declaration at the very start and switches the decoding of the
// following bytes to the declared encoding as expat and pyexpat do.
func (s *scanner) xmlDecl(p int, wide int) (int, error) {
	if !s.hasPrefix(p, "<?xml") {
		return p, nil
	}
	r, _, cls := s.at(p + 5)
	if cls == clsOK && isNameChar(r) || cls == clsOK && r == ':' {
		return p, nil // a PI such as <?xml-stylesheet?>
	}
	end, err := s.piEnd(p, p+5)
	if err != nil {
		return 0, err
	}
	encoding, at, err := s.declFields(p+5, end-2)
	if err != nil {
		return 0, err
	}
	if encoding == "" {
		return end, nil
	}
	choice := chooseCodec(encoding, wide)
	switch {
	case choice.keep:
	case choice.incorrect:
		return 0, s.fail(errIncorrectEnc, at)
	case choice.failure != "":
		return 0, &Error{Text: choice.failure}
	case choice.unknown:
		return 0, s.fail(errUnknownEnc, at)
	case wide != 0:
		return 0, s.fail(errInvalid, end)
	default:
		rest := transcode8(s.buf[end:], choice.table)
		s.buf = append(s.buf[:end:end], rest...)
	}
	return end, nil
}

func (s *scanner) hasPrefix(p int, text string) bool {
	return len(s.buf)-p >= len(text) && string(s.buf[p:p+len(text)]) == text
}

// declFields parses the pseudo-attributes between p (after "<?xml") and end (at "?>")
// exactly as expat's doParseXmlDecl; it returns the encoding name and its position.
func (s *scanner) declFields(p, end int) (string, int, error) {
	bad := func(at int) (string, int, error) { return "", 0, s.fail(errXMLDecl, at) }
	name, value, next, ok := s.pseudo(p, end)
	if !ok || name < 0 {
		return bad(next)
	}
	if s.word(name, "version") {
		s.doc.Version = string(s.buf[value:s.quoteEnd(value)])
		name, value, next, ok = s.pseudo(next, end)
		if !ok {
			return bad(next)
		}
		if name < 0 {
			return "", 0, nil
		}
	} else {
		return bad(name)
	}
	encoding, encodingAt := "", 0
	if s.word(name, "encoding") {
		if c := s.buf[value]; !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return bad(value)
		}
		encodingAt = value
		encoding = string(s.buf[value:s.quoteEnd(value)])
		name, value, next, ok = s.pseudo(next, end)
		if !ok {
			return bad(next)
		}
		if name < 0 {
			return encoding, encodingAt, nil
		}
	}
	if !s.word(name, "standalone") {
		return bad(name)
	}
	text := string(s.buf[value:s.quoteEnd(value)])
	if text != "yes" && text != "no" {
		return bad(value)
	}
	for next < end && isSpace(s.buf[next]) {
		next++
	}
	if next != end {
		return bad(next)
	}
	return encoding, encodingAt, nil
}

func (s *scanner) word(at int, text string) bool {
	if !s.hasPrefix(at, text) {
		return false
	}
	c := s.buf[at+len(text)]
	return c == '=' || isSpace(c)
}

func (s *scanner) quoteEnd(value int) int {
	open := s.buf[value-1]
	end := value
	for s.buf[end] != open {
		end++
	}
	return end
}

// pseudo is expat's parsePseudoAttribute: name < 0 means no more attributes; on failure
// next is the bad position.
func (s *scanner) pseudo(p, end int) (name, value, next int, ok bool) {
	if p == end {
		return -1, 0, p, true
	}
	if !isSpace(s.buf[p]) {
		return 0, 0, p, false
	}
	for p < end && isSpace(s.buf[p]) {
		p++
	}
	if p == end {
		return -1, 0, p, true
	}
	name = p
	for {
		if p >= end || s.buf[p] >= 0x80 {
			return 0, 0, p, false
		}
		c := s.buf[p]
		if c == '=' {
			break
		}
		if isSpace(c) {
			for p < end && isSpace(s.buf[p]) {
				p++
			}
			if p >= end || s.buf[p] != '=' {
				return 0, 0, p, false
			}
			break
		}
		p++
	}
	if p == name {
		return 0, 0, p, false
	}
	p++
	for p < end && isSpace(s.buf[p]) {
		p++
	}
	if p >= end || s.buf[p] != '"' && s.buf[p] != '\'' {
		return 0, 0, p, false
	}
	open := s.buf[p]
	p++
	value = p
	for ; ; p++ {
		if p >= end {
			return 0, 0, p, false
		}
		c := s.buf[p]
		if c == open {
			break
		}
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == '_') {
			return 0, 0, p, false
		}
	}
	return name, value, p + 1, true
}
