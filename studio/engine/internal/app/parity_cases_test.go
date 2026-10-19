package app

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// request is one HTTP exchange sent to both engines. In header values "{self}" is the
// engine's own base URL and "{port}" its port; auth is "", "bearer" or "cookie".
type request struct {
	name, method, path string
	headers            map[string]string
	body               string
	auth               string
	stream             bool
}

const jsonType = "application/json"

var parityRequests = []request{
	{name: "index", method: "GET", path: "/"},
	{name: "asset js", method: "GET", path: "/assets/app-1a2b.js"},
	{name: "asset css", method: "GET", path: "/assets/style-3c4d.css"},
	{name: "asset svg", method: "GET", path: "/assets/logo.svg"},
	{name: "asset binary", method: "GET", path: "/assets/data.bin"},
	{name: "asset missing", method: "GET", path: "/assets/missing.js"},
	{name: "asset traversal", method: "GET", path: "/assets/..%2findex.html"},
	{name: "asset folder", method: "GET", path: "/assets/"},
	{name: "index head", method: "HEAD", path: "/"},
	{name: "index post", method: "POST", path: "/"},
	{name: "foreign host", method: "GET", path: "/api/v1/project", headers: map[string]string{"Host": "example.com:{port}"}},
	{name: "foreign host static", method: "GET", path: "/", headers: map[string]string{"Host": "192.168.1.5"}},
	{name: "localhost host", method: "GET", path: "/api/v1/nope", headers: map[string]string{"Host": "localhost:{port}"}},
	{name: "no session", method: "GET", path: "/api/v1/project"},
	{name: "no session unknown", method: "GET", path: "/api/v1/nope"},
	{name: "bad bearer", method: "GET", path: "/api/v1/nope", headers: map[string]string{"Authorization": "Bearer nope"}},
	{name: "latin-1 bearer", method: "GET", path: "/api/v1/nope", headers: map[string]string{"Authorization": "Bearer \xe9"}},
	{name: "bearer unknown path", method: "GET", path: "/api/v1/nope", auth: "bearer"},
	{name: "cookie unknown path", method: "GET", path: "/api/v1/nope", auth: "cookie"},
	{name: "trailing slash", method: "GET", path: "/api/v1/project/", auth: "bearer"},
	{name: "wrong method", method: "DELETE", path: "/api/v1/project/test-panel", auth: "bearer"},
	{name: "options", method: "OPTIONS", path: "/api/v1/project/test-panel", auth: "bearer"},
	{name: "options unknown", method: "OPTIONS", path: "/api/v1/nope", auth: "bearer"},
	{name: "head unknown", method: "HEAD", path: "/api/v1/nope", auth: "bearer"},
	{name: "session get", method: "GET", path: "/api/v1/session"},
	{name: "session ok", method: "POST", path: "/api/v1/session", headers: map[string]string{"Content-Type": jsonType}, body: `{"token": "parity-token"}`},
	{name: "session wrong", method: "POST", path: "/api/v1/session", headers: map[string]string{"Content-Type": jsonType}, body: `{"token": "x"}`},
	{name: "session number", method: "POST", path: "/api/v1/session", headers: map[string]string{"Content-Type": jsonType}, body: `{"token": 5}`},
	{name: "session no type", method: "POST", path: "/api/v1/session", body: `{"token": "parity-token"}`},
	{name: "session malformed", method: "POST", path: "/api/v1/session", headers: map[string]string{"Content-Type": jsonType}, body: `{"token":`},
	{name: "session list", method: "POST", path: "/api/v1/session", headers: map[string]string{"Content-Type": jsonType}, body: `[1]`},
	{name: "session zero", method: "POST", path: "/api/v1/session", headers: map[string]string{"Content-Type": jsonType}, body: `0`},
	{name: "session non-ascii", method: "POST", path: "/api/v1/session", headers: map[string]string{"Content-Type": jsonType}, body: `{"token": "é"}`},
	{name: "session bom", method: "POST", path: "/api/v1/session", headers: map[string]string{"Content-Type": "application/json; charset=utf-8"}, body: "\xef\xbb\xbf{\"token\": \"parity-token\"}"},
	{name: "dev frontend ok", method: "GET", path: "/api/v1/nope", auth: "bearer", headers: map[string]string{"X-Studio-Frontend": "http://127.0.0.1:25173/"}},
	{name: "dev frontend old tab", method: "GET", path: "/api/v1/nope", auth: "bearer", headers: map[string]string{"X-Studio-Frontend": "http://127.0.0.1:5174"}},
	{name: "origin same", method: "GET", path: "/api/v1/nope", auth: "bearer", headers: map[string]string{"Origin": "{self}"}},
	{name: "origin foreign", method: "GET", path: "/api/v1/nope", auth: "bearer", headers: map[string]string{"Origin": "http://evil.example"}},
	{name: "origin null", method: "POST", path: "/api/v1/session", headers: map[string]string{"Origin": "null"}},
	{name: "origin bad ipv6", method: "GET", path: "/", headers: map[string]string{"Origin": "http://[::1"}},
	{name: "origin ipv4 brackets", method: "GET", path: "/", headers: map[string]string{"Origin": "http://[127.0.0.1]"}},
	{name: "license path", method: "GET", path: "/api/v1/licenseX", auth: "bearer"},
	{name: "preferences prefix", method: "GET", path: "/api/v1/preferencesX", auth: "bearer"},
	{name: "events bad cursor", method: "GET", path: "/api/v1/events?after=abc", auth: "bearer"},
	{name: "events long cursor", method: "GET", path: "/api/v1/events", auth: "bearer", headers: map[string]string{"Last-Event-ID": strings.Repeat("x", 300)}},
	{name: "events empty header", method: "GET", path: "/api/v1/events?after=5", auth: "bearer", headers: map[string]string{"Last-Event-ID": ""}},
	{name: "events stream", method: "GET", path: "/api/v1/events", auth: "bearer", stream: true},
	{name: "events stream cursor", method: "GET", path: "/api/v1/events?after=%2B1_0", auth: "bearer", stream: true},
	{name: "events post", method: "POST", path: "/api/v1/events", auth: "bearer"},
}

// exchange sends one request to base and returns the response description.
func exchange(t *testing.T, base string, item request, cookie string) string {
	port := base[strings.LastIndexByte(base, ':')+1:]
	replace := strings.NewReplacer("{self}", base, "{port}", port)
	req, err := http.NewRequest(item.method, base+item.path, strings.NewReader(item.body))
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range item.headers {
		if name == "Host" {
			req.Host = replace.Replace(value)
			continue
		}
		req.Header[name] = []string{replace.Replace(value)}
	}
	switch item.auth {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+parityToken)
	case "cookie":
		req.Header.Set("Cookie", cookie)
	}
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := client.Do(req)
	if err != nil {
		return "transport error: " + err.Error()
	}
	defer response.Body.Close()
	var body []byte
	if item.stream {
		body = firstEvent(response.Body)
	} else {
		body, _ = io.ReadAll(response.Body)
	}
	return describe(response, body)
}

// firstEvent reads an SSE body up to its first blank line.
func firstEvent(body io.Reader) []byte {
	done := make(chan []byte, 1)
	go func() {
		reader := bufio.NewReader(body)
		var out []byte
		for !strings.HasSuffix(string(out), "\n\n") {
			line, err := reader.ReadBytes('\n')
			out = append(out, line...)
			if err != nil {
				break
			}
		}
		done <- out
	}()
	select {
	case out := <-done:
		return out
	case <-time.After(5 * time.Second):
		return []byte("<no event within 5 s>")
	}
}

// sessionCookie signs in through POST /api/v1/session and returns "studio_session=...".
func sessionCookie(t *testing.T, base string) string {
	response, err := http.Post(base+"/api/v1/session", jsonType, strings.NewReader(`{"token": "parity-token"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	for _, cookie := range response.Cookies() {
		if cookie.Name == "studio_session" {
			return cookie.Name + "=" + cookie.Value
		}
	}
	t.Fatalf("%s gave no session cookie (%d)", base, response.StatusCode)
	return ""
}
