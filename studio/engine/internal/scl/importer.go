package scl

import (
	"crypto/rand"
	"encoding/hex"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/xmldoc"
)

// Importer is scl/importer.import_scl with its two outside effects injected: the XSD
// validator and the IED id generator (uuid4().hex in Python).
type Importer struct {
	Validator SchemaValidator
	NewID     func() string
}

// RandomID is uuid4().hex.
func RandomID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return hex.EncodeToString(b[:])
}

// Selection is the optional selected_names argument (nil Names with All=true = None).
type Selection struct {
	All   bool
	Names []any
}

// SelectAll imports every IED with a server.
var SelectAll = Selection{All: true}

// Import builds one model per selected IED, exactly import_scl(source, filename, selected).
func (im *Importer) Import(source []byte, filename string, selection Selection) ([]*pyjson.Object, error) {
	doc, err := ParseDocument(source, false)
	if err != nil {
		return nil, err
	}
	validation, err := ValidateSchema(source, doc, im.Validator)
	if err != nil {
		return nil, err
	}
	root := doc.Root
	selected := map[string]bool{}
	if !selection.All {
		if err := checkSelection(root, selection.Names, selected); err != nil {
			return nil, err
		}
	}
	declarations := root.Find("DataTypeTemplates")
	if declarations == nil {
		return nil, apperr.New("Missing DataTypeTemplates")
	}
	types := map[typeKey]*xmldoc.Element{}
	for _, declaration := range declarations.Children {
		id, present := declaration.Get("id")
		key := typeKey{declaration.Tag, id, present}
		if types[key] != nil {
			return nil, apperr.New("Duplicate type declaration " + noneOr(id, present))
		}
		types[key] = declaration
	}
	newID := im.NewID
	if newID == nil {
		newID = RandomID
	}
	run := &importRun{root: root, source: source, filename: filename, types: types, validation: validation, newID: newID}
	var result []*pyjson.Object
	names := map[string]bool{}
	for _, ied := range root.FindAll("IED") {
		value, present := ied.Get("name")
		name, err := Identifier(value, present, "IED name")
		if err != nil {
			return nil, err
		}
		if !selection.All && !selected[name] {
			continue
		}
		if names[name] {
			return nil, apperr.New("Duplicate IED name " + name)
		}
		names[name] = true
		servers := ied.FindAll("AccessPoint/Server")
		if len(servers) == 0 {
			if !selection.All {
				return nil, apperr.New("IED " + name + " has no server model")
			}
			continue
		}
		if len(servers) != 1 {
			return nil, apperr.New("IED " + name + " has multiple servers; select an access point first")
		}
		model, err := run.ied(ied, name, servers[0])
		if err != nil {
			return nil, err
		}
		result = append(result, model)
	}
	if len(result) == 0 {
		return nil, apperr.New("No IED servers found in SCL")
	}
	return result, nil
}

func checkSelection(root *xmldoc.Element, names []any, selected map[string]bool) error {
	invalid := apperr.New("Select one or more unique IED names")
	if len(names) == 0 {
		return invalid
	}
	for _, item := range names {
		name, ok := item.(string)
		if !ok {
			return invalid
		}
		if selected[name] {
			return invalid
		}
		selected[name] = true
	}
	available := map[string]bool{}
	for _, ied := range root.FindAll("IED") {
		if name, ok := ied.Get("name"); ok {
			available[name] = true
		}
	}
	for name := range selected {
		if !available[name] {
			return apperr.New("Selected IED is not present in this file")
		}
	}
	return nil
}
