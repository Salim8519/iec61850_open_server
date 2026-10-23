package projectapi

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func multipartBody(t *testing.T, content []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range fields {
		_ = writer.WriteField(key, value)
	}
	part, err := writer.CreateFormFile("file", "big.scd")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(content)
	_ = writer.Close()
	return body, writer.FormDataContentType()
}

// countingBody records how much of a request body the handler read.
type countingBody struct {
	reader io.Reader
	read   int
}

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.read += n
	return n, err
}

// A large upload refused while running gets the JSON answer, not a TCP reset (change 0186):
// the refused body is read to the end before the answer. Go's server reads at most 256 KiB of
// an unread body and then closes the connection, so a client still sending gets a reset.
func TestRefusedLargeUploadIsReadToTheEnd(t *testing.T) {
	handler, runtime := newArea(t)
	runtime.state = "running"
	content := bytes.Repeat([]byte("<!-- padding padding padding padding -->\n"), 3*1024*1024/41)
	for _, path := range []string{"/api/v1/scl/import", "/api/v1/ieds/template"} {
		body, contentType := multipartBody(t, content, map[string]string{"selected_names": `["A"]`})
		size := body.Len()
		counting := &countingBody{reader: body}
		request := httptest.NewRequest("POST", path, counting)
		request.Header.Set("Content-Type", contentType)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != 400 || !strings.Contains(recorder.Body.String(), "Stop the simulation before editing its structure") {
			t.Fatalf("%s: %d %s", path, recorder.Code, recorder.Body.String())
		}
		if counting.read != size {
			t.Fatalf("%s: read %d of %d body bytes before answering", path, counting.read, size)
		}
	}
}

// Preview imports nothing (API.md), so it is allowed while running (change 0186).
func TestPreviewWorksWhileRunning(t *testing.T) {
	handler, runtime := newArea(t)
	runtime.state = "running"
	body, contentType := multipartBody(t, []byte("<SCL"), nil)
	request := httptest.NewRequest("POST", "/api/v1/scl/preview", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if text := recorder.Body.String(); strings.Contains(text, "Stop the simulation") || !strings.Contains(text, "Invalid or unsafe XML") {
		t.Fatalf("%d %s", recorder.Code, text)
	}
}
