package platformapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// golden is testdata/generated/http_golden.json, written by testdata/http_gen.py from the
// frozen Flask app (create_app + test client).
type golden struct {
	AppVersion string          `json:"app_version"`
	Current    string          `json:"current"`
	ServerURL  string          `json:"server_url"`
	HWID       string          `json:"hwid"`
	Program    string          `json:"program"`
	Gateway    string          `json:"gateway"`
	NetworkEnv json.RawMessage `json:"network_env"`
	TestKey    struct {
		Kid    string `json:"kid"`
		Public string `json:"public"`
	} `json:"test_key"`
	Steps []step `json:"steps"`
}

// step is a request with Flask's answer, or a change to the fakes (see http_gen.py).
type step struct {
	Op          string            `json:"op"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Body        *string           `json:"body"`
	ContentType string            `json:"content_type"`
	Status      int               `json:"status"`
	Headers     map[string]string `json:"headers"`
	Response    string            `json:"response"`
	Value       json.RawMessage   `json:"value"`
	Key         string            `json:"key"`
	Enforce     bool              `json:"enforce"`
	Data        string            `json:"data"`
	Hex         string            `json:"hex"`
	Answers     json.RawMessage   `json:"answers"`
	Queue       json.RawMessage   `json:"queue"`
	State       string            `json:"state"`
	Seconds     float64           `json:"seconds"`
}

func loadGolden(t testing.TB) *golden {
	data, err := os.ReadFile("testdata/generated/http_golden.json")
	if err != nil {
		t.Fatalf("golden missing (run testdata/http_gen.py): %v", err)
	}
	var g golden
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}
	return &g
}

func decode(t testing.TB, raw json.RawMessage) any {
	value, err := pyjson.Decode(raw)
	if err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return value
}

func number(t testing.TB, raw json.RawMessage) float64 {
	switch value := decode(t, raw).(type) {
	case int64:
		return float64(value)
	case float64:
		return value
	}
	t.Fatalf("not a number: %s", raw)
	return 0
}

func (s step) request() *http.Request {
	body := ""
	if s.Body != nil {
		body = *s.Body
	}
	request := httptest.NewRequest(s.Method, "http://localhost"+s.Path, strings.NewReader(body))
	if s.ContentType != "" {
		request.Header.Set("Content-Type", s.ContentType)
	}
	return request
}

// TestPlatformRoutesAgainstFlask replays every recorded step in order against the Go area
// and compares the status, the user-visible headers and the exact body bytes.
func TestPlatformRoutesAgainstFlask(t *testing.T) {
	g := loadGolden(t)
	h := newHarness(t, g)
	requests, failures := 0, 0
	for index, s := range g.Steps {
		if s.Op != "request" {
			h.setup(s)
			continue
		}
		requests++
		recorder := httptest.NewRecorder()
		h.mux.ServeHTTP(recorder, s.request())
		if problems := compare(recorder, s); len(problems) > 0 {
			failures++
			t.Errorf("step %d %s %s %s: %s", index, s.Method, s.Path, clip(deref(s.Body)), strings.Join(problems, "; "))
		}
	}
	t.Logf("%d/%d Flask answers reproduced", requests-failures, requests)
}

func compare(recorder *httptest.ResponseRecorder, s step) []string {
	problems := []string{}
	got := recorder.Result()
	if got.StatusCode != s.Status {
		problems = append(problems, "status "+http.StatusText(got.StatusCode)+", want "+http.StatusText(s.Status))
	}
	for key, want := range s.Headers {
		if have := got.Header.Get(key); have != want {
			problems = append(problems, key+": "+have+" != "+want)
		}
	}
	for key := range got.Header {
		if _, ok := s.Headers[key]; !ok && key != "Content-Length" {
			problems = append(problems, "extra header "+key)
		}
	}
	if body := recorder.Body.String(); body != s.Response {
		problems = append(problems, "body\n got: "+clip(body)+"\nwant: "+clip(s.Response))
	}
	return problems
}

func deref(text *string) string {
	if text == nil {
		return ""
	}
	return *text
}

func clip(text string) string {
	if len(text) > 400 {
		return text[:400] + "..."
	}
	return text
}
