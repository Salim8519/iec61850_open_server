package goosedom

import (
	"sort"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// inputFields are the input keys a link carries, in Python order.
var inputFields = [...]string{"int_addr", "desc", "source"}

func linkInput(entry *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range inputFields {
		result.Set(key, pyv.Get(entry, key))
	}
	return result
}

// gooseLinks is _goose_links: links declared by subscriptions within the project.
func gooseLinks(d Declarations, involved stringSet) []*pyjson.Object {
	var links []*pyjson.Object
	for _, subscriber := range sortedKeys(d.Subscriptions) {
		subscriptions, _ := d.Subscriptions.Get(subscriber)
		for _, item := range pyv.Objects(pyv.List(subscriptions)) {
			publisher := pyv.Get(item, "publisher")
			if !involved[subscriber] && !involved[pyv.PyStr(publisher)] {
				continue
			}
			matches := map[string]bool{}
			inputs := []any{}
			for _, entry := range pyv.Objects(pyv.GetList(item, "inputs")) {
				matches[dictKey(pyv.Get(entry, "match"))] = true
				inputs = append(inputs, linkInput(entry))
			}
			match := "data"
			if len(matches) == 1 && matches[dictKey("named")] {
				match = "named"
			}
			status := pyv.Get(item, "status")
			if pyv.Equal(status, "sendable") {
				status = "declared"
			}
			links = append(links, pyv.O("id", linkID("goose", pyv.Get(item, "gocb_ref"), subscriber),
				"gocb_ref", pyv.Get(item, "gocb_ref"), "publisher", publisher, "subscriber", subscriber,
				"block", pyv.Get(item, "name"), "status", status, "remote", false, "match", match, "inputs", inputs))
		}
	}
	return links
}

// remoteLinks is _remote_links: inputs from publishers outside every file, one link per
// (subscriber, publisher, ld_inst, cb_name) in first-seen order.
func remoteLinks(d Declarations, involved stringSet) []*pyjson.Object {
	var links []*pyjson.Object
	grouped := map[string]*pyjson.Object{}
	for _, subscriber := range sortedKeys(d.Remote) {
		if !involved[subscriber] {
			continue
		}
		inputs, _ := d.Remote.Get(subscriber)
		for _, entry := range pyv.Objects(pyv.List(inputs)) {
			hint := pyv.Get(entry, "hint")
			var mac any = ""
			if pyv.Truthy(hint) {
				mac = pyv.Get(pyv.Obj(hint), "mac")
			}
			publisher, ldInst, cbName := pyv.Get(entry, "publisher"), pyv.Get(entry, "ld_inst"), pyv.Get(entry, "cb_name")
			key := tupleKey(subscriber, publisher, ldInst, cbName)
			link, found := grouped[key]
			if !found {
				link = newRemoteLink(subscriber, publisher, ldInst, cbName, hint, mac)
				grouped[key] = link
				links = append(links, link)
			}
			if !pyv.Truthy(pyv.Get(link, "mac")) {
				link.Set("mac", mac)
			}
			link.Set("inputs", append(pyv.GetList(link, "inputs"), linkInput(entry)))
		}
	}
	return links
}

func newRemoteLink(subscriber string, publisher, ldInst, cbName, hint, mac any) *pyjson.Object {
	var gocbRef any
	if pyv.Truthy(cbName) && pyv.Truthy(ldInst) {
		gocbRef = pyv.PyStr(publisher) + pyv.PyStr(ldInst) + "/LLN0$GO$" + pyv.PyStr(cbName)
	}
	match := "data"
	if pyv.Truthy(cbName) {
		match = "named"
	} else if pyv.Truthy(hint) {
		match = "vendor_hint"
	}
	return pyv.O("id", linkID("remote", subscriber, publisher, ldInst, cbName), "publisher", publisher,
		"subscriber", subscriber, "block", cbName, "status", "remote", "remote", true, "mac", mac,
		"gocb_ref", gocbRef, "match", match, "inputs", []any{})
}

// BuildLinkMap returns {ieds, blocks, links, diagnostics} limited to links touching a
// project IED. projectIEDs are {id, name, role} records. Pure and deterministic.
func BuildLinkMap(d Declarations, projectIEDs []*pyjson.Object) *pyjson.Object {
	project := map[string]*pyjson.Object{}
	involved := stringSet{}
	for _, ied := range projectIEDs {
		name := pyv.PyStr(pyv.Get(ied, "name"))
		project[name] = ied
		involved[name] = true
	}
	links := append(gooseLinks(d, involved), remoteLinks(d, involved)...)
	sort.SliceStable(links, func(i, j int) bool { return linkLess(links[i], links[j]) })
	named := stringSet{}
	for name := range involved {
		named[name] = true
	}
	for _, link := range links {
		named[pyv.PyStr(pyv.Get(link, "publisher"))] = true
		named[pyv.PyStr(pyv.Get(link, "subscriber"))] = true
	}
	blocks := []*pyjson.Object{}
	for _, entry := range sortedBlocks(d.Blocks) {
		if len(entry.Key) > 0 && named[entry.Key[0]] {
			blocks = append(blocks, blockView(entry.Block))
		}
	}
	ieds := []any{}
	for _, name := range named.sorted() {
		known := pyv.Obj(pyv.Get(d.IEDs, name))
		ied, inProject := project[name]
		var role any
		if ied != nil && ied.Len() > 0 {
			role = pyv.GetOr(ied, "role", "simulated")
		}
		ieds = append(ieds, pyv.O("name", name, "ied_id", pyv.Get(ied, "id"), "role", role,
			"in_project", inProject, "remote", pyv.Get(d.IEDs, name) == nil,
			"desc", pyv.GetOr(known, "desc", ""), "manufacturer", pyv.GetOr(known, "manufacturer", "")))
	}
	diagnostics := append(linkDiagnostics(links, d, involved), addressDiagnostics(blocks)...)
	return pyv.O("ieds", ieds, "blocks", objectList(blocks), "links", objectList(links), "diagnostics", diagnostics)
}

// linkLess orders links by (publisher, block or "", subscriber).
func linkLess(left, right *pyjson.Object) bool {
	a := []string{pyv.PyStr(pyv.Get(left, "publisher")), orEmpty(pyv.Get(left, "block")),
		pyv.PyStr(pyv.Get(left, "subscriber"))}
	b := []string{pyv.PyStr(pyv.Get(right, "publisher")), orEmpty(pyv.Get(right, "block")),
		pyv.PyStr(pyv.Get(right, "subscriber"))}
	return tupleLess(a, b)
}

func orEmpty(value any) string {
	if !pyv.Truthy(value) {
		return ""
	}
	return pyv.PyStr(value)
}

func objectList(items []*pyjson.Object) []any {
	result := make([]any, len(items))
	for index, item := range items {
		result[index] = item
	}
	return result
}
