package scl

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/scl/xmldoc"
)

// laterContent are the LN children a new DOI must precede (exporter._instance).
var laterContent = map[string]bool{"Inputs": true, "Log": true, "GSEControl": true,
	"SampledValueControl": true, "SettingControl": true, "SCLControl": true}

// path is an element with its ancestors (for namespace prefix lookup).
type path []*xmldoc.Element

func (p path) last() *xmldoc.Element { return p[len(p)-1] }

// sclPrefix finds the prefix lxml uses for a new SCL-namespace element: the nearest
// in-scope declaration of the namespace, searching from the parent upward.
func (p path) sclPrefix() string {
	for i := len(p) - 1; i >= 0; i-- {
		for _, decl := range p[i].NSDecls {
			if decl.URI == sclNamespace {
				return decl.Prefix
			}
		}
	}
	return ""
}

func inSCL(e *xmldoc.Element, tag string) bool { return e.Space == sclNamespace && e.Tag == tag }

// logicalNodes is exporter._logical_nodes: (domain, LN name) → the LN with its ancestors.
func logicalNodes(root, ied *xmldoc.Element, name string) map[[2]string]path {
	nodes := map[[2]string]path{}
	for _, ap := range ied.Children {
		if !inSCL(ap, "AccessPoint") {
			continue
		}
		for _, server := range ap.Children {
			if !inSCL(server, "Server") {
				continue
			}
			for _, device := range server.Children {
				if !inSCL(device, "LDevice") {
					continue
				}
				domain := device.GetOr("ldName", "")
				if domain == "" {
					domain = name + device.GetOr("inst", "")
				}
				for _, node := range device.Children {
					if inSCL(node, "LN") || inSCL(node, "LN0") {
						nodes[[2]string{domain, lnName(node)}] = path{root, ied, ap, server, device, node}
					}
				}
			}
		}
	}
	return nodes
}

// instance is exporter._instance: the unique named, unindexed child, created when absent.
func instance(parent path, tag, name string) (path, error) {
	var matches []*xmldoc.Element
	for _, child := range parent.last().Children {
		if inSCL(child, tag) && child.GetOr("name", "\x00") == name {
			matches = append(matches, child)
		}
	}
	indexed := false
	for _, child := range matches {
		if _, ok := child.Get("ix"); ok {
			indexed = true
		}
	}
	if len(matches) > 1 || indexed {
		return nil, apperr.New("Ambiguous or indexed SCL instance: " + name)
	}
	if len(matches) == 1 {
		return append(append(path{}, parent...), matches[0]), nil
	}
	owner := parent.last()
	child := &xmldoc.Element{Tag: tag, Space: sclNamespace, Prefix: parent.sclPrefix(),
		Attrs: []xmldoc.Attr{{Local: "name", Value: name}}}
	nodeAt, childAt := len(owner.Nodes), len(owner.Children)
	if tag == "DOI" {
		for i, node := range owner.Nodes {
			if node.Kind == xmldoc.ElementNode && node.Elem.Space == sclNamespace && laterContent[node.Elem.Tag] {
				nodeAt = i
				break
			}
		}
		for i, existing := range owner.Children {
			if existing.Space == sclNamespace && laterContent[existing.Tag] {
				childAt = i
				break
			}
		}
	}
	owner.Nodes = insertNode(owner.Nodes, nodeAt, xmldoc.Node{Kind: xmldoc.ElementNode, Elem: child})
	owner.Children = append(owner.Children[:childAt], append([]*xmldoc.Element{child}, owner.Children[childAt:]...)...)
	return append(append(path{}, parent...), child), nil
}

func insertNode(nodes []xmldoc.Node, at int, node xmldoc.Node) []xmldoc.Node {
	nodes = append(nodes, xmldoc.Node{})
	copy(nodes[at+1:], nodes[at:])
	nodes[at] = node
	return nodes
}

// setText is lxml's element.text = value: leading text nodes are replaced.
func setText(e *xmldoc.Element, value string) {
	start := 0
	for start < len(e.Nodes) && e.Nodes[start].Kind == xmldoc.TextNode {
		start++
	}
	rest := e.Nodes[start:]
	e.Nodes = append([]xmldoc.Node{{Kind: xmldoc.TextNode, Text: value}}, rest...)
	if len(e.Children) == 0 {
		e.Text = value
	}
}
