package scl

import (
	"strconv"

	"studio-engine/internal/scl/xmldoc"
)

// outsideSources are the signal-flow problems for ExtRefs without serviceType and srcCBName
// that name an IED with no section in the file: their source cannot be resolved (spec
// signal-flow, "an ExtRef whose source can't be resolved"; ADR-0114). One problem per source
// IED, in document order; ExtRefs with serviceType GOOSE or srcCBName already get one from
// gooseSubscriptions.
func outsideSources(ied *xmldoc.Element, known map[string]bool) []any {
	var order []string
	counts, first := map[string]int{}, map[string]string{}
	for _, ld := range ied.FindAll("AccessPoint/Server/LDevice") {
		for _, ln := range ld.Children {
			for _, extref := range ln.FindAll("Inputs/ExtRef") {
				source := extref.GetOr("iedName", "")
				if _, typed := extref.Get("serviceType"); typed || source == "" || known[source] ||
					extref.GetOr("srcCBName", "") != "" {
					continue
				}
				if counts[source] == 0 {
					order = append(order, source)
					first[source] = remoteSource(extref, source)
				}
				counts[source]++
			}
		}
	}
	problems := []any{}
	for _, source := range order {
		problems = append(problems, strconv.Itoa(counts[source])+" input(s) take data from IED "+source+
			", which is not in this file, so their source cannot be resolved (first: "+first[source]+")")
	}
	return problems
}
