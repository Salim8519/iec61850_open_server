package scl

import (
	"hash/fnv"
	"strconv"
	"sync"

	"studio-engine/internal/apperr"
	"studio-engine/internal/scl/xmldoc"
)

// ExtendedControls lists the typed controls (INC, ENC, APC, BAC, BSC, ISC) of one IED, read
// from the stored SCL with the model builder's own rules (ADR-0104). Like DataClasses it never
// changes the IED model: "controls" and every Python-shaped answer stay byte-identical, and
// this Go-only list is sent to the worker under the optional key "extended_controls".
func ExtendedControls(source []byte, iedName string) ([]any, error) {
	doc, err := ParseDocument(source, false)
	if err != nil {
		return nil, err
	}
	templates := doc.Root.Find("DataTypeTemplates")
	if templates == nil {
		return nil, apperr.New("Missing DataTypeTemplates")
	}
	types := map[typeKey]*xmldoc.Element{}
	for _, declaration := range templates.Children {
		id, present := declaration.Get("id")
		if key := (typeKey{declaration.Tag, id, present}); types[key] == nil {
			types[key] = declaration
		}
	}
	for _, ied := range doc.Root.FindAll("IED") {
		if ied.GetOr("name", "") != iedName {
			continue
		}
		servers := ied.FindAll("AccessPoint/Server")
		if len(servers) != 1 {
			return []any{}, nil
		}
		b := newBuilder(types)
		if err := walkObjects(b, servers[0], iedName); err != nil {
			return nil, err
		}
		if b.extended == nil {
			return []any{}, nil
		}
		return b.extended, nil
	}
	return []any{}, nil
}

// walkObjects runs the builder over every data object of a server, as importRun.ied does.
func walkObjects(b *builder, server *xmldoc.Element, iedName string) error {
	for _, ld := range server.FindAll("LDevice") {
		domain := ld.GetOr("ldName", "")
		if domain == "" {
			domain = iedName + ld.GetOr("inst", "")
		}
		for _, ln := range ld.Children {
			if ln.Tag != "LN" && ln.Tag != "LN0" {
				continue
			}
			typeID, hasType := ln.Get("lnType")
			definition, err := b.declaration("LNodeType", typeID, hasType)
			if err != nil {
				return err
			}
			for _, object := range definition.FindAll("DO") {
				objectName, hasName := object.Get("name")
				var instance *xmldoc.Element
				for _, doi := range ln.FindAll("DOI") {
					if value, present := doi.Get("name"); present == hasName && value == objectName {
						instance = doi
						break
					}
				}
				if err := b.dataObject(object, domain+"/"+lnName(ln), instance, 0); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

var extendedCache sync.Map // "<ied id>|<name>|<source hash>" -> []any

// CachedExtendedControls is ExtendedControls remembered per IED id, name and source text.
// The list is shared: callers must not change it.
func CachedExtendedControls(iedID, iedName, source string) ([]any, error) {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(source))
	key := iedID + "|" + iedName + "|" + strconv.FormatUint(hash.Sum64(), 16)
	if found, ok := extendedCache.Load(key); ok {
		return found.([]any), nil
	}
	controls, err := ExtendedControls([]byte(source), iedName)
	if err != nil {
		return nil, err
	}
	extendedCache.Store(key, controls)
	return controls, nil
}
