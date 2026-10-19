package projectapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// goldenCase is one request/answer pair recorded from the Flask app by http_gen.py.
type goldenCase struct {
	Name        string            `json:"name"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Body        *string           `json:"body"`
	ContentType string            `json:"content_type"`
	Form        map[string]string `json:"form"`
	File        *struct {
		Field    string `json:"field"`
		Filename string `json:"filename"`
		B64      string `json:"b64"`
	} `json:"file"`
	Status   int               `json:"status"`
	Headers  map[string]string `json:"headers"`
	Binary   bool              `json:"binary"`
	Response string            `json:"response"`
}

var recoveryFile = regexp.MustCompile(`"recovery_file":"[^"]*"`)

func normalise(text string) string {
	return recoveryFile.ReplaceAllString(text, `"recovery_file":"*"`)
}

func (c goldenCase) request(t *testing.T) *http.Request {
	var body bytes.Buffer
	contentType := c.ContentType
	switch {
	case c.Form != nil || c.File != nil:
		writer := multipart.NewWriter(&body)
		names := make([]string, 0, len(c.Form))
		for name := range c.Form {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			_ = writer.WriteField(name, c.Form[name])
		}
		if c.File != nil {
			data, err := base64.StdEncoding.DecodeString(c.File.B64)
			if err != nil {
				t.Fatal(err)
			}
			part, _ := writer.CreateFormFile(c.File.Field, c.File.Filename)
			_, _ = part.Write(data)
		}
		_ = writer.Close()
		contentType = writer.FormDataContentType()
	case c.Body != nil:
		body.WriteString(*c.Body)
	}
	request := httptest.NewRequest(c.Method, "http://localhost"+c.Path, &body)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	return request
}

// TestFlaskParity replays every recorded request in order against one Go area and
// compares status, headers (except Content-Length) and the exact body.
func TestFlaskParity(t *testing.T) {
	data, err := os.ReadFile("testdata/generated/http_golden.json")
	if err != nil {
		t.Skip("http golden missing")
	}
	var golden struct {
		Cases []goldenCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatal(err)
	}
	handler, runtime := newArea(t)
	failures := 0
	for _, c := range golden.Cases {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, c.request(t))
		got := recorder.Result()
		problems := []string{}
		if got.StatusCode != c.Status {
			problems = append(problems, "status "+http.StatusText(got.StatusCode))
		}
		for key, want := range c.Headers {
			if have := got.Header.Get(key); have != want {
				problems = append(problems, key+": "+have+" != "+want)
			}
		}
		for key := range got.Header {
			if _, ok := c.Headers[key]; !ok && key != "Content-Length" {
				problems = append(problems, "extra header "+key)
			}
		}
		body := recorder.Body.String()
		if !c.Binary && normalise(body) != normalise(c.Response) {
			problems = append(problems, "body\n got: "+clip(body)+"\nwant: "+clip(c.Response))
		}
		if len(problems) > 0 {
			failures++
			t.Errorf("%s (%s %s): %s", c.Name, c.Method, c.Path, strings.Join(problems, "; "))
		}
	}
	t.Logf("%d/%d Flask answers reproduced; events %v", len(golden.Cases)-failures, len(golden.Cases), runtime.events)
}

func clip(text string) string {
	if len(text) > 600 {
		return text[:600] + "..."
	}
	return text
}
