package updates

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

type cannedRelease struct {
	Body     *string `json:"body"`
	ETag     *string `json:"etag"`
	HTTP     int     `json:"http"`
	URLError bool    `json:"urlerror"`
}

type updateScenario struct {
	Name      string          `json:"name"`
	ServerURL json.RawMessage `json:"server_url"`
	Steps     []struct {
		Op      string          `json:"op"`
		Now     *float64        `json:"now"`
		Arg     json.RawMessage `json:"arg"`
		Data    string          `json:"data"`
		Current string          `json:"current"`
		Answers []cannedRelease `json:"answers"`
	} `json:"steps"`
	Results []struct {
		HTTP     int               `json:"http"`
		Body     string            `json:"body"`
		Value    *bool             `json:"value"`
		Crash    string            `json:"crash"`
		File     *string           `json:"file"`
		Requests []recordedRelease `json:"requests"`
	} `json:"results"`
}

// cannedOpener answers like the Python test opener, without any network.
type cannedOpener struct {
	queue    []cannedRelease
	requests []*http.Request
}

func (o *cannedOpener) Do(request *http.Request) (*http.Response, error) {
	o.requests = append(o.requests, request)
	item := o.queue[0]
	o.queue = o.queue[1:]
	if item.URLError {
		return nil, errors.New("no route")
	}
	response := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}
	if item.HTTP != 0 {
		response.StatusCode = item.HTTP
		return response, nil
	}
	response.Body = io.NopCloser(strings.NewReader(*item.Body))
	if item.ETag != nil {
		response.Header.Set("ETag", *item.ETag)
	}
	return response, nil
}

func configureAnswer(service *Service, arg any) (int, string, error) {
	status, err := service.Configure(arg)
	var config *ConfigError
	if errors.As(err, &config) {
		body, _ := pyjson.Flask(config.Body())
		return 400, string(body), nil
	}
	if err != nil {
		return 0, "", err
	}
	body, _ := pyjson.Flask(status)
	return 200, string(body), nil
}

func TestScenariosMatchPython(t *testing.T) {
	for _, sc := range load(t).Scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			folder, clock := t.TempDir(), 0.0
			opener := &cannedOpener{}
			url := "https://license.scadabench.com"
			if sc.ServerURL != nil {
				url = "" // JSON null: the license config cannot be read
				json.Unmarshal(sc.ServerURL, &url)
			}
			serverURL := func() (string, error) {
				if url == "" {
					return "", errors.New("license config is not an object")
				}
				return url, nil
			}
			options := Options{Directory: folder, ServerURL: serverURL, Current: "0.1.13", Keys: testKeys(t),
				Opener: opener, Clock: func() float64 { return clock }}
			service := New(options)
			path := filepath.Join(folder, "update_check.json")
			for index, step := range sc.Steps {
				if step.Now != nil {
					clock = *step.Now
				}
				service.options.Current = "0.1.13"
				if step.Current != "" {
					service.options.Current = step.Current
				}
				if step.Op == "write" {
					os.WriteFile(path, []byte(step.Data), 0o666)
					continue
				}
				opener.queue, opener.requests = append([]cannedRelease(nil), step.Answers...), nil
				httpStatus, body, value, err := runUpdateStep(t, service, step.Op, step.Arg)
				want := sc.Results[index]
				switch {
				case want.Crash != "":
					if err == nil {
						t.Errorf("step %d %s: no error, Python raised %s", index, step.Op, want.Crash)
					}
				case err != nil:
					t.Errorf("step %d %s: %v", index, step.Op, err)
				case want.Value != nil && value != *want.Value:
					t.Errorf("step %d due = %v, want %v", index, value, *want.Value)
				case want.Value == nil && (httpStatus != want.HTTP || body != want.Body):
					t.Errorf("step %d %s:\n got %d %s\nwant %d %s", index, step.Op, httpStatus, body, want.HTTP, want.Body)
				}
				checkRequests(t, index, opener.requests, want.Requests)
				checkFile(t, index, path, want.File)
			}
		})
	}
}

func runUpdateStep(t *testing.T, service *Service, op string, arg json.RawMessage) (int, string, bool, error) {
	var status *pyjson.Object
	var err error
	switch op {
	case "due":
		due, err := service.Due()
		return 0, "", due, err
	case "status":
		status, err = service.Status()
	case "check", "manual":
		status, err = service.Check(op == "manual")
	case "configure":
		var value any
		if arg != nil {
			value = decode(t, string(arg))
		}
		code, body, err := configureAnswer(service, value)
		return code, body, false, err
	}
	if err != nil {
		return 0, "", false, err
	}
	body, _ := pyjson.Flask(status)
	return 200, string(body), false, nil
}

// recordedRelease is one request the Python opener saw.
type recordedRelease struct {
	URL     string            `json:"url"`
	Timeout int               `json:"timeout"`
	Headers map[string]string `json:"headers"`
}

func checkRequests(t *testing.T, index int, requests []*http.Request, want []recordedRelease) {
	got := []recordedRelease{}
	for _, request := range requests {
		headers := map[string]string{"user-agent": request.Header.Get("User-Agent"), "accept": request.Header.Get("Accept")}
		if tag := request.Header.Get("If-None-Match"); tag != "" {
			headers["if-none-match"] = tag
		}
		got = append(got, recordedRelease{request.URL.String(), TimeoutSeconds, headers})
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("step %d requests %v, want %v", index, got, want)
	}
}

func checkFile(t *testing.T, index int, path string, want *string) {
	data, err := os.ReadFile(path)
	if want == nil {
		if err == nil {
			t.Errorf("step %d: update_check.json exists, Python wrote none", index)
		}
		return
	}
	expected, _ := base64.StdEncoding.DecodeString(*want)
	if runtime.GOOS != "windows" {
		expected = []byte(strings.ReplaceAll(string(expected), "\r\n", "\n"))
	}
	if string(data) != string(expected) {
		t.Errorf("step %d: update_check.json\n got %q\nwant %q", index, data, expected)
	}
}
