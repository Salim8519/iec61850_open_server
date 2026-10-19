package scl

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/xmldoc"
)

// gooseSubscriptions is goose_links.goose_subscriptions for one subscriber IED.
func gooseSubscriptions(ied *xmldoc.Element, cat *catalogue) ([]any, []any) {
	lgos := lgosTargets(ied)
	var order []string
	entries := map[string]*pyjson.Object{}
	var diagnostics []string
	for _, ld := range ied.FindAll("AccessPoint/Server/LDevice") {
		ldDomain := ld.GetOr("ldName", "")
		if ldDomain == "" {
			ldDomain = ied.GetOr("name", "") + ld.GetOr("inst", "")
		}
		for _, ln := range ld.Children {
			for _, extref := range ln.FindAll("Inputs/ExtRef") {
				service, hasService := extref.Get("serviceType")
				iedName := extref.GetOr("iedName", "")
				if hasService && service != "GOOSE" || iedName == "" {
					continue
				}
				source := iedName + "/" + extref.GetOr("ldInst", "")
				var chosen *publisher
				if cbName := extref.GetOr("srcCBName", ""); cbName != "" {
					ldInst := extref.GetOr("srcLDInst", "")
					if ldInst == "" {
						ldInst = extref.GetOr("ldInst", "")
					}
					key := [3]string{iedName, ldInst, cbName}
					i, ok := cat.index[key]
					if !ok {
						diagnostics = append(diagnostics, "Input "+inputName(extref)+" expects GOOSE block "+
							key[0]+"/"+key[1]+"/"+key[2]+", which is not a publishable GOOSE block in this SCD")
						continue
					}
					chosen = cat.entries[i]
				} else {
					var candidates []*publisher
					for _, entry := range cat.entries {
						if entry.key[0] != iedName {
							continue
						}
						for _, fcda := range entry.fcdas {
							if matches(extref, fcda) {
								candidates = append(candidates, entry)
								break
							}
						}
					}
					if len(candidates) != 1 {
						if hasService && service == "GOOSE" {
							reason := "no"
							if len(candidates) > 0 {
								reason = "more than one"
							}
							diagnostics = append(diagnostics, "Input from "+source+" matches "+reason+
								" GOOSE block; add srcCBName to the ExtRef")
						}
						continue
					}
					chosen = candidates[0]
				}
				ref := get(chosen.control, "gocb_ref").(string)
				entry := entries[ref]
				if entry == nil {
					entry = pyjson.NewObject()
					for _, key := range chosen.control.Keys() {
						entry.Set(key, get(chosen.control, key))
					}
					var target any
					if value, ok := lgos[ref]; ok {
						target = value
					}
					entry.Set("lgos", target).Set("member_count", int64(len(chosen.fcdas))).Set("inputs", []any{})
					entries[ref] = entry
					order = append(order, ref)
				}
				position, path := member(extref, chosen.fcdas)
				match := "data"
				if extref.GetOr("srcCBName", "") != "" {
					match = "named"
				}
				text := source + "/" + extref.GetOr("prefix", "") + extref.GetOr("lnClass", "") +
					extref.GetOr("lnInst", "") + "." + extref.GetOr("doName", "")
				if da := extref.GetOr("daName", ""); da != "" {
					text += "." + da
				}
				input := pyjson.NewObject().Set("int_addr", extref.GetOr("intAddr", "")).
					Set("desc", extref.GetOr("desc", "")).Set("match", match).Set("ld", ldDomain).
					Set("member", position).Set("member_path", path).Set("source", text)
				entry.Set("inputs", append(get(entry, "inputs").([]any), input))
			}
		}
	}
	subscriptions := []any{}
	for _, ref := range order {
		subscriptions = append(subscriptions, entries[ref])
	}
	unique := []any{}
	seen := map[string]bool{}
	for _, text := range diagnostics {
		if !seen[text] {
			seen[text] = true
			unique = append(unique, text)
		}
	}
	return subscriptions, unique
}

// inputName is `values.get('intAddr') or values.get('doName')` formatted by an f-string.
func inputName(extref *xmldoc.Element) string {
	if value := extref.GetOr("intAddr", ""); value != "" {
		return value
	}
	value, present := extref.Get("doName")
	return noneOr(value, present)
}
