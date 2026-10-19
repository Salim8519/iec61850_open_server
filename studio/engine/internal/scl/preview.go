package scl

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Preview is document.preview_ieds: one summary per IED without compiling models.
func Preview(source []byte) ([]*pyjson.Object, error) {
	doc, err := ParseDocument(source, false)
	if err != nil {
		return nil, err
	}
	var result []*pyjson.Object
	for _, ied := range doc.Root.FindAll("IED") {
		servers := ied.FindAll("AccessPoint/Server")
		reason := ""
		switch {
		case len(servers) == 0:
			reason = "No server model in this IED"
		case len(servers) > 1:
			reason = "Multiple servers are not supported yet"
		}
		name, _ := ied.Get("name") // parse_document guarantees a valid name
		result = append(result, pyjson.NewObject().Set("name", name).
			Set("description", ied.GetOr("desc", "")).Set("manufacturer", ied.GetOr("manufacturer", "")).
			Set("type", ied.GetOr("type", "")).
			Set("logical_devices", int64(len(ied.FindAll("AccessPoint/Server/LDevice")))).
			Set("unavailable_reason", reason))
	}
	if len(result) == 0 {
		return nil, apperr.New("No IEDs found in SCL")
	}
	return result, nil
}
