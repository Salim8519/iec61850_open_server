package netcfg

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// fakeProjects is make_service.Projects: simulated IEDs and recorded project changes.
type fakeProjects struct {
	ieds   []PlanIED
	mode   string
	events []any
}

func newFakeProjects(raw any) *fakeProjects {
	projects := &fakeProjects{mode: "loopback", events: []any{}}
	for _, object := range objects(raw) {
		if role, found := object.Get("role"); !found || role == "simulated" {
			projects.ieds = append(projects.ieds, iedFrom(object))
		}
	}
	return projects
}

func (p *fakeProjects) SimulatedIEDs() ([]PlanIED, error) { return p.ieds, nil }

func (p *fakeProjects) EnterLabMode(rows []Row, adapter Adapter) error {
	for _, row := range rows {
		p.events = append(p.events, []any{"set", row.IEDID, "lab_ip", row.Proposed})
	}
	p.events = append(p.events, []any{"network", pyjson.NewObject().Set("mode", "lab").
		Set("adapter_guid", adapter.GUID).Set("adapter_label", adapter.Name)})
	p.mode = "lab"
	return nil
}

func (p *fakeProjects) LeaveLabMode() error {
	if p.mode == "lab" {
		p.events = append(p.events, []any{"network", pyjson.NewObject().Set("mode", "loopback")})
		p.mode = "loopback"
	}
	return nil
}

// fakeWindows is make_service.fake_env: the Windows state a scenario changes with "set".
type fakeWindows struct {
	t        *testing.T
	env      *pyjson.Object
	folder   string
	probes   []any
	firewall []any
	launches []any
	calls    int
	tick     func(seconds float64)
}

// read returns env[key], or raises the "!Class:message" error the scenario asks for.
func (w *fakeWindows) read(key string) (any, error) {
	value, _ := w.env.Get(key)
	if text, ok := value.(string); ok && strings.HasPrefix(text, "!") {
		kind, message, _ := strings.Cut(text[1:], ":")
		return nil, map[string]error{"RuntimeError": apperr.RuntimeError(message),
			"ValueError": apperr.ValueError(message), "KeyError": apperr.KeyError(message)}[kind]
	}
	return value, nil
}

func (w *fakeWindows) adapters() ([]Adapter, error) {
	w.calls++
	value, err := w.read("adapters")
	return adaptersFrom(value), err
}

func (w *fakeWindows) names(key string) func() ([]string, error) {
	return func() ([]string, error) {
		value, err := w.read(key)
		return stringList(value), err
	}
}

func (w *fakeWindows) blocking(program string) ([]BlockingRule, error) {
	value, err := w.read("blocking")
	rules := []BlockingRule{}
	for _, rule := range objects(value) {
		rules = append(rules, BlockingRule{Name: field(rule, "name", ""), Display: field(rule, "display", "")})
	}
	return rules, err
}

func (w *fakeWindows) probe(addresses []string) (map[string]bool, error) {
	w.probes = append(w.probes, toAny(addresses))
	answering, _ := w.read("probe")
	used := map[string]bool{}
	for _, address := range stringList(answering) {
		used[address] = true
	}
	found := map[string]bool{}
	for _, address := range addresses {
		if used[address] {
			found[address] = true
		}
	}
	return found, nil
}

func (w *fakeWindows) state(names []string) (map[string]bool, map[string]bool, error) {
	w.firewall = append(w.firewall, toAny(names))
	value, _ := w.read("state")
	state, _ := value.(*pyjson.Object)
	return textSet(get(state, "gateway")), textSet(get(state, "disabled")), nil
}

// elevate answers like the generator's fake helper for the current "elevate" mode.
func (w *fakeWindows) elevate(executable string, arguments []string) (uint32, error) {
	argv := []any{executable}
	for _, argument := range arguments {
		argv = append(argv, strings.ReplaceAll(argument, w.folder, "<DATA>"))
	}
	request := arguments[1]
	payload, err := os.ReadFile(request)
	if err != nil {
		w.t.Fatal(err)
	}
	w.launches = append(w.launches, pyjson.NewObject().Set("argv", argv).
		Set("payload", base64.StdEncoding.EncodeToString(payload)))
	mode, _ := w.read("elevate")
	switch mode {
	case "cancel":
		return 0, ErrElevationCancelled
	case "runtime":
		return 0, apperr.RuntimeError("The network helper did not finish within two minutes")
	case "none":
		return 0, nil
	}
	decoded, _ := pyjson.Decode(payload)
	results := []any{}
	for index, item := range objects(get(decoded.(*pyjson.Object), "items")) {
		key, _ := ItemKey(item)
		failed := mode == "fail" && index == 1
		message := ""
		if failed {
			message = "Access denied"
		}
		entry := pyjson.NewObject().Set("key", key).Set("ok", !failed).Set("message", message)
		if mode == "no_ok" {
			entry = pyjson.NewObject().Set("key", key).Set("message", "m")
		}
		if mode != "partial" || index == 0 {
			results = append(results, entry)
		}
	}
	outcome := pyjson.NewObject().Set("refused", false).Set("results", results)
	if mode == "refused" {
		outcome = pyjson.NewObject().Set("refused", true).Set("reason", "Unknown helper action").Set("results", []any{})
	}
	data, _ := pyjson.Dumps(outcome)
	return 0, os.WriteFile(withSuffix(request, ".result.json"), data, 0o666)
}

// newScenarioService wires a NetworkService and GatewayFirewall to the fakes.
func newScenarioService(t *testing.T, w *fakeWindows, projects *fakeProjects, v serviceVectors) (*NetworkService, *GatewayFirewall) {
	ids, now := 0, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	service := NewNetworkService(projects, w.folder, v.Program, v.Exe)
	service.Journal.Now = fixedClock(t, v.Times)
	service.Source = AdapterFunc(w.adapters)
	service.Rules = w.names("rules")
	service.Elevator = ElevatorFunc(w.elevate)
	service.Probe = ProberFunc(w.probe)
	service.Firewall = w.state
	service.NewID = func() string { ids++; return fmt.Sprintf("%032x", ids) }
	service.Clock = func() time.Time { return now }
	gateway := &GatewayFirewall{Network: service, Program: v.Gateway, Rules: w.names("gateway_rules"), Blocking: w.blocking}
	w.tick = func(seconds float64) { now = now.Add(time.Duration(seconds * float64(time.Second))) }
	return service, gateway
}

func requestNames(folder string) []any {
	entries, _ := os.ReadDir(filepath.Join(folder, "network", "requests"))
	names := []any{}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}
