package scl

import (
	"strconv"

	"studio-engine/internal/scl/xmldoc"
)

// DataClasses returns the data class (the DOType's cdc: SPS, DPS, BCR, SPC …) of every data
// object of one IED, keyed by object reference exactly as the model builder writes it
// ("LD/LN.DO", "LD/LN.DO.SDO", "LD/LN.DO(0)"). ADR-0095: the frontend's point kinds use it to
// tell counters and single/double commands apart. It only reads the stored SCL; the IED model
// and every Flask-shaped answer stay unchanged. Objects without a known type are left out.
func DataClasses(source []byte, iedName string) (map[string]string, error) {
	doc, err := ParseDocument(source, false)
	if err != nil {
		return nil, err
	}
	types := map[typeKey]*xmldoc.Element{}
	if templates := doc.Root.Find("DataTypeTemplates"); templates != nil {
		for _, declaration := range templates.Children {
			id, present := declaration.Get("id")
			if key := (typeKey{declaration.Tag, id, present}); types[key] == nil {
				types[key] = declaration
			}
		}
	}
	classes := map[string]string{}
	for _, ied := range doc.Root.FindAll("IED") {
		if ied.GetOr("name", "") != iedName {
			continue
		}
		for _, server := range ied.FindAll("AccessPoint/Server") {
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
						objectClasses(types, object, domain+"/"+lnName(ln), classes, 0)
					}
				}
			}
		}
	}
	return classes, nil
}

// objectClasses records one DO or SDO (every array element) and its sub-objects.
func objectClasses(types map[typeKey]*xmldoc.Element, e *xmldoc.Element, parent string,
	classes map[string]string, depth int) {
	name, named := e.Get("name")
	typeID, typed := e.Get("type")
	definition := types[typeKey{"DOType", typeID, typed}]
	if !named || definition == nil || depth > 32 {
		return
	}
	refs := []string{parent + "." + name}
	if count, err := pyCount(e); err == nil && count > 0 && count <= 256 {
		refs = refs[:0]
		for index := int64(0); index < count; index++ {
			refs = append(refs, parent+"."+name+"("+strconv.FormatInt(index, 10)+")")
		}
	}
	class := definition.GetOr("cdc", "")
	for _, ref := range refs {
		if class != "" {
			classes[ref] = class
		}
		for _, child := range definition.Children {
			if child.Tag == "SDO" {
				objectClasses(types, child, ref, classes, depth+1)
			}
		}
	}
}

var classCache sourceCache

// CachedDataClasses is DataClasses remembered per IED id and source text, so a list that asks
// again (or a filter on every page) neither parses nor hashes the SCL again (ADR-0114).
func CachedDataClasses(iedID, iedName, source string) (map[string]string, error) {
	value, err := classCache.get(iedID, iedName, source, func() (any, error) {
		return DataClasses([]byte(source), iedName)
	})
	if err != nil {
		return nil, err
	}
	return value.(map[string]string), nil
}
