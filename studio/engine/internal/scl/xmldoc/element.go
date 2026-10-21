package xmldoc

import "strings"

// element applies expat's storeAtts order to a tokenised start tag: per attribute the
// duplicate check, value normalisation and namespace declaration; then prefixed
// attribute names, expanded-name duplicates and the element name. Namespace errors are
// reported at the tag's "<".
func (s *scanner) element(lt int, qname string, raw []rawAttr) (*Element, error) {
	s.ns.push()
	e := &Element{}
	values := make([]string, len(raw))
	for i, a := range raw {
		for _, previous := range raw[:i] {
			if previous.qname == a.qname {
				return nil, s.fail(errDuplicate, a.nameAt)
			}
		}
		value, err := s.attrValue(lt, a)
		if err != nil {
			return nil, err
		}
		values[i] = value
		if a.qname == "xmlns" || strings.HasPrefix(a.qname, "xmlns:") {
			prefix := strings.TrimPrefix(strings.TrimPrefix(a.qname, "xmlns"), ":")
			if problem := s.ns.bind(prefix, value); problem != "" {
				return nil, s.fail(problem, lt)
			}
			e.NSDecls = append(e.NSDecls, NSDecl{Prefix: prefix, URI: value})
		}
	}
	for i, a := range raw {
		if a.qname == "xmlns" || strings.HasPrefix(a.qname, "xmlns:") {
			continue
		}
		prefix, local := splitName(a.qname)
		attr := Attr{Local: local, Prefix: prefix, Value: values[i]}
		if prefix != "" {
			uri, ok := s.ns.resolve(prefix)
			if !ok {
				return nil, s.fail(errUnbound, lt)
			}
			attr.Space = uri
		}
		e.Attrs = append(e.Attrs, attr)
	}
	for i := range e.Attrs {
		if e.Attrs[i].Space == "" {
			continue
		}
		for j := range e.Attrs[:i] {
			if e.Attrs[j].Space == e.Attrs[i].Space && e.Attrs[j].Local == e.Attrs[i].Local {
				return nil, s.fail(errDuplicate, lt)
			}
		}
	}
	prefix, local := splitName(qname)
	uri, ok := s.ns.resolve(prefix)
	if !ok {
		return nil, s.fail(errUnbound, lt)
	}
	e.Tag, e.Space, e.Prefix = local, uri, prefix
	return e, nil
}

// Get returns the attribute value under an ElementTree key ("name" or "{uri}name").
func (e *Element) Get(key string) (string, bool) {
	for i := range e.Attrs {
		a := &e.Attrs[i]
		if a.Space == "" && a.Local == key {
			return a.Value, true
		}
		if a.Space != "" && len(key) > 0 && key[0] == '{' && a.Key() == key {
			return a.Value, true
		}
	}
	return "", false
}

// GetOr returns the attribute or fallback when absent (ElementTree get(key, default)).
func (e *Element) GetOr(key, fallback string) string {
	if value, ok := e.Get(key); ok {
		return value
	}
	return fallback
}

// FindAll returns the children matching a simple ElementTree path "A/B/C" (local names).
func (e *Element) FindAll(path string) []*Element {
	current := []*Element{e}
	for _, step := range strings.Split(path, "/") {
		var next []*Element
		for _, parent := range current {
			for _, child := range parent.Children {
				if child.Tag == step {
					next = append(next, child)
				}
			}
		}
		current = next
	}
	return current
}

// Find returns the first match of FindAll, or nil.
func (e *Element) Find(path string) *Element {
	if !strings.Contains(path, "/") {
		for _, child := range e.Children {
			if child.Tag == path {
				return child
			}
		}
		return nil
	}
	if all := e.FindAll(path); len(all) > 0 {
		return all[0]
	}
	return nil
}

// FindText is ElementTree findtext(path): the text of the first match ("" when it has
// none) and false when nothing matches.
func (e *Element) FindText(path string) (string, bool) {
	match := e.Find(path)
	if match == nil {
		return "", false
	}
	return match.Text, true
}
