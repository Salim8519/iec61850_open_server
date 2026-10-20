package selftest

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"studio-engine/internal/scl"
)

const sclNamespace = "http://www.iec.ch/61850/2003/SCL"

// Fixture is self_test.fixture(): the SelfTest template whose GGIO AnIn2 becomes an MV
// array (DOType SelfTestArray with SDO samples count 3, inserted before the first DAType),
// written exactly as ET.tostring(root, encoding="utf-8") writes it: no XML declaration and
// the SCL namespace as the generated prefix "ns0".
func Fixture() ([]byte, error) {
	template, err := scl.MakeTemplate("SelfTest")
	if err != nil {
		return nil, err
	}
	root, err := parseTree(template)
	if err != nil {
		return nil, err
	}
	var types *element
	for _, child := range root.children {
		if child.tag == "DataTypeTemplates" {
			types = child
			break
		}
	}
	if types == nil {
		return nil, errors.New("the SelfTest template has no DataTypeTemplates")
	}
	types.walk(func(item *element) {
		if item.tag == "DO" && item.attr("name") == "AnIn2" {
			item.set("type", "SelfTestArray")
		}
	})
	wrapper := &element{tag: "DOType", attrs: [][2]string{{"id", "SelfTestArray"}, {"cdc", "MV"}}}
	wrapper.children = []*element{{tag: "SDO", attrs: [][2]string{{"name", "samples"}, {"type", "StudioMV"}, {"count", "3"}}}}
	position := -1
	for index, item := range types.children {
		if item.tag == "DAType" {
			position = index
			break
		}
	}
	if position < 0 {
		// Python's next() without a default raised StopIteration here.
		return nil, errors.New("the SelfTest template has no DAType")
	}
	types.children = append(types.children[:position], append([]*element{wrapper}, types.children[position:]...)...)
	var out strings.Builder
	root.write(&out, true)
	return []byte(out.String()), nil
}

// element is the part of an ElementTree element the fixture needs: an SCL-namespace tag,
// attributes in document order, text and tail.
type element struct {
	tag        string
	attrs      [][2]string
	text, tail string
	children   []*element
}

func (e *element) attr(name string) string {
	for _, attr := range e.attrs {
		if attr[0] == name {
			return attr[1]
		}
	}
	return ""
}

// set is Element.set: an existing attribute keeps its position, a new one goes last.
func (e *element) set(name, value string) {
	for i := range e.attrs {
		if e.attrs[i][0] == name {
			e.attrs[i][1] = value
			return
		}
	}
	e.attrs = append(e.attrs, [2]string{name, value})
}

// walk is Element.iter: the element itself, then its descendants in document order.
func (e *element) walk(visit func(*element)) {
	visit(e)
	for _, child := range e.children {
		child.walk(visit)
	}
}

// parseTree is ET.fromstring for the generated template: only SCL-namespace elements and
// plain attributes occur there; anything else is refused rather than guessed.
func parseTree(data []byte) (*element, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var stack []*element
	var root *element
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch typed := token.(type) {
		case xml.StartElement:
			if typed.Name.Space != sclNamespace {
				return nil, fmt.Errorf("unexpected element %s in the SelfTest template", typed.Name.Local)
			}
			item := &element{tag: typed.Name.Local}
			for _, attr := range typed.Attr {
				if attr.Name.Space == "" && attr.Name.Local == "xmlns" && attr.Value == sclNamespace {
					continue
				}
				if attr.Name.Space != "" || attr.Name.Local == "xmlns" {
					return nil, fmt.Errorf("unexpected attribute %s:%s in the SelfTest template", attr.Name.Space, attr.Name.Local)
				}
				item.attrs = append(item.attrs, [2]string{attr.Name.Local, attr.Value})
			}
			if len(stack) == 0 {
				root = item
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, item)
			}
			stack = append(stack, item)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 {
				continue // whitespace around the root element: not part of the tree
			}
			parent := stack[len(stack)-1]
			if count := len(parent.children); count > 0 {
				parent.children[count-1].tail += string(typed)
			} else {
				parent.text += string(typed)
			}
		}
	}
	if root == nil {
		return nil, errors.New("the SelfTest template has no root element")
	}
	return root, nil
}

// write is ElementTree's _serialize_xml with the SCL namespace as "ns0".
func (e *element) write(out *strings.Builder, root bool) {
	out.WriteString("<ns0:" + e.tag)
	if root {
		out.WriteString(` xmlns:ns0="` + escapeAttribute(sclNamespace) + `"`)
	}
	for _, attr := range e.attrs {
		out.WriteString(" " + attr[0] + `="` + escapeAttribute(attr[1]) + `"`)
	}
	if e.text == "" && len(e.children) == 0 {
		out.WriteString(" />")
	} else {
		out.WriteString(">" + escapeText(e.text))
		for _, child := range e.children {
			child.write(out, false)
		}
		out.WriteString("</ns0:" + e.tag + ">")
	}
	out.WriteString(escapeText(e.tail))
}

func escapeText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

func escapeAttribute(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;",
		"\r", "&#13;", "\n", "&#10;", "\t", "&#09;").Replace(s)
}
