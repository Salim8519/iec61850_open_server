package projectapi

import (
	"io"
	"mime"
	"net/http"
	"strings"

	"studio-engine/internal/pyjson"
)

// form is Werkzeug's parsed multipart body: request.files and request.form. A part is a
// file when its Content-Disposition carries a filename parameter, even an empty one, and
// the filename is kept as sent (Werkzeug does not strip directories).
type form struct {
	files  map[string]filePart
	values map[string]string
}

type filePart struct {
	name string
	data []byte
}

// parseForm reads a multipart/form-data body; any other body yields an empty form, as
// request.files and request.form are empty then.
func parseForm(r *http.Request) (*form, error) {
	parsed := &form{files: map[string]filePart{}, values: map[string]string{}}
	mediatype, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediatype != "multipart/form-data" {
		return parsed, nil
	}
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, &httpError{400, badRequestText}
	}
	for {
		part, err := reader.NextRawPart()
		if err == io.EOF {
			return parsed, nil
		}
		if err != nil {
			return nil, &httpError{400, badRequestText}
		}
		_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if err != nil {
			continue
		}
		data, err := io.ReadAll(part)
		if err != nil {
			return nil, &httpError{400, badRequestText}
		}
		name := params["name"]
		filename, isFile := params["filename"]
		if isFile {
			if _, seen := parsed.files[name]; !seen { // MultiDict[key] is the first value
				parsed.files[name] = filePart{name: filename, data: data}
			}
		} else if _, seen := parsed.values[name]; !seen {
			parsed.values[name] = string(data)
		}
	}
}

// file is request.files[field]: Werkzeug's generic 400 when it is missing.
func (f *form) file(field string) ([]byte, string, error) {
	part, ok := f.files[field]
	if !ok {
		return nil, "", &httpError{400, badRequestText}
	}
	return part.data, part.name, nil
}

// upload is parseForm then file(field).
func upload(r *http.Request, field string) ([]byte, string, error) {
	parsed, err := parseForm(r)
	if err != nil {
		return nil, "", err
	}
	return parsed.file(field)
}

// isJSON is Werkzeug's request.is_json.
func isJSON(r *http.Request) bool {
	mediatype, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}
	return mediatype == "application/json" || strings.HasPrefix(mediatype, "application/") && strings.HasSuffix(mediatype, "+json")
}

// getJSON is Flask's request.get_json(silent): nil when silent and the body is not JSON;
// otherwise 415 for a non-JSON content type and Werkzeug's generic 400 for invalid JSON.
func getJSON(r *http.Request, silent bool) (any, error) {
	if !isJSON(r) {
		if silent {
			return nil, nil
		}
		return nil, &httpError{415, notJSONText}
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, &httpError{400, badRequestText}
	}
	text, err := jsonText(body)
	var value any
	if err == nil {
		value, err = pyjson.Decode(text)
	}
	if err != nil {
		if silent {
			return nil, nil
		}
		// Flask outside debug mode replaces Werkzeug's "Failed to decode JSON object: ..."
		// with a bare BadRequest.
		return nil, &httpError{400, badRequestText}
	}
	return value, nil
}

// object returns value as a dict, or nil.
func object(value any) *pyjson.Object {
	o, _ := value.(*pyjson.Object)
	return o
}
