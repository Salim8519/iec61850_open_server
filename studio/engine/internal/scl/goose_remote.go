package scl

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/xmldoc"
)

// remoteInputs is goose_remote.remote_inputs: ExtRefs naming an IED outside known that
// are clearly GOOSE (serviceType GOOSE, a srcCBName or a packed vendor intAddr).
func remoteInputs(ied *xmldoc.Element, known map[string]bool) []any {
	found := []any{}
	iterTag(ied, "ExtRef", func(extref *xmldoc.Element) {
		publisher := extref.GetOr("iedName", "")
		service, hasService := extref.Get("serviceType")
		if publisher == "" || known[publisher] || hasService && service != "GOOSE" {
			return
		}
		var hint any
		if packed := packedInputHint(extref.GetOr("intAddr", "")); packed != nil && get(packed, "publisher") == publisher {
			hint = packed
		}
		if service != "GOOSE" && extref.GetOr("srcCBName", "") == "" && hint == nil {
			return
		}
		ldInst := extref.GetOr("srcLDInst", "")
		if ldInst == "" {
			ldInst = extref.GetOr("ldInst", "")
		}
		found = append(found, pyjson.NewObject().Set("publisher", publisher).Set("ld_inst", ldInst).
			Set("cb_name", extref.GetOr("srcCBName", "")).Set("int_addr", extref.GetOr("intAddr", "")).
			Set("desc", extref.GetOr("desc", "")).Set("source", remoteSource(extref, publisher)).Set("hint", hint))
	})
	return found
}

// remoteSource is goose_remote._source.
func remoteSource(extref *xmldoc.Element, publisher string) string {
	path := extref.GetOr("prefix", "") + extref.GetOr("lnClass", "") + extref.GetOr("lnInst", "") + "." +
		extref.GetOr("doName", "")
	if da := extref.GetOr("daName", ""); da != "" {
		path += "." + da
	}
	return publisher + "/" + extref.GetOr("ldInst", "") + "/" + path
}
