package scl

import (
	"sort"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/xmldoc"
)

// GooseDocument is one stored SCL text with the IED names loaded from it.
type GooseDocument struct {
	Text   string
	Loaded []string
}

type goosePart struct {
	root   *xmldoc.Element
	loaded map[string]bool
}

type gooseOwner struct {
	index int
	ied   *xmldoc.Element
}

// ReadGooseSources is goose_sources.read_goose_sources: the merged GOOSE declarations of
// every document, in the JSON layout acquisition/goosedom.DeclarationsFromJSON reads:
//
//	{ieds: {name: {desc, manufacturer, type, loaded}}, blocks: [[[ied, ldInst, cbName], block], ...],
//	 subscriptions: {name: [...]}, problems: {name: [...]}, remote: {name: [...]}}
//
// (Python's blocks dict has tuple keys; the list keeps its order.) Errors are
// parse_document's StudioErrors.
func ReadGooseSources(documents []GooseDocument) (*pyjson.Object, error) {
	parts := make([]goosePart, 0, len(documents))
	loaded := map[string]bool{}
	for _, document := range documents {
		parsed, err := ParseDocument([]byte(document.Text), false)
		if err != nil {
			return nil, err
		}
		names := map[string]bool{}
		for _, name := range document.Loaded {
			names[name], loaded[name] = true, true
		}
		parts = append(parts, goosePart{root: parsed.Root, loaded: names})
	}
	owners := gooseOwners(parts)
	blocks := &catalogue{index: map[[3]string]int{}}
	for index, part := range parts {
		for _, entry := range blockInventory(part.root).entries {
			if owners[entry.key[0]].index == index {
				blocks.put(entry)
			}
		}
	}
	names := make([]string, 0, len(owners))
	known := map[string]bool{}
	for name := range owners {
		names = append(names, name)
		known[name] = true
	}
	sort.Strings(names)
	ieds, subscriptions, problems, remote := pyjson.NewObject(), pyjson.NewObject(), pyjson.NewObject(), pyjson.NewObject()
	for _, name := range names {
		ied, root := owners[name].ied, parts[owners[name].index].root
		ieds.Set(name, pyjson.NewObject().Set("desc", ied.GetOr("desc", "")).
			Set("manufacturer", ied.GetOr("manufacturer", "")).Set("type", ied.GetOr("type", "")).
			Set("loaded", loaded[name]))
		found, diagnostics := gooseSubscriptions(root, ied, blocks)
		inputs := remoteInputs(root, ied, known)
		subscriptions.Set(name, found)
		problems.Set(name, withoutOutside(diagnostics, inputs))
		remote.Set(name, inputs)
	}
	listed := []any{}
	for _, entry := range blocks.entries {
		listed = append(listed, []any{[]any{entry.key[0], entry.key[1], entry.key[2]}, entry.control})
	}
	return pyjson.NewObject().Set("ieds", ieds).Set("blocks", listed).Set("subscriptions", subscriptions).
		Set("problems", problems).Set("remote", remote), nil
}

// gooseOwners is goose_sources._owners: each IED name is read from the file it was loaded
// from, otherwise from the first file that contains it.
func gooseOwners(parts []goosePart) map[string]gooseOwner {
	owners := map[string]gooseOwner{}
	for _, preferLoaded := range []bool{true, false} {
		for index, part := range parts {
			for _, ied := range part.root.FindAll("IED") {
				name := ied.GetOr("name", "")
				if _, seen := owners[name]; !seen && (part.loaded[name] || !preferLoaded) {
					owners[name] = gooseOwner{index: index, ied: ied}
				}
			}
		}
	}
	return owners
}

// withoutOutside drops the unresolved-input messages about publishers that are in no file
// (they are listed as remote inputs instead).
func withoutOutside(diagnostics []any, remote []any) []any {
	outside := map[string]bool{}
	for _, item := range remote {
		outside[get(item.(*pyjson.Object), "publisher").(string)] = true
	}
	kept := []any{}
	for _, item := range diagnostics {
		message := item.(string)
		drop := false
		for other := range outside {
			if strings.Contains(message, "GOOSE block "+other+"/") || strings.Contains(message, "Input from "+other+"/") {
				drop = true
				break
			}
		}
		if !drop {
			kept = append(kept, message)
		}
	}
	return kept
}
