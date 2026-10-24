package scl

import (
	"sort"

	"studio-engine/internal/scl/xmldoc"
)

// deadbandWrites are the db and zeroDb attributes of one IED's measurements that a client may
// change online, as model references (domain/LN.DO[.SDO].db), sorted (ADR-0118). valKind Set,
// or no valKind (IEC 61850-6 default Set), is writable; RO, Conf and Spec stay read-only. An
// instance (DOI/SDI/DAI) valKind overrides the type's.
func deadbandWrites(server *xmldoc.Element, types map[typeKey]*xmldoc.Element, iedName string) []string {
	var refs []string
	for _, ld := range server.FindAll("LDevice") {
		domain := ld.GetOr("ldName", "")
		if domain == "" {
			domain = iedName + ld.GetOr("inst", "")
		}
		for _, ln := range ld.Children {
			if ln.Tag != "LN" && ln.Tag != "LN0" {
				continue
			}
			typeID, present := ln.Get("lnType")
			definition := types[typeKey{"LNodeType", typeID, present}]
			if definition == nil {
				continue
			}
			for _, object := range definition.FindAll("DO") {
				name := object.GetOr("name", "")
				typeID, present := object.Get("type")
				instance := named(ln, "DOI", name)
				refs = deadbandsOf(refs, types, typeKey{"DOType", typeID, present}, instance, domain+"/"+lnName(ln)+"."+name, 0)
			}
		}
	}
	sort.Strings(refs)
	return refs
}

func deadbandsOf(refs []string, types map[typeKey]*xmldoc.Element, key typeKey, instance *xmldoc.Element, ref string, depth int) []string {
	definition := types[key]
	if definition == nil || depth > 32 {
		return refs
	}
	for _, child := range definition.Children {
		name := child.GetOr("name", "")
		switch {
		case child.Tag == "SDO":
			typeID, present := child.Get("type")
			refs = deadbandsOf(refs, types, typeKey{"DOType", typeID, present}, named(instance, "SDI", name), ref+"."+name, depth+1)
		case child.Tag == "DA" && (name == "db" || name == "zeroDb") && child.GetOr("fc", "") == "CF":
			kind := child.GetOr("valKind", "")
			if value := named(instance, "DAI", name); value != nil {
				if override, has := value.Get("valKind"); has {
					kind = override
				}
			}
			if kind == "" || kind == "Set" {
				refs = append(refs, ref+"."+name)
			}
		}
	}
	return refs
}

// named is the instance child with this tag and name, or nil.
func named(parent *xmldoc.Element, tag, name string) *xmldoc.Element {
	if parent == nil {
		return nil
	}
	for _, child := range parent.Children {
		if child.Tag == tag && child.GetOr("name", "") == name {
			return child
		}
	}
	return nil
}
