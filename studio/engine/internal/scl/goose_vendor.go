package scl

import (
	"regexp"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/xmldoc"
)

// goose_vendor: vendor GOOSE hints read from SCL without inventing data (change 0094).

var (
	packedMAC  = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$`)
	packedName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)
)

// pyMatch is re.match for a pattern ending in "$": Python's "$" also matches before a
// final newline.
func pyMatch(pattern *regexp.Regexp, text string) bool {
	return pattern.MatchString(text) || strings.HasSuffix(text, "\n") && pattern.MatchString(text[:len(text)-1])
}

// vendorAttributes is vendor_attributes: every namespaced attribute as
// {namespace: {name: value}} in attribute order.
func vendorAttributes(element *xmldoc.Element) *pyjson.Object {
	result := pyjson.NewObject()
	for _, attr := range element.Attrs {
		if attr.Space == "" {
			continue
		}
		group, ok := get(result, attr.Space).(*pyjson.Object)
		if !ok {
			group = pyjson.NewObject()
			result.Set(attr.Space, group)
		}
		group.Set(attr.Local, attr.Value)
	}
	return result
}

// geLabel is ge_label: the GE Multilin Label attribute or "".
func geLabel(element *xmldoc.Element) string {
	return element.GetOr("{"+geMultilin+"}Label", "")
}

// packedInputHint is packed_input_hint: publisher, MAC and type from a packed vendor
// intAddr "LD|IED|index|MAC|type[|...]", or nil.
func packedInputHint(intAddr string) *pyjson.Object {
	parts := strings.Split(intAddr, "|")
	if len(parts) < 5 || !pyMatch(packedName, parts[1]) || !pyMatch(packedMAC, parts[3]) {
		return nil
	}
	return pyjson.NewObject().Set("ld_inst", parts[0]).Set("publisher", parts[1]).Set("index", parts[2]).
		Set("mac", strings.ToUpper(strings.ReplaceAll(parts[3], ":", "-"))).Set("value_type", parts[4]).
		Set("vendor", "packed intAddr")
}

// attrib is dict(element.attrib): ElementTree keys ("name" or "{uri}name") in order.
func attrib(element *xmldoc.Element) *pyjson.Object {
	result := pyjson.NewObject()
	for _, attr := range element.Attrs {
		result.Set(attr.Key(), attr.Value)
	}
	return result
}

// iterTag is element.iter(tag): the element and its descendants in document order.
func iterTag(element *xmldoc.Element, tag string, visit func(*xmldoc.Element)) {
	if element.Tag == tag {
		visit(element)
	}
	for _, child := range element.Children {
		iterTag(child, tag, visit)
	}
}
