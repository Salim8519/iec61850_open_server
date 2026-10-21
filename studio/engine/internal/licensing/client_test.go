package licensing

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"studio-engine/internal/pyjson"
)

type clientCase struct {
	outcome
	Call    string `json:"call"`
	URL     string `json:"url"`
	Answers []struct {
		Status   int    `json:"status"`
		Body     string `json:"body"`
		URLError bool   `json:"urlerror"`
	} `json:"answers"`
	Requests []struct {
		Method      string  `json:"method"`
		Path        string  `json:"path"`
		Body        *string `json:"body"`
		ContentType string  `json:"content_type"`
		UserAgent   string  `json:"user_agent"`
	} `json:"requests"`
}

type recordedRequest struct{ method, path, body, contentType, userAgent string }

// replayDoer answers from the case's canned list: "urlerror" fails like an unreachable
// server, every other answer comes from a real httptest server over loopback.
type replayDoer struct {
	server  *httptest.Server
	answers []bool // true = urlerror
	index   int
}

func (d *replayDoer) Do(request *http.Request) (*http.Response, error) {
	defer func() { d.index++ }()
	if d.index < len(d.answers) && d.answers[d.index] {
		return nil, errors.New("no route")
	}
	return d.server.Client().Do(request)
}

func TestHTTPClientMatchesPython(t *testing.T) {
	var recorded struct {
		HWID  string       `json:"hwid"`
		Key   string       `json:"key"`
		Cases []clientCase `json:"cases"`
	}
	readVectors(t, "client.json", &recorded)
	for index, c := range recorded.Cases {
		var mu sync.Mutex
		var requests []recordedRequest
		next := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			requests = append(requests, recordedRequest{r.Method, strings.TrimPrefix(r.URL.Path, "/"), string(body),
				r.Header.Get("Content-Type"), r.Header.Get("User-Agent")})
			answer := c.Answers[next]
			next++
			mu.Unlock()
			w.WriteHeader(answer.Status)
			io.WriteString(w, answer.Body)
		}))
		doer := &replayDoer{server: server}
		for _, answer := range c.Answers {
			doer.answers = append(doer.answers, answer.URLError)
			if answer.URLError {
				next++
			}
		}
		url := c.URL
		if strings.HasPrefix(url, "http://") {
			url = server.URL + strings.TrimPrefix(strings.TrimPrefix(url, "http://127.0.0.1:1"), "http://localhost:1")
		}
		client := NewHTTPClient(url, doer)
		var result any
		var err error
		switch c.Call {
		case "trial":
			result, err = client.StartTrial(recorded.HWID, nil, nil, nil)
		case "trial_parts":
			result, err = client.StartTrial(recorded.HWID, trialParts("g", "", "f"), nil, nil)
		case "trial_empty_parts":
			result, err = client.StartTrial(recorded.HWID, trialParts("", "", ""), nil, nil)
		case "trial_email":
			result, err = client.StartTrial(recorded.HWID, pyjson.NewObject(), "me@company.com", " 123456 ")
		case "trial_email_number":
			result, err = client.StartTrial(recorded.HWID, nil, "me@company.com", int64(123456))
		case "trial_email_nocode":
			result, err = client.StartTrial(recorded.HWID, nil, "me@company.com", nil)
		case "trial_no_email":
			result, err = client.StartTrial(recorded.HWID, nil, "", "123")
		case "email_code":
			result, err = client.RequestTrialCode(recorded.HWID, "me@company.com")
		case "activate":
			result, err = client.Activate(recorded.Key, recorded.HWID)
		case "transfer":
			result, err = client.Transfer(recorded.Key, recorded.HWID)
		case "verify":
			result, err = client.Verify(recorded.Key, recorded.HWID)
		case "verify_number_key":
			result, err = client.Verify(int64(5), recorded.HWID)
		}
		server.Close()
		checkClientOutcome(t, index, c, result, err)
		if len(requests) != countServed(c) {
			t.Errorf("case %d: %d requests reached the server, want %d", index, len(requests), countServed(c))
			continue
		}
		served := 0
		for _, want := range c.Requests {
			if want.Body == nil && want.Method == "GET" && served < len(requests) && requests[served].body != "" {
				t.Errorf("case %d: GET with a body", index)
			}
			if served >= len(requests) {
				break
			}
			got := requests[served]
			wantBody := ""
			if want.Body != nil {
				wantBody = *want.Body
			}
			if got.method != want.Method || got.path != want.Path || got.body != wantBody ||
				got.contentType != want.ContentType || got.userAgent != want.UserAgent {
				t.Errorf("case %d: request %+v, want %+v (body %q)", index, got, want, wantBody)
			}
			served++
		}
	}
}

func trialParts(guid, volume, firmware string) *pyjson.Object {
	return pyjson.NewObject().Set("guid", guid).Set("volume", volume).Set("firmware", firmware)
}

func countServed(c clientCase) int {
	served := 0
	for _, answer := range c.Answers {
		if !answer.URLError {
			served++
		}
	}
	return min(served, len(c.Requests))
}

func checkClientOutcome(t *testing.T, index int, c clientCase, result any, err error) {
	var server *ServerError
	var crash *UncaughtError
	switch {
	case c.OK != nil:
		var want string
		json.Unmarshal(c.OK, &want)
		if err != nil || compactText(t, result) != want {
			t.Errorf("case %d: got %v %v, want %s", index, result, err, want)
		}
	case c.Error == "LicenseServerError":
		wantCode, _ := json.Marshal(c.Code)
		if !errors.As(err, &server) || server.Message != c.Message || compactText(t, server.Code) != string(wantCode) {
			t.Errorf("case %d: got %v, want %q %s", index, err, c.Message, wantCode)
		}
	default:
		if !errors.As(err, &crash) {
			t.Errorf("case %d: got %v, want uncaught %s", index, err, c.Error)
		}
	}
}
