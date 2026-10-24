package scl

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

// importRun holds what one import_scl call shares between its IEDs.
type importRun struct {
	root       *xmldoc.Element
	source     []byte
	filename   any
	types      map[typeKey]*xmldoc.Element
	validation *pyjson.Object
	newID      func() string
	catalogue  *catalogue
	text       *string
}

// ied builds one IED model in import_scl's order and with its exact key order.
func (r *importRun) ied(ied *xmldoc.Element, name string, server *xmldoc.Element) (*pyjson.Object, error) {
	b := newBuilder(r.types)
	b.lines = append(b.lines, "MODEL("+name+"){")
	var datasets, reports, publishes, streams []*pyjson.Object
	comm, warnings := readCommunication(r.root, name)
	for _, ld := range server.FindAll("LDevice") {
		value, present := ld.Get("inst")
		inst, err := LDInst(value, present, "logical device")
		if err != nil {
			return nil, err
		}
		ldName := ld.GetOr("ldName", "")
		if ldName != "" {
			if _, err := Identifier(ldName, true, "functional logical device name"); err != nil {
				return nil, err
			}
		}
		domain := ldName
		header := "LD(" + inst + "){"
		if ldName == "" {
			domain = name + inst
		} else {
			header = "LD(" + inst + " " + ldName + "){"
		}
		b.lines = append(b.lines, header)
		for _, ln := range ld.Children {
			if ln.Tag != "LN" && ln.Tag != "LN0" {
				continue
			}
			node := lnName(ln)
			if _, err := Identifier(node, true, "logical node"); err != nil {
				return nil, located(err, domain)
			}
			typeID, hasType := ln.Get("lnType")
			definition, err := b.declaration("LNodeType", typeID, hasType)
			if err != nil {
				return nil, located(err, domain+"/"+node)
			}
			b.lines = append(b.lines, "LN("+node+"){")
			for _, object := range definition.FindAll("DO") {
				objectName, hasName := object.Get("name")
				var instance *xmldoc.Element
				for _, doi := range ln.FindAll("DOI") {
					if value, present := doi.Get("name"); present == hasName && value == objectName {
						instance = doi
						break
					}
				}
				if err := b.dataObject(object, domain+"/"+node, instance, 0); err != nil {
					return nil, err
				}
			}
			ds, rs, err := compileReports(ln, inst, &b.lines, &warnings)
			if err != nil {
				return nil, located(err, domain+"/"+node)
			}
			datasets = append(datasets, ds...)
			reports = append(reports, rs...)
			if ln.Tag == "LN0" {
				gaps := gooseGaps{rejected: comm.rejected, datasets: unusableDatasets(ln)}
				controls := gooseControls(ln, domain, inst, comm.gse, ds, name, &warnings, gaps)
				for _, control := range controls {
					b.lines = append(b.lines, gooseCfgLines(control)...)
				}
				publishes = append(publishes, controls...)
				streams = append(streams, svControls(ln, domain, inst, comm.smv, name, &warnings, comm.rejectedSMV)...)
			}
			b.lines = append(b.lines, "}")
		}
		b.lines = append(b.lines, "}")
	}
	b.lines = append(b.lines, "}")
	enrichReports(server, name, datasets, reports, b.signals)
	for _, item := range b.signals {
		attachCompanions(item.(*pyjson.Object), b.byReference)
	}
	warnings = append(warnings, b.controlWarnings...)
	warnings = append(warnings, b.notes.warnings()...)
	if r.catalogue == nil {
		r.catalogue = publisherCatalogue(r.root)
	}
	subscribes, diagnostics := gooseSubscriptions(r.root, ied, r.catalogue)
	if status := get(r.validation, "status").(string); status != "valid" {
		first := get(r.validation, "errors").([]any)[0].(*pyjson.Object)
		warnings = append(warnings, "Source document XSD validation: "+status+". "+get(first, "message").(string))
	}
	if r.text == nil {
		text, err := pyrules.DecodeUTF8Sig(r.source)
		if err != nil {
			return nil, err
		}
		r.text = &text
	}
	edition := int64(2)
	if _, ok := r.root.Get("version"); !ok {
		edition = 1
	}
	warningList := make([]any, len(warnings))
	for i, text := range warnings {
		warningList[i] = text
	}
	return pyjson.NewObject().Set("id", r.newID()).Set("name", name).Set("filename", r.filename).
		Set("source", *r.text).Set("cfg", strings.Join(b.lines, "\n")+"\n").Set("signals", b.signals).
		Set("controls", b.controls).Set("datasets", objects(datasets)).Set("reports", objects(reports)).
		Set("warnings", warningList).Set("validation", r.validation).Set("scl_address", comm.address).
		Set("goose", pyjson.NewObject().Set("publishes", objects(publishes)).Set("subscribes", subscribes).
			Set("diagnostics", diagnostics)).
		Set("sv", pyjson.NewObject().Set("publishes", objects(streams))).
		Set("edition", edition).Set("ip", "").Set("port", int64(102)).Set("state", "stopped"), nil
}

func objects(items []*pyjson.Object) []any {
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}

// attachCompanions is domain/sample.companions applied as import_scl does.
func attachCompanions(signal *pyjson.Object, byReference map[string]*pyjson.Object) {
	found := map[string]*pyjson.Object{}
	fc := get(signal, "fc")
	parent := get(signal, "ref").(string)
	if i := strings.LastIndexByte(parent, '.'); i >= 0 {
		parent = parent[:i]
	}
	for strings.Contains(parent, ".") {
		for _, pair := range [][2]string{{"q", "Quality"}, {"t", "Timestamp"}} {
			candidate := byReference[parent+"."+pair[0]]
			if found[pair[0]] == nil && candidate != nil && get(candidate, "kind") == pair[1] && get(candidate, "fc") == fc {
				found[pair[0]] = candidate
			}
		}
		parent = parent[:strings.LastIndexByte(parent, '.')]
	}
	refOf := func(key string) any {
		if found[key] == nil {
			return nil
		}
		return get(found[key], "ref")
	}
	valueOf := func(key string) any {
		if found[key] == nil {
			return int64(0)
		}
		return get(found[key], "value")
	}
	signal.Set("quality_ref", refOf("q")).Set("timestamp_ref", refOf("t")).
		Set("quality", valueOf("q")).Set("timestamp", valueOf("t"))
}
