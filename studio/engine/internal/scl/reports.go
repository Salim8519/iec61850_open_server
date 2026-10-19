package scl

import (
	"fmt"
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/xmldoc"
)

type bit struct {
	key   string
	value int
}

var (
	reportTriggers = []bit{{"dchg", 1}, {"qchg", 2}, {"dupd", 4}, {"period", 8}, {"gi", 16}}
	reportOptions  = []bit{{"seqNum", 1}, {"timeStamp", 2}, {"reasonCode", 4}, {"dataSet", 8},
		{"dataRef", 16}, {"bufOvfl", 32}, {"entryID", 64}, {"configRef", 128}}
)

func bits(e *xmldoc.Element, pairs []bit) int {
	if e == nil {
		return 0
	}
	total := 0
	for _, pair := range pairs {
		if value, ok := e.Get(pair.key); ok && value == "true" {
			total += pair.value
		}
	}
	return total
}

// usableReports is reports.usable_reports: ReportControl elements with a non-blank name.
func usableReports(ln *xmldoc.Element) []*xmldoc.Element {
	var result []*xmldoc.Element
	for _, report := range ln.FindAll("ReportControl") {
		if pyrules.Strip(report.GetOr("name", "")) != "" {
			result = append(result, report)
		}
	}
	return result
}

// pyIntAttr is int(element.get(key, fallback)) with Python's ValueError text.
func pyIntAttr(e *xmldoc.Element, key, fallback string) (int64, error) {
	text := e.GetOr(key, fallback)
	value, err := pyrules.IntBig(text, 10)
	if err != nil {
		return 0, apperr.ValueError(pyrules.IntMessage(text))
	}
	if !value.IsInt64() {
		return 1 << 62, nil
	}
	return value.Int64(), nil
}

// compileReports is reports.compile_reports for one LN: DS/DE/RC lines and the dataset
// and report dicts.
func compileReports(ln *xmldoc.Element, ldInst string, lines *[]string, warnings *[]string) ([]*pyjson.Object, []*pyjson.Object, error) {
	var datasets, reports []*pyjson.Object
	names := map[string]bool{}
	for _, dataset := range ln.FindAll("DataSet") {
		value, present := dataset.Get("name")
		name, err := Identifier(value, present, "dataset")
		if err != nil {
			return nil, nil, err
		}
		*lines = append(*lines, "DS("+name+"){")
		members := []any{}
		for _, item := range dataset.FindAll("FCDA") {
			member, err := datasetMember(item, ldInst, lines)
			if err != nil {
				return nil, nil, err
			}
			members = append(members, member)
		}
		*lines = append(*lines, "}")
		datasets = append(datasets, pyjson.NewObject().Set("name", name).Set("members", members))
		names[name] = true
	}
	all := ln.FindAll("ReportControl")
	usable := usableReports(ln)
	if skipped := len(all) - len(usable); skipped > 0 && warnings != nil {
		*warnings = append(*warnings, fmt.Sprintf("%d report block(s) in %s/%s have no name in the file and were "+
			"skipped. Ask the IED engineer to fix the CID.", skipped, ldInst, ln.GetOr("lnClass", "LLN0")))
	}
	for _, report := range usable {
		compiled, err := compileReport(report, names, lines)
		if err != nil {
			return nil, nil, err
		}
		reports = append(reports, compiled...)
	}
	return datasets, reports, nil
}

func datasetMember(item *xmldoc.Element, ldInst string, lines *[]string) (string, error) {
	ldValue := item.GetOr("ldInst", ldInst)
	memberLD, err := Identifier(ldValue, true, "dataset logical device")
	if err != nil {
		return "", err
	}
	node := item.GetOr("prefix", "") + item.GetOr("lnClass", "") + item.GetOr("lnInst", "")
	if _, err := Identifier(node, true, "dataset logical node"); err != nil {
		return "", err
	}
	fc := item.GetOr("fc", "ST")
	objectPath := item.GetOr("doName", "")
	attribute := item.GetOr("daName", "")
	parts := strings.Split(objectPath, ".")
	if attribute != "" {
		parts = append(parts, strings.Split(attribute, ".")...)
	}
	for _, part := range parts {
		if _, err := Identifier(part, true, "dataset member"); err != nil {
			return "", err
		}
	}
	_, indexed := item.Get("ix")
	memberParts := parts
	if indexed {
		memberParts = strings.Split(objectPath, ".")
	}
	member := strings.Join(append([]string{node, fc}, memberParts...), "$")
	if memberLD != ldInst {
		member = memberLD + "/" + member
	}
	index, component := "", ""
	if indexed {
		ix, err := pyIntAttr(item, "ix", "")
		if err != nil {
			return "", err
		}
		index = " " + strconv.FormatInt(ix, 10)
		if ix < 0 {
			return "", apperr.New("Dataset array index must be non-negative")
		}
		if attribute != "" {
			component = " " + strings.ReplaceAll(attribute, ".", "$")
		}
	}
	*lines = append(*lines, "DE("+member+index+component+");")
	return member + index + component, nil
}

func compileReport(report *xmldoc.Element, datasets map[string]bool, lines *[]string) ([]*pyjson.Object, error) {
	value, present := report.Get("name")
	name, err := Identifier(value, present, "report control")
	if err != nil {
		return nil, err
	}
	ds := "-"
	if datSet, ok := report.Get("datSet"); ok && datSet != "" {
		if ds, err = Identifier(datSet, true, "report dataset"); err != nil {
			return nil, err
		}
		if !datasets[ds] {
			return nil, apperr.New("Report " + name + " refers to unresolved dataset " + ds)
		}
	}
	buffered := report.GetOr("buffered", "") == "true"
	triggers := bits(report.Find("TrgOps"), reportTriggers)
	options := bits(report.Find("OptFields"), reportOptions)
	count := int64(1)
	if enabled := report.Find("RptEnabled"); enabled != nil {
		if count, err = pyIntAttr(enabled, "max", "1"); err != nil {
			return nil, err
		}
	}
	if count < 1 || count > 64 {
		return nil, apperr.New("Invalid report client count for " + name)
	}
	var result []*pyjson.Object
	for index := int64(0); index < count; index++ {
		finalName := name
		if report.GetOr("indexed", "true") == "true" {
			finalName = fmt.Sprintf("%s%02d", name, index+1)
		}
		rptID := report.GetOr("rptID", finalName)
		if rptID == "" {
			rptID = finalName
		}
		if pyrules.HasSpace(rptID) || strings.ContainsAny(rptID, "();{}") {
			return nil, apperr.New("Unsupported report ID syntax: " + rptID)
		}
		confRev, err := pyIntAttr(report, "confRev", "1")
		if err != nil {
			return nil, err
		}
		numbers := []int64{}
		for _, field := range []struct{ key, fallback string }{{"bufTime", "0"}, {"intgPd", "1000"}} {
			number, err := pyIntAttr(report, field.key, field.fallback)
			if err != nil {
				return nil, err
			}
			numbers = append(numbers, number)
		}
		*lines = append(*lines, fmt.Sprintf("RC(%s %s %d %s %d %d %d %d %d);", finalName, rptID,
			boolInt(buffered), ds, confRev, triggers, options, numbers[0], numbers[1]))
		result = append(result, pyjson.NewObject().Set("name", finalName).Set("dataset", ds).
			Set("buffered", buffered).Set("conf_rev", confRev))
		if report.GetOr("indexed", "") == "false" {
			break
		}
	}
	return result, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
