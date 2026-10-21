package snmpfile

import (
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"

	"studio-engine/internal/snmp/mib"
	"studio-engine/internal/snmp/model"
)

// csvHeader is the ScadaBench CSV header, in the order WriteCSV writes it.
var csvHeader = []string{"oid", "name", "type", "value", "hex", "behaviour", "rate", "writable"}

var errHeader = errors.New("the first line of the CSV must be the header " +
	"\"oid,name,type,value,hex,behaviour,rate,writable\"")

// ParseCSV reads ScadaBench CSV. A row may leave oid empty when name resolves, and type
// empty to use the MIB syntax (or octets). Invalid rows are reported and skipped; values
// are stored in canonical form. Rows keep the file order; for a repeated OID the later row
// wins. It fails when the header is wrong or no data row can be used.
func ParseCSV(text string, lib *mib.Library) ([]model.Point, []Problem, error) {
	lib = library(lib)
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(text, "\ufeff")))
	r.FieldsPerRecord = -1
	head, err := r.Read()
	if err != nil {
		return nil, nil, errHeader
	}
	cols := map[string]int{}
	for i, h := range head {
		name := strings.ToLower(strings.TrimSpace(h))
		if !validColumn(name) {
			return nil, nil, errHeader
		}
		cols[name] = i
	}
	_, hasOID := cols["oid"]
	_, hasName := cols["name"]
	if _, ok := cols["value"]; !ok || !(hasOID || hasName) {
		return nil, nil, errHeader
	}
	c := newCollector()
	rows := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		var pe *csv.ParseError
		if errors.As(err, &pe) {
			c.problem(pe.StartLine, "the CSV is not well formed here: "+pe.Err.Error())
			rows++
			continue
		} else if err != nil {
			return nil, c.problems, err
		}
		line, _ := r.FieldPos(0)
		rows++
		p, why := parseRow(lib, cols, rec)
		if why != "" {
			c.problem(line, why)
			continue
		}
		c.add(line, p)
	}
	if rows > 0 && len(c.points) == 0 {
		return nil, c.problems, errors.New("none of the " + strconv.Itoa(rows) +
			" CSV rows could be used; see the listed problems")
	}
	return c.points, c.problems, nil
}

func validColumn(name string) bool {
	for _, h := range csvHeader {
		if h == name {
			return true
		}
	}
	return false
}

// parseRow converts one CSV row to a point, or returns why it cannot be used.
func parseRow(lib *mib.Library, cols map[string]int, rec []string) (model.Point, string) {
	get := func(col string) string {
		if i, ok := cols[col]; ok && i < len(rec) {
			return rec[i]
		}
		return ""
	}
	oid := strings.TrimPrefix(strings.TrimSpace(get("oid")), ".")
	name := strings.TrimSpace(get("name"))
	switch {
	case oid != "" && !mib.ValidOID(oid):
		return model.Point{}, "oid \"" + oid + "\" is not a dotted number such as 1.3.6.1.2.1.1.5.0"
	case oid == "" && name == "":
		return model.Point{}, "the row has neither an oid nor a name"
	case oid == "":
		var ok bool
		if oid, ok = lib.Resolve(name); !ok {
			return model.Point{}, name + ": unknown name — import the MIB that defines it, or fill in the oid column"
		}
	}
	node, _ := lib.Lookup(oid)
	typ := strings.ToLower(strings.TrimSpace(get("type")))
	fromMIB := false
	if typ == "" {
		typ = model.TypeOctets
		if node != nil && node.Syntax != "" && (node.Kind == "object" || node.Kind == "column") {
			typ, fromMIB = node.Syntax, true
		}
	}
	if !validType(typ) {
		return model.Point{}, "type \"" + typ + "\" is unknown; use one of " + strings.Join(model.Types, ", ")
	}
	hexText := get("hex")
	hex, ok := parseBool(hexText)
	if !ok {
		return model.Point{}, "hex \"" + hexText + "\" is not true/false, yes/no or 1/0"
	}
	if strings.TrimSpace(hexText) == "" && fromMIB {
		hex = node.Hex
	}
	hex = hex && typ == model.TypeOctets
	value, why := checkValue(lib, oid, typ, get("value"), hex)
	if why != "" {
		return model.Point{}, why
	}
	behaviour := strings.ToLower(strings.TrimSpace(get("behaviour")))
	if behaviour == "" {
		behaviour = defaultBehaviour(oid)
		if typ != model.TypeTimeTicks {
			behaviour = model.BehaviourFixed
		}
	}
	if why := checkBehaviour(behaviour, typ); why != "" {
		return model.Point{}, why
	}
	rate, ok := parseRate(strings.TrimSpace(get("rate")))
	if !ok {
		return model.Point{}, "rate \"" + get("rate") + "\" is not a number"
	}
	writable, ok := parseBool(get("writable"))
	if !ok {
		return model.Point{}, "writable \"" + get("writable") + "\" is not true/false, yes/no or 1/0"
	}
	return model.Point{OID: oid, Type: typ, Value: value, Hex: hex, Behaviour: behaviour,
		Rate: rate, Writable: writable}, ""
}

// WriteCSV writes points as ScadaBench CSV with module-qualified names from lib. An empty
// behaviour is written as "fixed".
func WriteCSV(points []model.Point, lib *mib.Library) string {
	lib = library(lib)
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write(csvHeader)
	for _, p := range points {
		name := lib.Name(p.OID)
		if name == strings.TrimPrefix(p.OID, ".") {
			name = ""
		}
		behaviour := p.Behaviour
		if behaviour == "" {
			behaviour = model.BehaviourFixed
		}
		rate := ""
		if p.Rate != 0 {
			rate = strconv.FormatFloat(p.Rate, 'g', -1, 64)
		}
		_ = w.Write([]string{p.OID, name, p.Type, p.Value, strconv.FormatBool(p.Hex), behaviour,
			rate, strconv.FormatBool(p.Writable)})
	}
	w.Flush()
	return b.String()
}
