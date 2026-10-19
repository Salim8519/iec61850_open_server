package netcfg

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"studio-engine/internal/pyjson"
)

type planVectors struct {
	Normalize []struct {
		outcome
		Raw json.RawMessage `json:"raw"`
	} `json:"normalize"`
	Status [][2]json.RawMessage `json:"status"`
	Speed  [][2]string          `json:"speed"`
	MacRaw [][3]json.RawMessage `json:"mac"`
	Listed [][3]json.RawMessage `json:"listed"`
	Manual []struct {
		outcome
		Address json.RawMessage `json:"address"`
		Prefix  json.RawMessage `json:"prefix"`
	} `json:"manual"`
	OnLink []struct {
		outcome
		Address string          `json:"address"`
		Adapter json.RawMessage `json:"adapter"`
	} `json:"on_link"`
	AddressPlan []struct {
		outcome
		IEDs     json.RawMessage `json:"ieds"`
		Adapter  json.RawMessage `json:"adapter"`
		All      json.RawMessage `json:"all"`
		Strategy string          `json:"strategy"`
		Avoid    []string        `json:"avoid"`
	} `json:"address_plan"`
	BuildPlan []struct {
		Adapter  json.RawMessage `json:"adapter"`
		Rows     json.RawMessage `json:"rows"`
		Ports    []int           `json:"ports"`
		Program  string          `json:"program"`
		Existing []string        `json:"existing"`
		Out      string          `json:"out"`
		Texts    []map[string]outcome
	} `json:"build_plan"`
	GatewayPlan []struct {
		Program  string         `json:"program"`
		Existing []string       `json:"existing"`
		Blocking []BlockingRule `json:"blocking"`
		Out      string         `json:"out"`
		Texts    []map[string]outcome
	} `json:"gateway_plan"`
}

func TestNormalizeAndWordsMatchPython(t *testing.T) {
	var v planVectors
	loadVectors(t, "plan.json", &v)
	for index, c := range v.Normalize {
		raw := py(t, c.Raw)
		var adapters []Adapter
		var err error
		switch text := raw.(type) {
		case nil:
			adapters, err = NormalizeJSON("")
		case string:
			adapters, err = NormalizeJSON(text)
		default:
			adapters = Normalize(raw)
		}
		got := ""
		if err == nil {
			got = dumpsText(t, AdaptersObject(adapters))
		}
		checkOutcome(t, fmt.Sprintf("normalize %d", index), got, err, c.outcome, false)
	}
	for _, pair := range v.Status {
		var code int32
		var word string
		json.Unmarshal(pair[0], &code)
		json.Unmarshal(pair[1], &word)
		if got := StatusWord(code); got != word {
			t.Errorf("status %d: %q want %q", code, got, word)
		}
	}
	for _, pair := range v.Speed {
		bits, _ := strconv.ParseUint(pair[0], 10, 64)
		if got := SpeedWords(bits); got != pair[1] {
			t.Errorf("speed %s: %q want %q", pair[0], got, pair[1])
		}
	}
	for _, item := range v.MacRaw {
		var raw []byte
		var bytes [8]byte
		var length uint32
		var word string
		var ints []int
		json.Unmarshal(item[0], &ints)
		for index, value := range ints {
			raw = append(raw, byte(value))
			bytes[index] = byte(value)
		}
		json.Unmarshal(item[1], &length)
		json.Unmarshal(item[2], &word)
		if got := MACWords(bytes, length); got != word {
			t.Errorf("mac %v/%d: %q want %q", raw, length, got, word)
		}
	}
	for _, item := range v.Listed {
		var ifType uint32
		var description *string
		var listed bool
		json.Unmarshal(item[0], &ifType)
		json.Unmarshal(item[1], &description)
		json.Unmarshal(item[2], &listed)
		text := ""
		if description != nil {
			text = *description
		}
		if got := IsListed(ifType, text); got != listed {
			t.Errorf("listed %d %q: %v", ifType, text, got)
		}
	}
}

func TestAddressPlanMatchesPython(t *testing.T) {
	var v planVectors
	loadVectors(t, "plan.json", &v)
	for index, c := range v.Manual {
		ip, prefix, err := ValidateManual(py(t, c.Address), py(t, c.Prefix))
		got := ""
		if err == nil {
			got = dumpsText(t, []any{ip, prefix})
		}
		checkOutcome(t, fmt.Sprintf("manual %d %s/%s", index, c.Address, c.Prefix), got, err, c.outcome, true)
	}
	for index, c := range v.OnLink {
		linked, err := OnLink(c.Address, adapterFrom(pyObject(t, c.Adapter)))
		checkOutcome(t, fmt.Sprintf("on_link %d", index), dumpsText(t, linked), err, c.outcome, false)
	}
	for index, c := range v.AddressPlan {
		ieds := []PlanIED{}
		for _, object := range objects(py(t, c.IEDs)) {
			ieds = append(ieds, iedFrom(object))
		}
		avoid := map[string]bool{}
		for _, address := range c.Avoid {
			avoid[address] = true
		}
		rows, err := PlanAddresses(ieds, adapterFrom(pyObject(t, c.Adapter)), adaptersFrom(py(t, c.All)), c.Strategy, avoid)
		got := ""
		if err == nil {
			got = dumpsText(t, RowsObject(rows))
		}
		checkOutcome(t, fmt.Sprintf("address_plan %d", index), got, err, c.outcome, true)
	}
}

func rowFrom(object *pyjson.Object) Row {
	row := Row{Name: field(object, "name", ""), Proposed: field(object, "proposed", ""), Status: field(object, "status", "")}
	if prefix, ok := intField(object, "prefix"); ok {
		row.Prefix = intPointer(int(prefix))
	}
	return row
}

func checkTexts(t *testing.T, label string, items []*pyjson.Object, texts []map[string]outcome) {
	t.Helper()
	if len(items) != len(texts) {
		t.Fatalf("%s: %d items, want %d", label, len(items), len(texts))
	}
	for index, item := range items {
		for name, function := range map[string]func(*pyjson.Object) (string, error){
			"describe": Describe, "apply": ApplyCommand, "remove": RemoveCommand} {
			text, err := function(item)
			checkOutcome(t, fmt.Sprintf("%s item %d %s", label, index, name), text, err, texts[index][name], false)
		}
	}
}

func TestChangePlansMatchPython(t *testing.T) {
	var v planVectors
	loadVectors(t, "plan.json", &v)
	for index, c := range v.BuildPlan {
		rows := []Row{}
		for _, object := range objects(py(t, c.Rows)) {
			rows = append(rows, rowFrom(object))
		}
		plan, err := BuildPlan(adapterFrom(pyObject(t, c.Adapter)), rows, c.Ports, c.Program, c.Existing)
		if err != nil {
			t.Fatal(err)
		}
		label := fmt.Sprintf("build_plan %d", index)
		got := dumpsText(t, pyjson.NewObject().Set("items", itemsAny(plan.Items)).Set("fingerprint", plan.Fingerprint))
		if got != c.Out {
			t.Errorf("%s:\n got %s\nwant %s", label, got, c.Out)
		}
		checkTexts(t, label, plan.Items, c.Texts)
	}
	for index, c := range v.GatewayPlan {
		plan, err := GatewayPlan(c.Program, c.Existing, c.Blocking)
		if err != nil {
			t.Fatal(err)
		}
		label := fmt.Sprintf("gateway_plan %d", index)
		got := dumpsText(t, pyjson.NewObject().Set("items", itemsAny(plan.Items)).Set("fingerprint", plan.Fingerprint))
		if got != c.Out {
			t.Errorf("%s:\n got %s\nwant %s", label, got, c.Out)
		}
		checkTexts(t, label, plan.Items, c.Texts)
	}
}
