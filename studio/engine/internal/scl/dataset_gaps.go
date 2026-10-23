package scl

import "studio-engine/internal/scl/xmldoc"

// A dataset member written without doName (FCDA@doName is optional in the XSD: "all data of
// the logical node") cannot be expanded into members without guessing, so its dataset is
// left out with a warning and the report and GOOSE blocks that use it are not published;
// the rest of the IED imports (ADR-0114).

// missingDoName returns the first FCDA of a dataset without doName, or nil.
func missingDoName(dataset *xmldoc.Element) *xmldoc.Element {
	if !adr0114() {
		return nil
	}
	for _, item := range dataset.FindAll("FCDA") {
		if item.GetOr("doName", "") == "" {
			return item
		}
	}
	return nil
}

// unusableDatasets are the names of an LN's datasets that have a member without doName.
func unusableDatasets(ln *xmldoc.Element) map[string]bool {
	result := map[string]bool{}
	for _, dataset := range ln.FindAll("DataSet") {
		if missingDoName(dataset) != nil {
			result[dataset.GetOr("name", "")] = true
		}
	}
	return result
}

// datasetGapWarning names the dataset and the member written without doName.
func datasetGapWarning(ldInst, ln, name string, item *xmldoc.Element) string {
	member := item.GetOr("ldInst", ldInst) + "/" + item.GetOr("prefix", "") + item.GetOr("lnClass", "") +
		item.GetOr("lnInst", "") + " [" + item.GetOr("fc", "ST") + "]"
	return "Dataset " + ldInst + "/" + ln + "." + name + " has a member without doName (" + member +
		", meaning all data of that logical node). ScadaBench does not guess its members, so this dataset " +
		"and the report and GOOSE blocks that use it are not published; the rest of the IED is imported."
}
