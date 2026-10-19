package sld

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// switchClass is sld_bindings.SWITCH_CLASS.
var switchClass = map[string]string{"CB": "XCBR", "DS": "XSWI", "ES": "XSWI"}

func text(object *pyjson.Object, key string) string {
	if object == nil {
		return ""
	}
	value, _ := object.Get(key)
	result, _ := value.(string)
	return result
}

func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case int64:
		return v != 0
	case float64:
		return v != 0
	case []any:
		return len(v) > 0
	case *pyjson.Object:
		return v.Len() > 0
	}
	return true
}

// reference is _reference: "<domain>/<prefix><lnClass><lnInst>.Pos", "" without a domain.
func reference(lnode scl.LNode, domains map[[2]string]string) string {
	domain, ok := domains[[2]string{lnode.IEDName, lnode.LDInst}]
	if !ok {
		return ""
	}
	return domain + "/" + lnode.Prefix + lnode.LNClass + lnode.LNInst + ".Pos"
}

func firstByRef(list any, ref string) *pyjson.Object {
	items, _ := list.([]any)
	for _, item := range items {
		if entry, _ := item.(*pyjson.Object); text(entry, "ref") == ref {
			return entry
		}
	}
	return nil
}

// state is _state: a direct mapping to a writable Dbpos position, or nil.
func state(ied *pyjson.Object, ref string) *pyjson.Object {
	wanted := ref + ".stVal"
	signals, _ := ied.Get("signals")
	signal := firstByRef(signals, wanted)
	if signal == nil || text(signal, "kind") != "Dbpos" || !truthy(get(signal, "writable")) || text(signal, "fc") == "CF" {
		return nil
	}
	return pyjson.NewObject().Set("mode", "direct").
		Set("source", pyjson.NewObject().Set("ied_id", get(ied, "id")).Set("ref", wanted).Set("report_ref", nil))
}

// control is _control: a DPC control binding, or nil.
func control(ied *pyjson.Object, ref string) *pyjson.Object {
	controls, _ := ied.Get("controls")
	found := firstByRef(controls, ref)
	if found == nil || get(found, "cdc") != "DPC" {
		return nil
	}
	return pyjson.NewObject().Set("ied_id", get(ied, "id")).Set("ref", ref)
}

func get(object *pyjson.Object, key string) any {
	value, _ := object.Get(key)
	return value
}

// Bind is sld_bindings.bind: (state_mapping, control_binding, reasons) for one piece of
// equipment, from the LNodes written on it only; nil mappings are None.
func Bind(kind string, lnodes []scl.LNode, ieds []*pyjson.Object, domains map[[2]string]string) (*pyjson.Object, *pyjson.Object, []string) {
	wanted, isSwitch := switchClass[kind]
	if !isSwitch {
		return nil, nil, []string{}
	}
	var sw, command *scl.LNode
	for index := range lnodes {
		item := &lnodes[index]
		if item.IEDName == "" || item.IEDName == "None" {
			continue
		}
		if sw == nil && item.LNClass == wanted {
			sw = item
		}
		if command == nil && item.LNClass == "CSWI" {
			command = item
		}
	}
	if sw == nil && command == nil {
		return nil, nil, []string{"No IED is linked to this equipment in the SCD"}
	}
	reasons := []string{}
	byName := map[string]*pyjson.Object{}
	for _, ied := range ieds {
		byName[text(ied, "name")] = ied
	}
	for _, lnode := range []*scl.LNode{sw, command} {
		if lnode != nil && byName[lnode.IEDName] == nil {
			reasons = append(reasons, "IED "+lnode.IEDName+" is in the SCD but not loaded in this project")
		}
	}
	var mapping, binding *pyjson.Object
	if sw != nil && byName[sw.IEDName] != nil {
		ref := reference(*sw, domains)
		if ref != "" {
			mapping = state(byName[sw.IEDName], ref)
		}
		if mapping == nil {
			where := ref
			if where == "" {
				where = sw.LNClass
			}
			reasons = append(reasons, "No position signal found at "+where)
		}
	}
	for _, lnode := range []*scl.LNode{command, sw} {
		if lnode != nil && binding == nil && byName[lnode.IEDName] != nil {
			if ref := reference(*lnode, domains); ref != "" {
				binding = control(byName[lnode.IEDName], ref)
			}
		}
	}
	if binding == nil && len(reasons) == 0 {
		reasons = append(reasons, "No controllable switch command is linked; choose the control in Properties")
	} else if binding == nil {
		reasons = append(reasons, "No control bound; choose the control in Properties")
	}
	return mapping, binding, reasons
}
