package xmldoc

import "strings"

const (
	xmlNamespace   = "http://www.w3.org/XML/1998/namespace"
	xmlnsNamespace = "http://www.w3.org/2000/xmlns/"
)

// namespaces is the in-scope prefix table with an undo log per open element.
type namespaces struct {
	bound map[string]string
	undo  []binding
	marks []int
}

type binding struct {
	prefix   string
	previous string
	had      bool
}

func (n *namespaces) init() {
	n.bound = map[string]string{"xml": xmlNamespace}
}

func (n *namespaces) push() { n.marks = append(n.marks, len(n.undo)) }

func (n *namespaces) pop() {
	mark := n.marks[len(n.marks)-1]
	n.marks = n.marks[:len(n.marks)-1]
	for i := len(n.undo) - 1; i >= mark; i-- {
		b := n.undo[i]
		if b.had {
			n.bound[b.prefix] = b.previous
		} else {
			delete(n.bound, b.prefix)
		}
	}
	n.undo = n.undo[:mark]
}

// bind declares prefix ("" = default namespace) as expat's addBinding does and returns
// the expat error text when the declaration is not allowed.
func (n *namespaces) bind(prefix, uri string) string {
	if uri == "" && prefix != "" {
		return errUndeclare
	}
	if prefix == "xmlns" {
		return errReservedNS
	}
	isXML := uri == xmlNamespace
	if (prefix == "xml") != isXML {
		if prefix == "xml" {
			return errReservedXML
		}
		return errReservedURI
	}
	if uri == xmlnsNamespace {
		return errReservedURI
	}
	previous, had := n.bound[prefix]
	n.undo = append(n.undo, binding{prefix: prefix, previous: previous, had: had})
	n.bound[prefix] = uri
	return ""
}

// resolve returns the namespace of a prefix ("" prefix = default namespace, which may be
// absent) and whether a non-empty prefix is bound.
func (n *namespaces) resolve(prefix string) (string, bool) {
	uri, ok := n.bound[prefix]
	if prefix == "" {
		return uri, true
	}
	return uri, ok && uri != ""
}

// splitName splits "p:local" into prefix and local name.
func splitName(qname string) (string, string) {
	if i := strings.IndexByte(qname, ':'); i >= 0 {
		return qname[:i], qname[i+1:]
	}
	return "", qname
}
