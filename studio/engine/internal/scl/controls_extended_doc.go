package scl

import (
	"hash/fnv"
	"strconv"
	"sync"

	"studio-engine/internal/apperr"
	"studio-engine/internal/scl/xmldoc"
)

// extendedResult is ExtendedControls' answer for one IED name.
type extendedResult struct {
	list []any
	err  error
}

// extendedForDocument gives, for every IED name of one SCL text, exactly what ExtendedControls
// gives for that name, reading the text once and building only the data objects whose type can
// hold an INC/ENC/APC/BAC/BSC/ISC control (ADR-0116). Run start used to read the whole file
// and build every signal again for each IED of it.
func extendedForDocument(source []byte) (map[string]extendedResult, error) {
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
	holds := typedHolders{types: types, known: map[typeKey]bool{}}
	results := map[string]extendedResult{}
	for _, ied := range doc.Root.FindAll("IED") {
		name := ied.GetOr("name", "")
		if _, done := results[name]; done {
			continue // ExtendedControls answers from the first IED of a name
		}
		servers := ied.FindAll("AccessPoint/Server")
		if len(servers) != 1 {
			results[name] = extendedResult{list: []any{}}
			continue
		}
		b := newBuilder(types)
		b.skipObject = holds.skip
		if err := walkObjects(b, servers[0], name); err != nil {
			results[name] = extendedResult{err: err}
			continue
		}
		if b.extended == nil {
			results[name] = extendedResult{list: []any{}}
		} else {
			results[name] = extendedResult{list: b.extended}
		}
	}
	return results, nil
}

// typedHolders tells whether a DO's type, or any SDO type below it, has a typed control class.
type typedHolders struct {
	types map[typeKey]*xmldoc.Element
	known map[typeKey]bool
}

// skip is true only for a DO whose whole type tree resolves and holds no typed control class;
// anything unresolved is built as before, so its errors are unchanged.
func (h typedHolders) skip(object *xmldoc.Element) bool {
	id, present := object.Get("type")
	return !h.holds(typeKey{"DOType", id, present}, 0)
}

func (h typedHolders) holds(key typeKey, depth int) bool {
	if result, found := h.known[key]; found {
		return result
	}
	definition := h.types[key]
	if definition == nil || depth > 32 {
		return true
	}
	cdc, _ := definition.Get("cdc")
	_, result := extendedShapes[cdc]
	for _, child := range definition.Children {
		if result {
			break
		}
		if child.Tag == "SDO" {
			id, present := child.Get("type")
			result = h.holds(typeKey{"DOType", id, present}, depth+1)
		}
	}
	h.known[key] = result
	return result
}

// documentCalls shares one extendedForDocument per source text: IEDs of the same file start
// together, the first one reads it and the others wait for that answer.
var documentCalls = struct {
	mu    sync.Mutex
	calls map[string]*documentCall
}{calls: map[string]*documentCall{}}

type documentCall struct {
	done    chan struct{}
	results map[string]extendedResult
	err     error
}

func extendedFromDocument(source, iedName string) ([]any, error) {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(source))
	key := strconv.FormatUint(hash.Sum64(), 16) + "|" + strconv.Itoa(len(source))
	documentCalls.mu.Lock()
	call := documentCalls.calls[key]
	if call == nil {
		if len(documentCalls.calls) >= sourceCacheLimit {
			documentCalls.calls = map[string]*documentCall{}
		}
		call = &documentCall{done: make(chan struct{})}
		documentCalls.calls[key] = call
		documentCalls.mu.Unlock()
		call.results, call.err = extendedForDocument([]byte(source))
		close(call.done)
	} else {
		documentCalls.mu.Unlock()
		<-call.done
	}
	if call.err != nil {
		return nil, call.err
	}
	result, found := call.results[iedName]
	if !found {
		return []any{}, nil
	}
	return result.list, result.err
}
