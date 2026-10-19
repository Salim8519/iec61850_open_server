// Package xmldoc parses XML in one pass into a small ordered tree, accepting and rejecting
// exactly what Python's defusedxml.ElementTree (expat 2.6.3) accepts and rejects, with the
// same error texts and positions. The tree offers ElementTree's view (local tag names,
// text/tail without comments) for the importer, and keeps comments, processing
// instructions, namespace declarations and prefixes for the lxml-style exporter.
package xmldoc

// Attr is one attribute in document order. Space is the namespace URI ("" when the name
// has no prefix), Prefix the prefix as written.
type Attr struct {
	Space, Local, Prefix, Value string
}

// Key returns the ElementTree attribute key: "local" or "{uri}local".
func (a Attr) Key() string {
	if a.Space == "" {
		return a.Local
	}
	return "{" + a.Space + "}" + a.Local
}

// NSDecl is one xmlns declaration as written on an element (Prefix "" = default namespace).
type NSDecl struct {
	Prefix, URI string
}

// NodeKind says what a content node is.
type NodeKind uint8

const (
	// TextNode is merged character data (entities, character references and CDATA resolved).
	TextNode NodeKind = iota
	// ElementNode is a child element.
	ElementNode
	// CommentNode is a comment; Text holds its content.
	CommentNode
	// PINode is a processing instruction; Text holds the target, Data the rest.
	PINode
)

// Node is one item of mixed content, kept only when the parse asked for it (export).
type Node struct {
	Kind NodeKind
	Text string
	Data string
	Elem *Element
}

// Element is one element. Tag is the local name (ElementTree view after the importer
// strips namespaces); Text and Tail follow ElementTree (comments and PIs dropped).
type Element struct {
	Tag      string
	Space    string
	Prefix   string
	Attrs    []Attr
	NSDecls  []NSDecl
	Children []*Element
	Text     string
	Tail     string
	Nodes    []Node
	HasText  bool
}

// Document is a parsed XML document.
type Document struct {
	Root *Element
	// Before and After are the comments and PIs outside the root element (export only).
	Before, After []Node
	// Doctype is true when the document has a DOCTYPE declaration.
	Doctype bool
}
