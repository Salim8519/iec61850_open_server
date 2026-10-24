package scl

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

// publisher is one entry of goose_links.publisher_catalogue.
type publisher struct {
	key     [3]string
	control *pyjson.Object // control without "members", plus "publisher"
	fcdas   []*xmldoc.Element
}

// catalogue keeps Python dict semantics: insertion order, later duplicates replace values.
type catalogue struct {
	entries []*publisher
	index   map[[3]string]int
}

func publisherCatalogue(root *xmldoc.Element) *catalogue {
	result := &catalogue{index: map[[3]string]int{}}
	for _, ied := range root.FindAll("IED") {
		name := ied.GetOr("name", "")
		comm, _ := readCommunication(root, name)
		for _, ld := range stationDevices(root, ied) {
			ln0 := ld.Find("LN0")
			if ln0 == nil {
				continue
			}
			inst := ld.GetOr("inst", "")
			domain := ld.GetOr("ldName", "")
			if domain == "" {
				domain = name + inst
			}
			var datasets []*pyjson.Object
			fcdas := map[string][]*xmldoc.Element{}
			for _, item := range ln0.FindAll("DataSet") {
				if missingDoName(item) != nil {
					continue // not published by the importer (ADR-0114)
				}
				dsName := item.GetOr("name", "")
				datasets = append(datasets, pyjson.NewObject().Set("name", dsName).Set("members", []any{}))
				if _, seen := fcdas[dsName]; !seen {
					fcdas[dsName] = item.FindAll("FCDA")
				}
			}
			for _, control := range gooseControls(ln0, domain, inst, comm.gse, datasets, name, nil, gooseGaps{}) {
				entry := pyjson.NewObject()
				for _, key := range control.Keys() {
					if key != "members" {
						entry.Set(key, get(control, key))
					}
				}
				entry.Set("publisher", name)
				result.put(&publisher{key: [3]string{name, inst, get(control, "name").(string)}, control: entry,
					fcdas: fcdas[get(control, "dataset").(string)]})
			}
		}
	}
	return result
}

func (c *catalogue) put(p *publisher) {
	if i, ok := c.index[p.key]; ok {
		c.entries[i] = p
		return
	}
	c.index[p.key] = len(c.entries)
	c.entries = append(c.entries, p)
}

func attrOrEmpty(e *xmldoc.Element, key string) string { return e.GetOr(key, "") }

// matches is goose_links._matches.
func matches(extref, fcda *xmldoc.Element) bool {
	for _, key := range []string{"ldInst", "prefix", "lnClass", "lnInst", "doName"} {
		if attrOrEmpty(extref, key) != attrOrEmpty(fcda, key) {
			return false
		}
	}
	given, wanted := attrOrEmpty(fcda, "daName"), attrOrEmpty(extref, "daName")
	return given == "" || wanted == "" || wanted == given || strings.HasPrefix(wanted, given+".")
}

// member is goose_links._member: dataset position and the remaining path, or nil.
func member(extref *xmldoc.Element, fcdas []*xmldoc.Element) (any, string) {
	for index, fcda := range fcdas {
		if matches(extref, fcda) {
			wanted, given := attrOrEmpty(extref, "daName"), attrOrEmpty(fcda, "daName")
			if strings.HasPrefix(wanted, given) {
				return int64(index), strings.TrimLeft(wanted[len(given):], ".")
			}
			return int64(index), ""
		}
	}
	return nil, ""
}

// lgosTargets is goose_links._lgos_targets.
func lgosTargets(root, ied *xmldoc.Element) map[string]string {
	targets := map[string]string{}
	for _, ld := range stationDevices(root, ied) {
		domain := ld.GetOr("ldName", "")
		if domain == "" {
			domain = ied.GetOr("name", "") + ld.GetOr("inst", "")
		}
		for _, ln := range ld.FindAll("LN") {
			if ln.GetOr("lnClass", "") != "LGOS" {
				continue
			}
			if value, ok := setSrcRef(ln); ok && pyrules.Strip(value) != "" {
				target := domain + "/" + lnName(ln)
				targets[pyrules.Strip(value)] = target
				if mms := gocbMMSRef(pyrules.Strip(value)); mms != "" && adr0114() {
					if _, taken := targets[mms]; !taken {
						targets[mms] = target
					}
				}
			}
		}
	}
	return targets
}

// gocbMMSRef turns the ObjectReference form of a GoCB ("LD/LLN0.gocb0", as SCD tools write
// setSrcRef) into the MMS form the GOOSE blocks use ("LD/LLN0$GO$gocb0"); "" when the value
// is not that form (ADR-0114).
func gocbMMSRef(value string) string {
	slash := strings.IndexByte(value, '/')
	if slash <= 0 || strings.Contains(value, "$") {
		return ""
	}
	rest := value[slash+1:]
	if !strings.HasPrefix(rest, "LLN0.") || strings.Count(rest, ".") != 1 || len(rest) == len("LLN0.") {
		return ""
	}
	return value[:slash] + "/LLN0$GO$" + rest[len("LLN0."):]
}

// setSrcRef is findtext("DOI[@name='GoCBRef']/DAI[@name='setSrcRef']/Val").
func setSrcRef(ln *xmldoc.Element) (string, bool) {
	for _, doi := range ln.FindAll("DOI") {
		if doi.GetOr("name", "\x00") != "GoCBRef" {
			continue
		}
		for _, dai := range doi.FindAll("DAI") {
			if dai.GetOr("name", "\x00") != "setSrcRef" {
				continue
			}
			if val := dai.Find("Val"); val != nil {
				return val.Text, true
			}
		}
	}
	return "", false
}
