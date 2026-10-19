package xmldoc

import "strings"

// SerializeLXML writes the document exactly as lxml's
// etree.tostring(tree, encoding="UTF-8", xml_declaration=True) does for a document parsed
// with lxml's defaults: declaration with single quotes and the source version, top-level
// comments and PIs without separators, namespace declarations before attributes, empty
// elements as "<x/>", libxml2's text and attribute escapes. The tree must have been
// parsed with Options{Nodes: true}.
func (d *Document) SerializeLXML() []byte {
	var b strings.Builder
	b.WriteString("<?xml version='" + d.Version + "' encoding='UTF-8'?>\n")
	for _, node := range d.Before {
		writeNode(&b, node)
	}
	writeElement(&b, d.Root)
	for _, node := range d.After {
		writeNode(&b, node)
	}
	return []byte(b.String())
}

func qualified(prefix, local string) string {
	if prefix == "" {
		return local
	}
	return prefix + ":" + local
}

func writeElement(b *strings.Builder, e *Element) {
	name := qualified(e.Prefix, e.Tag)
	b.WriteString("<" + name)
	for _, decl := range e.NSDecls {
		if decl.Prefix == "" {
			b.WriteString(" xmlns=\"")
		} else {
			b.WriteString(" xmlns:" + decl.Prefix + "=\"")
		}
		b.WriteString(escapeAttr(decl.URI) + "\"")
	}
	for _, attr := range e.Attrs {
		b.WriteString(" " + qualified(attr.Prefix, attr.Local) + "=\"" + escapeAttr(attr.Value) + "\"")
	}
	if len(e.Nodes) == 0 {
		b.WriteString("/>")
		return
	}
	b.WriteString(">")
	for _, node := range e.Nodes {
		writeNode(b, node)
	}
	b.WriteString("</" + name + ">")
}

func writeNode(b *strings.Builder, node Node) {
	switch node.Kind {
	case TextNode:
		b.WriteString(escapeText(node.Text))
	case ElementNode:
		writeElement(b, node.Elem)
	case CommentNode:
		b.WriteString("<!--" + node.Text + "-->")
	case PINode:
		if node.Data == "" {
			b.WriteString("<?" + node.Text + "?>")
		} else {
			b.WriteString("<?" + node.Text + " " + node.Data + "?>")
		}
	}
}

var (
	textEscaper = strings.NewReplacer("<", "&lt;", ">", "&gt;", "&", "&amp;", "\r", "&#13;")
	attrEscaper = strings.NewReplacer("<", "&lt;", ">", "&gt;", "&", "&amp;", "\"", "&quot;",
		"\n", "&#10;", "\r", "&#13;", "\t", "&#9;")
)

func escapeText(s string) string { return textEscaper.Replace(s) }

func escapeAttr(s string) string { return attrEscaper.Replace(s) }
