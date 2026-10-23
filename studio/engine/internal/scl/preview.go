package scl

import (
	"runtime"
	"strings"
	"sync"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

// Preview is document.preview_ieds: one summary per IED without compiling models.
func Preview(source []byte) ([]*pyjson.Object, error) {
	doc, err := ParseDocument(source, false)
	if err != nil {
		return nil, err
	}
	var result []*pyjson.Object
	failures := map[string]string{}
	if adr0114() {
		failures = ImportFailures(doc, source)
	}
	for _, ied := range doc.Root.FindAll("IED") {
		servers := ied.FindAll("AccessPoint/Server")
		reason := ""
		switch {
		case len(servers) == 0:
			reason = "No server model in this IED"
		case len(servers) > 1:
			reason = "Multiple servers are not supported yet"
		default:
			if failure := failures[ied.GetOr("name", "")]; strings.HasPrefix(failure, "at ") {
				reason = "Cannot be imported " + failure // ADR-0114: flagged before the import
			} else if failure != "" {
				reason = "Cannot be imported: " + failure
			}
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

// ImportFailures builds every IED with one server as the import would (without the XSD
// check) and returns, per IED name, why it would fail; IEDs that build are left out. The IEDs
// are built in parallel (each builder is its own; the document is only read), so a 30-IED
// station previews in about the time of its largest IED.
func ImportFailures(doc *xmldoc.Document, source []byte) map[string]string {
	failures := map[string]string{}
	root := doc.Root
	types, err := declaredTypes(root)
	if err == nil {
		_, err = pyrules.DecodeUTF8Sig(source)
	}
	if err != nil {
		for _, ied := range root.FindAll("IED") {
			failures[ied.GetOr("name", "")] = err.Error()
		}
		return failures
	}
	text := ""
	run := &importRun{root: root, source: source, filename: "", types: types, text: &text,
		catalogue: publisherCatalogue(root), validation: pyjson.NewObject().Set("status", "valid"),
		newID: func() string { return "" }}
	var mu sync.Mutex
	var wait sync.WaitGroup
	slots := make(chan struct{}, runtime.NumCPU())
	for _, ied := range root.FindAll("IED") {
		servers := ied.FindAll("AccessPoint/Server")
		if len(servers) != 1 {
			continue
		}
		wait.Add(1)
		slots <- struct{}{}
		go func(ied, server *xmldoc.Element) {
			defer func() { <-slots; wait.Done() }()
			name := ied.GetOr("name", "")
			if _, err := run.ied(ied, name, server); err != nil {
				mu.Lock()
				failures[name] = err.Error()
				mu.Unlock()
			}
		}(ied, servers[0])
	}
	wait.Wait()
	return failures
}

var unloadableCache sourceCache

// UnloadableIEDs says, per IED name of the file that the project could never load, why: no
// server, several servers or an import failure (ADR-0114). Importable IEDs are left out; a
// name with no IED section is answered by the caller. Remembered per source text.
func UnloadableIEDs(source string) (map[string]string, error) {
	value, err := unloadableCache.get("", "unloadable", source, func() (any, error) {
		doc, err := ParseDocument([]byte(source), false)
		if err != nil {
			return nil, err
		}
		reasons := map[string]string{}
		failures := ImportFailures(doc, []byte(source))
		for _, ied := range doc.Root.FindAll("IED") {
			name := ied.GetOr("name", "")
			switch servers := ied.FindAll("AccessPoint/Server"); {
			case len(servers) == 0:
				reasons[name] = "it has no server model, so it cannot be simulated"
			case len(servers) > 1:
				reasons[name] = "it has more than one server, which is not supported yet"
			case strings.HasPrefix(failures[name], "at "):
				reasons[name] = "importing it fails " + failures[name]
			case failures[name] != "":
				reasons[name] = "importing it fails: " + failures[name]
			}
		}
		return reasons, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(map[string]string), nil
}
