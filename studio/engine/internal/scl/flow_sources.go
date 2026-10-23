package scl

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/xmldoc"
)

// Flow is flow_sources.read_flow: what one SCL file declares about signal flow.
type Flow struct {
	IEDs          []*pyjson.Object // {name, desc, manufacturer} in document order
	GoCBs         []*pyjson.Object // publishable GoCBs with "members" texts
	Subscriptions map[string]FlowSubscriptions
	Reports       []*pyjson.Object // report_clients.report_blocks
}

// FlowSubscriptions is goose_subscriptions' (subscriptions, diagnostics) of one IED.
type FlowSubscriptions struct {
	Subscriptions []any
	Problems      []any
}

// ReadFlow is read_flow(source).
func ReadFlow(source string) (*Flow, error) {
	parsed, err := ParseDocument([]byte(source), false)
	if err != nil {
		return nil, err
	}
	root := parsed.Root
	cat := publisherCatalogue(root)
	flow := &Flow{Subscriptions: map[string]FlowSubscriptions{}}
	known := map[string]bool{}
	for _, ied := range root.FindAll("IED") {
		known[ied.GetOr("name", "")] = true
	}
	for _, ied := range root.FindAll("IED") {
		found, problems := gooseSubscriptions(ied, cat)
		problems = append(problems, outsideSources(ied, known)...)
		flow.Subscriptions[ied.GetOr("name", "")] = FlowSubscriptions{found, problems}
		flow.IEDs = append(flow.IEDs, pyjson.NewObject().Set("name", ied.GetOr("name", "")).
			Set("desc", ied.GetOr("desc", "")).Set("manufacturer", ied.GetOr("manufacturer", "")))
	}
	for _, entry := range cat.entries {
		control := pyjson.NewObject()
		for _, key := range entry.control.Keys() {
			control.Set(key, get(entry.control, key))
		}
		members := []any{}
		for _, fcda := range entry.fcdas {
			members = append(members, memberText(fcda))
		}
		flow.GoCBs = append(flow.GoCBs, control.Set("members", members))
	}
	flow.Reports = reportBlocks(root)
	return flow, nil
}

// memberText is flow_sources._fcda / report_clients._member: "LD/LN.do.da [FC]".
func memberText(fcda *xmldoc.Element) string {
	node := fcda.GetOr("prefix", "") + fcda.GetOr("lnClass", "") + fcda.GetOr("lnInst", "")
	var parts []string
	for _, key := range []string{"doName", "daName"} {
		if value := fcda.GetOr(key, ""); value != "" {
			parts = append(parts, value)
		}
	}
	return fcda.GetOr("ldInst", "") + "/" + node + "." + strings.Join(parts, ".") + " [" + fcda.GetOr("fc", "") + "]"
}

// reportBlocks is report_clients.report_blocks: every named ReportControl with dataset
// members and declared clients.
func reportBlocks(root *xmldoc.Element) []*pyjson.Object {
	var blocks []*pyjson.Object
	for _, ied := range root.FindAll("IED") {
		name := ied.GetOr("name", "")
		for _, device := range ied.FindAll("AccessPoint/Server/LDevice") {
			domain := device.GetOr("ldName", "")
			if domain == "" {
				domain = name + device.GetOr("inst", "")
			}
			for _, ln := range device.Children {
				if ln.Tag != "LN" && ln.Tag != "LN0" {
					continue
				}
				datasets := map[string]*xmldoc.Element{} // {item.get("name"): item}; None never matches
				for _, item := range ln.FindAll("DataSet") {
					if key, present := item.Get("name"); present {
						datasets[key] = item
					}
				}
				for _, report := range usableReports(ln) {
					blocks = append(blocks, reportBlock(report, ln, name, domain, datasets))
				}
			}
		}
	}
	return blocks
}

func reportBlock(report, ln *xmldoc.Element, publisher, domain string, datasets map[string]*xmldoc.Element) *pyjson.Object {
	buffered := report.GetOr("buffered", "") == "true"
	dataset := report.GetOr("datSet", "")
	found := datasets[dataset]
	clients := []any{}
	if enabled := report.Find("RptEnabled"); enabled != nil {
		for _, client := range enabled.FindAll("ClientLN") {
			entry := pyjson.NewObject()
			for _, key := range []string{"iedName", "ldInst", "prefix", "lnClass", "lnInst"} {
				entry.Set(key, client.GetOr(key, ""))
			}
			clients = append(clients, entry)
		}
	}
	members := []any{}
	if found != nil {
		for _, fcda := range found.FindAll("FCDA") {
			members = append(members, memberText(fcda))
		}
	}
	kind := "RP"
	if buffered {
		kind = "BR"
	}
	reportName, _ := report.Get("name")
	return pyjson.NewObject().Set("publisher", publisher).Set("name", reportName).
		Set("ref", domain+"/"+lnName(ln)+"."+kind+"."+reportName).Set("buffered", buffered).
		Set("dataset", dataset).Set("dataset_found", found != nil).Set("members", members).Set("clients", clients)
}
