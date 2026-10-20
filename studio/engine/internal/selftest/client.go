package selftest

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"studio-engine/internal/pyjson"
)

// client is Flask's app.test_client() for the composed handler: requests run in process
// as a loopback caller on host "localhost", and the session cookie is kept between them.
type client struct {
	handler http.Handler
	cookies map[string]string
	order   []string
}

// response is what the self-test reads from a test-client response.
type response struct {
	status int
	data   []byte
}

func newClient(handler http.Handler) *client {
	return &client{handler: handler, cookies: map[string]string{}}
}

// json is the parsed JSON body (get_json()), nil when the body is not JSON.
func (r response) json() any {
	decoded, err := pyjson.Decode(r.data)
	if err != nil {
		return nil
	}
	return decoded
}

func (c *client) get(path string) response { return c.send(http.MethodGet, path, nil, "") }

// sendJSON is client.<method>(path, json=payload).
func (c *client) sendJSON(method, path string, payload any) (response, error) {
	encoded, err := pyjson.Flask(payload)
	if err != nil {
		return response{}, err
	}
	return c.send(method, path, bytes.NewReader(encoded), "application/json"), nil
}

// upload is client.post(path, data={"file": (source, filename), field: value}).
func (c *client) upload(path, filename string, source []byte, field, value string) (response, error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", filename)
	if err != nil {
		return response{}, err
	}
	if _, err := part.Write(source); err != nil {
		return response{}, err
	}
	if err := form.WriteField(field, value); err != nil {
		return response{}, err
	}
	if err := form.Close(); err != nil {
		return response{}, err
	}
	return c.send(http.MethodPost, path, &body, form.FormDataContentType()), nil
}

func (c *client) send(method, path string, body io.Reader, contentType string) response {
	request := httptest.NewRequest(method, "http://localhost"+path, body)
	request.RemoteAddr = "127.0.0.1:50000"
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	for _, name := range c.order {
		if value, ok := c.cookies[name]; ok {
			request.AddCookie(&http.Cookie{Name: name, Value: value})
		}
	}
	recorder := httptest.NewRecorder()
	c.handler.ServeHTTP(recorder, request)
	result := recorder.Result()
	for _, cookie := range result.Cookies() {
		if cookie.MaxAge < 0 || cookie.Value == "" {
			delete(c.cookies, cookie.Name)
			continue
		}
		if _, known := c.cookies[cookie.Name]; !known {
			c.order = append(c.order, cookie.Name)
		}
		c.cookies[cookie.Name] = cookie.Value
	}
	return response{status: recorder.Code, data: recorder.Body.Bytes()}
}
