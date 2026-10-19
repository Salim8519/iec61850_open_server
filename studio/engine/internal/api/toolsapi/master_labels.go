package toolsapi

import (
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// clone is dict(o): a copy in the same key order (an empty object for nil or non-objects).
func clone(o any) *pyjson.Object {
	result := pyjson.NewObject()
	source, ok := o.(*pyjson.Object)
	if !ok {
		return result
	}
	for _, key := range source.Keys() {
		item, _ := source.Get(key)
		result.Set(key, item)
	}
	return result
}

// protocolLabels returns the master config, its protocol, all labels and that protocol's labels.
func (a *Area) protocolLabels() (*pyjson.Object, string, *pyjson.Object, *pyjson.Object) {
	current := a.deps.Master.Config()
	protocol := pytext.Str(value(current, "protocol", nil))
	stored := clone(value(current, "labels", nil))
	return current, protocol, stored, clone(value(stored, protocol, nil))
}

// saveLabel names one discovered point (an empty name removes it).
func (a *Area) saveLabel(r *http.Request) (any, error) {
	data, err := body(r)
	if err != nil {
		return nil, err
	}
	key, keyText := value(data, "key", nil).(string)
	label, labelText := value(data, "label", "").(string)
	if !keyText || pytext.Len(key) > 40 || !labelText {
		return nil, apperr.ValueError("Send the point key and its name")
	}
	current, protocol, stored, labels := a.protocolLabels()
	if name := pytext.Strip(label); name != "" {
		labels.Set(key, pytext.Head(name, 80))
	} else {
		labels.Delete(key)
	}
	if err := a.deps.Master.Save(copyWith(current, "labels", stored.Set(protocol, labels))); err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("key", key).Set("label", value(labels, key, "")), nil
}

// importLabels merges names from a CSV with group, address and label columns.
func (a *Area) importLabels(r *http.Request) (any, error) {
	data, err := body(r)
	if err != nil {
		return nil, err
	}
	imported, problems, err := gateway.ImportLabels(pytext.Str(value(data, "text", "")))
	if err != nil {
		return nil, err
	}
	current, protocol, stored, labels := a.protocolLabels()
	for _, key := range imported.Keys() {
		labels.Set(key, value(imported, key, nil))
	}
	if err := a.deps.Master.Save(copyWith(current, "labels", stored.Set(protocol, labels))); err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("imported", int64(imported.Len())).Set("problems", toStrings(problems)), nil
}
