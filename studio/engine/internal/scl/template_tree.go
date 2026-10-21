package scl

import "strings"

// etNode is a tiny ElementTree element for generated SCL, serialised exactly like
// xml.etree.ElementTree.tostring (attributes in insertion order, "<tag />" when empty).
type etNode struct {
	tag      string
	attrs    [][2]string
	text     string
	children []*etNode
}

func (n *etNode) add(tag string, attrs ...string) *etNode {
	child := &etNode{tag: tag}
	for i := 0; i+1 < len(attrs); i += 2 {
		child.attrs = append(child.attrs, [2]string{attrs[i], attrs[i+1]})
	}
	n.children = append(n.children, child)
	return child
}

func (n *etNode) write(b *strings.Builder) {
	b.WriteString("<" + n.tag)
	for _, attr := range n.attrs {
		b.WriteString(" " + attr[0] + "=\"" + escapeETAttr(attr[1]) + "\"")
	}
	if n.text == "" && len(n.children) == 0 {
		b.WriteString(" />")
		return
	}
	b.WriteString(">" + escapeETText(n.text))
	for _, child := range n.children {
		child.write(b)
	}
	b.WriteString("</" + n.tag + ">")
}

func escapeETText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

func escapeETAttr(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;",
		"\r", "&#13;", "\n", "&#10;", "\t", "&#09;").Replace(s)
}

// tostring is ET.tostring(root, encoding="utf-8", xml_declaration=True).
func (n *etNode) tostring() []byte {
	var b strings.Builder
	b.WriteString("<?xml version='1.0' encoding='utf-8'?>\n")
	n.write(&b)
	return []byte(b.String())
}
