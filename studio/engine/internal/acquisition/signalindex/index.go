// Package signalindex answers GET /api/v1/ieds/<ied_id>/signals (api/signals.py
// list_signals) from an index built once per immutable IED engineering model. Current
// point values are read through a ValueReader, so live values never rebuild the index.
// Finding the IED is the caller's job: an unknown id answers
// StudioError("Unknown IED", code="not_found") (apperr.Coded(..., "not_found")).
package signalindex

import (
	"sort"
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// signal is the prebuilt, read-only view of one ied["signals"] item.
type signal struct {
	object     *pyjson.Object
	ref        string
	fc         string
	fcOK       bool // fc is a string
	kind       string
	kindOK     bool // kind is a string
	writable   bool // signal["writable"] and signal["fc"] != "CF"
	ld         string
	haystack   string // (ref + " " + description).casefold()
	qualityRef any
	timeRef    any
}

// Index is the per-IED search index. It never changes after Build and is safe for
// concurrent use; it shares (and never modifies) the model's signal objects.
type Index struct {
	ied       *pyjson.Object
	signals   []signal
	byRef     map[string][]int          // positions of every signal with that reference
	last      map[string]*pyjson.Object // {signal["ref"]: signal}: the last one wins
	devices   []string                  // sorted logical devices of refs containing "/"
	reports   map[string]memberSet      // report_members of each report reference
	inDataset memberSet                 // refs of every dataset's signal_refs
	text      textIndex                 // all haystacks for the q filter
}

// memberSet is a set of signal references with the positions of the signals it selects.
type memberSet struct {
	refs      map[string]bool
	positions []int
}

// Build indexes one IED model. References and descriptions are expected to be strings, as
// the SCL importer writes them (Python would raise TypeError for others; here they are
// read with str()).
func Build(ied *pyjson.Object) *Index {
	objects := pyv.Objects(pyv.GetList(ied, "signals"))
	ix := &Index{ied: ied, signals: make([]signal, len(objects)), byRef: make(map[string][]int, len(objects)),
		last: make(map[string]*pyjson.Object, len(objects)), reports: map[string]memberSet{}}
	devices := map[string]bool{}
	haystacks := make([]string, len(objects))
	for position, object := range objects {
		item := signal{object: object, ref: pyv.PyStr(pyv.Get(object, "ref")),
			qualityRef: pyv.Get(object, "quality_ref"), timeRef: pyv.Get(object, "timestamp_ref")}
		item.fc, item.fcOK = pyv.Get(object, "fc").(string)
		item.kind, item.kindOK = pyv.Get(object, "kind").(string)
		item.writable = pyv.Truthy(pyv.Get(object, "writable")) && !(item.fcOK && item.fc == "CF")
		item.ld, _, _ = strings.Cut(item.ref, "/")
		if strings.Contains(item.ref, "/") {
			devices[item.ld] = true
		}
		haystacks[position] = pyv.Casefold(item.ref + " " + pyv.PyStr(pyv.Get(object, "description")))
		ix.signals[position] = item
		ix.byRef[item.ref] = append(ix.byRef[item.ref], position)
		ix.last[item.ref] = object
	}
	ix.text, haystacks = newTextIndex(haystacks)
	for position, haystack := range haystacks {
		ix.signals[position].haystack = haystack
	}
	for device := range devices {
		ix.devices = append(ix.devices, device)
	}
	sort.Strings(ix.devices)
	ix.indexDatasets()
	return ix
}

// indexDatasets prebuilds report_members for every report and the in_dataset set.
func (ix *Index) indexDatasets() {
	datasets := pyv.Objects(pyv.List(pyv.GetOr(ix.ied, "datasets", []any{})))
	all := map[string]bool{}
	for _, dataset := range datasets {
		for _, ref := range pyv.List(pyv.Get(dataset, "signal_refs")) {
			all[pyv.PyStr(ref)] = true
		}
	}
	ix.inDataset = ix.members(all)
	for _, report := range pyv.Objects(pyv.GetList(ix.ied, "reports")) {
		ref, ok := pyv.Get(report, "ref").(string)
		if _, seen := ix.reports[ref]; !ok || seen {
			continue // report_members takes the first report with that reference
		}
		refs := map[string]bool{}
		for _, dataset := range datasets {
			if pyv.Equal(pyv.Get(dataset, "ref"), pyv.Get(report, "dataset_ref")) {
				for _, member := range pyv.List(pyv.Get(dataset, "signal_refs")) {
					refs[pyv.PyStr(member)] = true
				}
				break
			}
		}
		ix.reports[ref] = ix.members(refs)
	}
}

// members returns the set with the positions, in model order, of the signals it selects.
func (ix *Index) members(refs map[string]bool) memberSet {
	set := memberSet{refs: refs}
	for ref := range refs {
		set.positions = append(set.positions, ix.byRef[ref]...)
	}
	sort.Ints(set.positions)
	return set
}

// Devices returns the sorted logical devices (a new list).
func (ix *Index) Devices() []string { return append([]string(nil), ix.devices...) }

// Len is the number of signals of the model.
func (ix *Index) Len() int { return len(ix.signals) }

// IED returns the model object the index was built from.
func (ix *Index) IED() *pyjson.Object { return ix.ied }
