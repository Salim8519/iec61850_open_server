package xmldoc

type open struct {
	elem  *Element
	qname string
}

// content reads the root element and everything inside it (p is the root's "<") and
// returns the position after the root's end tag.
func (s *scanner) content(p int) (int, error) {
	var stack []open
	var text []byte
	flush := func() {
		if len(stack) > 0 && len(text) > 0 {
			s.addText(stack[len(stack)-1].elem, string(text))
		}
		text = text[:0]
	}
	for {
		if s.doc.Root != nil && len(stack) == 0 {
			return p, nil
		}
		c := byte(0)
		if p < len(s.buf) {
			c = s.buf[p]
		}
		switch {
		case p >= len(s.buf):
			return 0, s.eof(errNoElements)
		case c == '<':
			flush()
			next, err := s.markup(p, &stack)
			if err != nil {
				return 0, err
			}
			p = next
		case c == '&':
			end, bad, eof, partial := s.reference(p)
			switch {
			case eof:
				return 0, s.fail(errUnclosed, p)
			case partial:
				return 0, s.fail(errPartial, p)
			case bad >= 0:
				return 0, s.fail(errInvalid, bad)
			}
			value, problem := s.resolveRef(p, end)
			if problem != "" {
				return 0, s.fail(problem, p)
			}
			text = append(text, value...)
			p = end
		case c == '\r':
			text = append(text, '\n')
			p++
			if p < len(s.buf) && s.buf[p] == '\n' {
				p++
			}
		case c == ']' && p+2 < len(s.buf) && s.buf[p+1] == ']' && s.buf[p+2] == '>':
			return 0, s.fail(errInvalid, p+2)
		case c >= 0x20 && c < 0x80 || c == '\n' || c == '\t':
			start := p
			for p < len(s.buf) {
				c = s.buf[p]
				if c < 0x20 && c != '\n' && c != '\t' || c >= 0x80 || c == '<' || c == '&' || c == ']' {
					break
				}
				p++
			}
			if p == start {
				text = append(text, c)
				p++
				continue
			}
			text = append(text, s.buf[start:p]...)
		default:
			_, w, cls := s.at(p)
			switch cls {
			case clsPartial:
				return 0, s.fail(errPartial, p)
			case clsInvalid:
				return 0, s.fail(errInvalid, p)
			}
			text = append(text, s.buf[p:p+w]...)
			p += w
		}
	}
}

// markup handles one "<" inside content: start tag, end tag, comment, CDATA or PI.
func (s *scanner) markup(lt int, stack *[]open) (int, error) {
	r, _, cls := s.at(lt + 1)
	switch {
	case cls == clsEOF:
		return 0, s.fail(errUnclosed, lt)
	case cls == clsPartial:
		return 0, s.fail(errPartial, lt)
	case cls == clsOK && r == '/':
		return s.endTag(lt, stack)
	case cls == clsOK && r == '?':
		end, target, data, err := s.pi(lt)
		if err != nil {
			return 0, err
		}
		if target == "xml" {
			return 0, s.fail(errMisplacedPI, lt)
		}
		s.addNode((*stack)[len(*stack)-1].elem, Node{Kind: PINode, Text: target, Data: data})
		return end, nil
	case cls == clsOK && r == '!':
		next, _, nextCls := s.at(lt + 2)
		switch {
		case nextCls == clsEOF:
			return 0, s.fail(errUnclosed, lt)
		case nextCls == clsOK && next == '-':
			end, text, err := s.comment(lt)
			if err == nil {
				s.addNode((*stack)[len(*stack)-1].elem, Node{Kind: CommentNode, Text: text})
			}
			return end, err
		case nextCls == clsOK && next == '[':
			end, text, err := s.cdata(lt)
			if err == nil {
				s.addText((*stack)[len(*stack)-1].elem, text)
			}
			return end, err
		}
		return 0, s.fail(errInvalid, lt+2)
	case cls == clsOK && isNameStart(r):
		end, qname, raw, empty, err := s.startTag(lt)
		if err != nil {
			return 0, err
		}
		elem, err := s.element(lt, qname, raw)
		if err != nil {
			return 0, err
		}
		if len(*stack) == 0 {
			s.doc.Root = elem
		} else {
			parent := (*stack)[len(*stack)-1].elem
			parent.Children = append(parent.Children, elem)
			s.addNode(parent, Node{Kind: ElementNode, Elem: elem})
		}
		if empty {
			s.ns.pop()
			return end, nil
		}
		*stack = append(*stack, open{elem: elem, qname: qname})
		return end, nil
	}
	return 0, s.fail(errInvalid, lt+1)
}

// endTag checks "</name>" against the open element.
func (s *scanner) endTag(lt int, stack *[]open) (int, error) {
	r, _, cls := s.at(lt + 2)
	switch {
	case cls == clsEOF:
		return 0, s.fail(errUnclosed, lt)
	case cls == clsPartial:
		return 0, s.fail(errPartial, lt)
	case cls != clsOK || !isNameStart(r):
		return 0, s.fail(errInvalid, lt+2)
	}
	nameEnd, ok, partial := s.name(lt + 2)
	if partial {
		return 0, s.fail(errPartial, lt)
	}
	if !ok {
		return 0, s.fail(errInvalid, nameEnd)
	}
	p := s.skipSpace(nameEnd)
	if err := s.expectByte(lt, p, '>'); err != nil {
		return 0, err
	}
	top := (*stack)[len(*stack)-1]
	if string(s.buf[lt+2:nameEnd]) != top.qname {
		return 0, s.fail(errMismatch, lt+2)
	}
	*stack = (*stack)[:len(*stack)-1]
	s.ns.pop()
	return p + 1, nil
}

// addText adds character data to e as ElementTree does (text before the first child,
// else the last child's tail) and to the mixed content when kept.
func (s *scanner) addText(e *Element, text string) {
	if len(e.Children) == 0 {
		e.Text += text
		e.HasText = true
	} else {
		e.Children[len(e.Children)-1].Tail += text
	}
	if !s.keep {
		return
	}
	if n := len(e.Nodes); n > 0 && e.Nodes[n-1].Kind == TextNode {
		e.Nodes[n-1].Text += text
		return
	}
	e.Nodes = append(e.Nodes, Node{Kind: TextNode, Text: text})
}

func (s *scanner) addNode(e *Element, node Node) {
	if s.keep {
		e.Nodes = append(e.Nodes, node)
	}
}
