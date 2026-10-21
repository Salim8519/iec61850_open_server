package licensing

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

type cannedAnswer struct {
	Return json.RawMessage   `json:"return"`
	Raise  []json.RawMessage `json:"raise"`
}

type scenarioStep struct {
	Op   string          `json:"op"`
	Now  *float64        `json:"now"`
	Arg  json.RawMessage `json:"arg"`
	Path string          `json:"path"`
	Data *string         `json:"data"`
	Hex  string          `json:"hex"`
	// Trial steps: start_trial(email, code), verify_online_if_due(trial_only).
	Email     json.RawMessage `json:"email"`
	Code      json.RawMessage `json:"code"`
	TrialOnly bool            `json:"trial_only"`
}

type scenarioResult struct {
	HTTP  int               `json:"http"`
	Body  string            `json:"body"`
	Value json.RawMessage   `json:"value"`
	Crash string            `json:"crash"`
	Calls [][]any           `json:"calls"`
	Files map[string]string `json:"files"`
	// Registry is the in-memory HKCU\Software\ScadaBench after the step (registry scenarios).
	Registry map[string]string `json:"registry"`
}

type scenario struct {
	Name     string          `json:"name"`
	Enforce  *bool           `json:"enforce"`
	HWID     json.RawMessage `json:"hwid"`
	Legacy   *string         `json:"legacy"`
	Keys     string          `json:"keys"`
	Debugger bool            `json:"debugger"`
	Registry bool            `json:"registry"`
	// TestServer is the test-only seam (Options.AllowTestServer): a server on this PC is allowed.
	TestServer bool                      `json:"test_server"`
	Expected   map[string]string         `json:"expected"`
	Client     map[string][]cannedAnswer `json:"client"`
	Steps      []scenarioStep            `json:"steps"`
	Results    []scenarioResult          `json:"results"`
}

// fakeClient replays the canned license-server answers and records every call.
type fakeClient struct {
	t       *testing.T
	answers map[string][]cannedAnswer
	calls   *[][]any
}

func (f *fakeClient) answer(name string, args ...any) (any, error) {
	*f.calls = append(*f.calls, append([]any{name}, args...))
	queue := f.answers[name]
	if len(queue) == 0 {
		return nil, &ServerError{Message: "no canned answer", Code: "offline"}
	}
	f.answers[name] = queue[1:]
	if queue[0].Raise != nil {
		var message string
		json.Unmarshal(queue[0].Raise[0], &message)
		return nil, &ServerError{Message: message, Code: mustDecode(f.t, string(queue[0].Raise[1]))}
	}
	return mustDecode(f.t, string(queue[0].Return)), nil
}

func (f *fakeClient) StartTrial(hwid string, parts *pyjson.Object, email, code any) (any, error) {
	return f.answer("trial", hwid, plainJSON(parts), plainJSON(email), plainJSON(code))
}
func (f *fakeClient) RequestTrialCode(hwid, email string) (any, error) {
	return f.answer("email-code", hwid, email)
}
func (f *fakeClient) Activate(key, hwid string) (any, error) {
	return f.answer("activate", key, hwid)
}
func (f *fakeClient) Verify(key any, hwid, nonce string) (any, error) {
	return f.answer("verify", key, hwid, nonce)
}

// plainJSON turns a decoded value into what encoding/json decodes from the recorded call
// (objects become maps), so fmt.Sprint compares Go and Python calls alike.
func plainJSON(value any) any {
	if value == nil {
		return nil
	}
	data, _ := pyjson.Compact(value)
	var plain any
	json.Unmarshal(data, &plain)
	return plain
}

// scenarioParts is scenario_defs.PARTS: what the parts provider answers.
func scenarioParts() *pyjson.Object {
	return pyjson.NewObject().Set("guid", strings.Repeat("1", 64)).Set("volume", "").
		Set("firmware", strings.Repeat("f", 64))
}

func newScenarioService(t *testing.T, sc scenario, root string, clock *float64, calls *[][]any,
	registry Registry) *Service {
	native := filepath.Join(root, "native")
	os.MkdirAll(native, 0o777)
	os.WriteFile(filepath.Join(native, "studio_ied.exe"), []byte("worker"), 0o666)
	answers := map[string][]cannedAnswer{}
	for name, queue := range sc.Client {
		answers[name] = append([]cannedAnswer(nil), queue...)
	}
	options := Options{Home: filepath.Join(root, "home"), DataDir: filepath.Join(root, "data"),
		Enforce: sc.Enforce == nil || *sc.Enforce, Clock: func() float64 { return *clock },
		Keys: keysNamed(t, sc.Keys), NativeFolder: native, ExpectedFiles: sc.Expected,
		Debugger: func() bool { return sc.Debugger }, Logf: t.Logf, Registry: registry,
		AllowTestServer: sc.TestServer, Nonce: func() string { return strings.Repeat("0123456789abcdef", 2) },
		Parts: func() (*pyjson.Object, error) { return scenarioParts(), nil },
		ClientFactory: func(url string) Client {
			*calls = append(*calls, []any{"factory", url})
			return &fakeClient{t: t, answers: answers, calls: calls}
		}}
	hwid := strings.Repeat("ab", 32)
	options.HWID = func() (string, error) { return hwid, nil }
	if string(sc.HWID) == "null" {
		options.HWID = func() (string, error) { return "", errors.New("Machine licensing is available on Windows only") }
	}
	if options.Keys == nil {
		options.Keys = testKeys(t)
	}
	return New(options)
}

func stepArg(t *testing.T, raw json.RawMessage) any {
	if raw == nil {
		return ""
	}
	return mustDecode(t, string(raw))
}

// optionalArg is step.get(key): nil when the step has no such field.
func optionalArg(t *testing.T, raw json.RawMessage) any {
	if raw == nil {
		return nil
	}
	return mustDecode(t, string(raw))
}

func runScenarioStep(t *testing.T, service *Service, step scenarioStep, appVersion string) (scenarioResult, error) {
	var result *pyjson.Object
	var err error
	switch step.Op {
	case "status":
		body, err := service.StatusBody(appVersion)
		return scenarioResult{HTTP: 200, Body: string(body)}, err
	case "licensed":
		licensed, err := service.Licensed()
		return scenarioResult{Value: json.RawMessage(fmt.Sprint(licensed))}, err
	case "verify":
		verdict, err := service.VerifyOnlineIfDue(step.TrialOnly)
		return scenarioResult{Value: json.RawMessage(compactText(t, verdict))}, err
	case "trial":
		result, err = service.StartTrial(optionalArg(t, step.Email), optionalArg(t, step.Code))
	case "email_code":
		result, err = service.RequestTrialCode(stepArg(t, step.Arg))
	case "check":
		result, err = service.CheckOnlineNow()
	case "activate":
		result, err = service.Activate(stepArg(t, step.Arg))
	case "offline":
		result, err = service.ActivateOffline(stepArg(t, step.Arg))
	case "config":
		result, err = service.SetServerURL(stepArg(t, step.Arg))
	default:
		t.Fatalf("unknown op %s", step.Op)
	}
	if err != nil {
		status, body, ok := ErrorResponse(err)
		if !ok {
			return scenarioResult{}, err
		}
		data, _ := pyjson.Flask(body)
		return scenarioResult{HTTP: status, Body: string(data)}, nil
	}
	data, _ := pyjson.Flask(result)
	return scenarioResult{HTTP: 200, Body: string(data)}, nil
}
