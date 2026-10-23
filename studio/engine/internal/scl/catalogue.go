package scl

import (
	"sort"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/xmldoc"
)

type ldDomain struct {
	value   string
	present bool
}

// enrichReports is report_catalogue.enrich_reports: dataset references, exact member
// signals, unresolved members and the report → dataset links, in import order.
func enrichReports(server *xmldoc.Element, iedName string, datasets, reports []*pyjson.Object, signals []any) {
	byFC := map[string][]string{}
	for _, item := range signals {
		signal := item.(*pyjson.Object)
		fc := get(signal, "fc").(string)
		byFC[fc] = append(byFC[fc], get(signal, "ref").(string))
	}
	for _, refs := range byFC {
		sort.Strings(refs)
	}
	devices := server.FindAll("LDevice")
	domains := map[string]string{}
	for _, ld := range devices {
		inst := ld.GetOr("inst", "")
		if name := ld.GetOr("ldName", ""); name != "" {
			domains[inst] = name
		} else {
			domains[inst] = iedName + inst
		}
	}
	nextDataset, nextReport := 0, 0
	for _, ld := range devices {
		inst := ld.GetOr("inst", "")
		for _, ln := range ld.Children {
			if ln.Tag != "LN" && ln.Tag != "LN0" {
				continue
			}
			owner := domains[inst] + "/" + lnName(ln)
			local := map[string]*pyjson.Object{}
			unusable := unusableDatasets(ln)
			for _, declaration := range ln.FindAll("DataSet") {
				if missingDoName(declaration) != nil {
					continue // left out by compileReports (ADR-0114)
				}
				dataset := datasets[nextDataset]
				nextDataset++
				enrichDataset(dataset, declaration, inst, owner, domains, byFC)
				local[get(dataset, "name").(string)] = dataset
			}
			for _, declaration := range usableReports(ln) {
				if unusable[declaration.GetOr("datSet", "")] {
					continue
				}
				count := int64(1)
				if enabled := declaration.Find("RptEnabled"); enabled != nil {
					count, _ = pyIntAttr(enabled, "max", "1")
				}
				if declaration.GetOr("indexed", "") == "false" {
					count = 1
				}
				for i := int64(0); i < count; i++ {
					linkReport(reports[nextReport], owner, local)
					nextReport++
				}
			}
		}
	}
}

func lnName(ln *xmldoc.Element) string {
	if ln.Tag == "LN0" {
		return "LLN0"
	}
	return ln.GetOr("prefix", "") + ln.GetOr("lnClass", "") + ln.GetOr("inst", "")
}

func enrichDataset(dataset *pyjson.Object, declaration *xmldoc.Element, inst, owner string,
	domains map[string]string, byFC map[string][]string) {
	resolved := map[string]bool{}
	unresolved, fcdaRefs := []any{}, []any{}
	for _, member := range declaration.FindAll("FCDA") {
		domain, known := domains[member.GetOr("ldInst", inst)]
		path := member.GetOr("doName", "")
		if ix, ok := member.Get("ix"); ok {
			value, _ := pyIntAttr(member, "ix", ix)
			path += "(" + strconv.FormatInt(value, 10) + ")"
		}
		if da := member.GetOr("daName", ""); da != "" {
			path += "." + da
		}
		shown := domain
		if !known || domain == "" {
			shown = "<unresolved LD>"
		}
		reference := shown + "/" + member.GetOr("prefix", "") + member.GetOr("lnClass", "") +
			member.GetOr("lnInst", "") + "." + path
		fcdaRefs = append(fcdaRefs, reference)
		fc := member.GetOr("fc", "ST")
		refs := byFC[fc]
		matched := false
		if known && domain != "" {
			for i := sort.SearchStrings(refs, reference); i < len(refs) && strings.HasPrefix(refs[i], reference); i++ {
				ref := refs[i]
				if ref == reference || strings.HasPrefix(ref, reference+".") || strings.HasPrefix(ref, reference+"(") {
					resolved[ref] = true
					matched = true
				}
			}
		}
		if !matched {
			unresolved = append(unresolved, pyjson.NewObject().Set("ref", reference).Set("fc", fc))
		}
	}
	signalRefs := make([]string, 0, len(resolved))
	for ref := range resolved {
		signalRefs = append(signalRefs, ref)
	}
	sort.Strings(signalRefs)
	refs := make([]any, len(signalRefs))
	for i, ref := range signalRefs {
		refs[i] = ref
	}
	dataset.Set("ref", owner+"$"+get(dataset, "name").(string)).Set("signal_refs", refs).
		Set("unresolved", unresolved).Set("fcda_refs", fcdaRefs)
}

func linkReport(report *pyjson.Object, owner string, local map[string]*pyjson.Object) {
	block := "RP."
	if get(report, "buffered").(bool) {
		block = "BR."
	}
	report.Set("ref", owner+"."+block+get(report, "name").(string))
	dataset := local[get(report, "dataset").(string)]
	if dataset == nil {
		report.Set("dataset_ref", nil).Set("signal_count", int64(0)).Set("unresolved", []any{})
		return
	}
	report.Set("dataset_ref", get(dataset, "ref")).
		Set("signal_count", int64(len(get(dataset, "signal_refs").([]any)))).
		Set("unresolved", get(dataset, "unresolved"))
}
