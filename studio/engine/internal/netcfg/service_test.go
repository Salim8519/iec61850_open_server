package netcfg

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

type serviceVectors struct {
	Times       []string `json:"times"`
	Exe         string   `json:"exe"`
	Program     string   `json:"program"`
	Gateway     string   `json:"gateway"`
	GatewayRule string   `json:"gateway_rule"`
	Scenarios   []struct {
		Name         string            `json:"name"`
		IEDs         json.RawMessage   `json:"ieds"`
		Env          json.RawMessage   `json:"env"`
		Steps        []json.RawMessage `json:"steps"`
		Results      []*outcome        `json:"results"`
		Journal      string            `json:"journal"`
		Requests     json.RawMessage   `json:"requests"`
		Probes       json.RawMessage   `json:"probes"`
		Firewall     json.RawMessage   `json:"firewall"`
		Launches     json.RawMessage   `json:"launches"`
		Events       json.RawMessage   `json:"events"`
		AdapterCalls int               `json:"adapter_calls"`
	} `json:"scenarios"`
}

// resolve is make_service.resolve: "$fp", "$clean", "$gw", "$rule" and "$gwrule" tokens.
func resolve(value any, last map[string]any, gatewayRule string) any {
	switch v := value.(type) {
	case string:
		switch {
		case v == "$gwrule":
			return gatewayRule
		case v == "$rule":
			names := []any{}
			for _, item := range objects(last["$fp_items"]) {
				if get(item, "kind") == "firewall" {
					names = append(names, get(item, "name"))
				}
			}
			return names
		case strings.HasPrefix(v, "$"):
			if found, ok := last[v]; ok {
				return found
			}
			return ""
		}
	case []any:
		items := make([]any, len(v))
		for index, item := range v {
			items[index] = resolve(item, last, gatewayRule)
		}
		return items
	case *pyjson.Object:
		object := pyjson.NewObject()
		for _, key := range v.Keys() {
			object.Set(key, resolve(get(v, key), last, gatewayRule))
		}
		return object
	}
	return value
}

func runServiceStep(service *NetworkService, gateway *GatewayFirewall, step []any) (*pyjson.Object, error) {
	arg := func(index int) any { return step[index] }
	switch step[0] {
	case "preview":
		return service.Preview(arg(1), pyStr(arg(2)), arg(3))
	case "apply":
		return service.Apply(arg(1), arg(2))
	case "adapters":
		adapters, err := service.Adapters(truthy(arg(1)))
		return pyjson.NewObject().Set("adapters", AdaptersObject(adapters)), err
	case "cleanup_preview":
		return service.CleanupPreview()
	case "cleanup":
		return service.Cleanup(arg(1), arg(2))
	case "gw_preview":
		return gateway.Preview()
	}
	return gateway.Apply(arg(1), arg(2))
}

// checkStudioFields compares the StudioError code and details Flask would send.
func checkStudioFields(t *testing.T, label string, err error, want *outcome) {
	var known *apperr.Error
	if !errors.As(err, &known) || known.Kind != apperr.Studio {
		return
	}
	if known.Code != want.Code {
		t.Errorf("%s: code %q want %v", label, known.Code, want.Code)
	}
	details := "[]"
	if known.Details != nil {
		details = dumpsText(t, known.Details)
	}
	if wantDetails := dumpsText(t, py(t, want.Details)); details != wantDetails {
		t.Errorf("%s: details %s want %s", label, details, wantDetails)
	}
}

func checkRecorded(t *testing.T, label string, got any, want json.RawMessage) {
	t.Helper()
	if gotText, wantText := dumpsText(t, got), dumpsText(t, py(t, want)); gotText != wantText {
		t.Errorf("%s:\n got %s\nwant %s", label, gotText, wantText)
	}
}

func TestServiceScenariosMatchPython(t *testing.T) {
	var v serviceVectors
	loadVectors(t, "service.json", &v)
	for _, scenario := range v.Scenarios {
		folder := t.TempDir()
		windows := &fakeWindows{t: t, env: pyObject(t, scenario.Env), folder: folder, probes: []any{},
			firewall: []any{}, launches: []any{}}
		projects := newFakeProjects(py(t, scenario.IEDs))
		service, gateway := newScenarioService(t, windows, projects, v)
		last := map[string]any{}
		for index, raw := range scenario.Steps {
			step := py(t, raw).([]any)
			label := fmt.Sprintf("%s step %d %v", scenario.Name, index, step[0])
			switch step[0] {
			case "set":
				windows.env.Set(pyStr(step[1]), resolve(step[2], last, v.GatewayRule))
				continue
			case "tick":
				seconds, _ := pyInt(step[1])
				windows.tick(float64(seconds.Int64()))
				continue
			}
			step = resolve(step, last, v.GatewayRule).([]any)
			result, err := runServiceStep(service, gateway, step)
			got := ""
			if err == nil {
				body, _ := pyjson.Flask(result)
				got = string(body)
				if tag := map[any]string{"preview": "$fp", "cleanup_preview": "$clean", "gw_preview": "$gw"}[step[0]]; tag != "" {
					last[tag], last[tag+"_items"] = get(result, "fingerprint"), get(result, "items")
				}
			}
			want := scenario.Results[index]
			checkOutcome(t, label, got, err, *want, true)
			checkStudioFields(t, label, err, want)
		}
		journal, _ := os.ReadFile(filepath.Join(folder, "network", "journal.jsonl"))
		wantJournal, _ := base64.StdEncoding.DecodeString(scenario.Journal)
		if !sameJournalBytes(journal, wantJournal) {
			t.Errorf("%s journal:\n got %s\nwant %s", scenario.Name, journal, wantJournal)
		}
		checkRecorded(t, scenario.Name+" requests", requestNames(folder), scenario.Requests)
		checkRecorded(t, scenario.Name+" probes", windows.probes, scenario.Probes)
		checkRecorded(t, scenario.Name+" firewall", windows.firewall, scenario.Firewall)
		checkRecorded(t, scenario.Name+" launches", windows.launches, scenario.Launches)
		checkRecorded(t, scenario.Name+" events", projects.events, scenario.Events)
		if windows.calls != scenario.AdapterCalls {
			t.Errorf("%s: %d adapter listings, want %d", scenario.Name, windows.calls, scenario.AdapterCalls)
		}
	}
}
