//go:build parity

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/recovery"
	"studio-engine/internal/updates"
)

// The callback-backed fakes of the seams file (tests/parity/server_seams.py): an object
// seam standing in for the license service, the update opener and the fault points the Go
// engine has a seam for.

// recoverFault is the fault target the Go engine can wrap (create_app's background
// recovery calls it with the project service, which crosses as a handle).
const recoverFault = "studio_app.projects.background_recovery.recover_startup"

// copyFault is shutil.copyfile inside startup_recovery: the copy of an unreadable autosave
// (internal/store/recovery's parity copy seam). The fake gets (source, backup) as paths.
const copyFault = "studio_app.projects.startup_recovery.shutil.copyfile"

// faultSeams are the fault targets the Go parity build can carry.
var faultSeams = map[string]bool{recoverFault: true, copyFault: true}

// objectSeam is {"object": {"attrs": {...}, "methods": {name: <callback>}}}.
type objectSeam struct {
	Attrs   map[string]json.RawMessage `json:"attrs"`
	Methods map[string]callback        `json:"methods"`
}

// objectLicense is the license gate of a duck-typed license service: enforce from its
// attributes, is_licensed() through its callback on every request (the gate's call).
type objectLicense struct {
	enforce    bool
	isLicensed callback
}

func newObjectLicense(raw json.RawMessage) (*objectLicense, error) {
	var seam objectSeam
	if err := json.Unmarshal(raw, &seam); err != nil {
		return nil, err
	}
	gate := &objectLicense{isLicensed: seam.Methods["is_licensed"]}
	if value, found := seam.Attrs["enforce"]; found {
		_ = json.Unmarshal(value, &gate.enforce)
	}
	if !gate.isLicensed.valid() {
		return nil, errors.New("the license object seam has no is_licensed method")
	}
	return gate, nil
}

func (g *objectLicense) Enforced() bool { return g.enforce }

func (g *objectLicense) Licensed() bool {
	value, err := g.isLicensed.call()
	licensed, _ := value.(bool)
	return err == nil && licensed
}

// callbackOpener is remote_opener: the request is handed to the test's canned urllib opener
// as {"url", "method", "headers", "data"} and its answer {"body", "headers"} becomes the
// response; an HTTPError it raised becomes that status, anything else a network failure.
type callbackOpener struct{ fake callback }

func (o callbackOpener) Do(request *http.Request) (*http.Response, error) {
	headers := map[string]string{}
	for key, values := range request.Header {
		headers[urllibHeader(key)] = values[0]
	}
	record := pyjson.NewObject().Set("url", request.URL.String()).Set("method", request.Method).
		Set("headers", wireEncode(headers)).Set("data", nil)
	value, err := o.fake.call(record, float64(updates.TimeoutSeconds))
	var raised *remoteError
	switch {
	case errors.As(err, &raised) && raised.HTTPCode != 0:
		return response(int(raised.HTTPCode), nil, nil), nil
	case err != nil:
		return nil, err
	}
	body, _ := field(value, "body").(wireBytes)
	answer := http.Header{}
	if object, ok := field(value, "headers").(*pyjson.Object); ok {
		keys := object.Keys()
		sort.Strings(keys)
		for _, key := range keys {
			item, _ := object.Get(key)
			answer.Set(key, fmt.Sprint(item))
		}
	}
	return response(http.StatusOK, answer, body), nil
}

func response(status int, header http.Header, body []byte) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(bytes.NewReader(body))}
}

// urllibHeader is urllib.request.Request's header key form ("User-agent").
func urllibHeader(key string) string {
	if key == "" {
		return key
	}
	lower := []byte(http.CanonicalHeaderKey(key))
	for i := 1; i < len(lower); i++ {
		if lower[i] >= 'A' && lower[i] <= 'Z' {
			lower[i] += 'a' - 'A'
		}
	}
	return string(lower)
}

// recoverWithFault runs the fake standing in for recover_startup(projects): it answers
// "original" when it touches the project service (the engine then recovers itself), a
// returned status object, or the exception it raised.
func recoverWithFault(fake callback) func(func() (*pyjson.Object, error)) (*pyjson.Object, error) {
	return func(original func() (*pyjson.Object, error)) (*pyjson.Object, error) {
		value, err := fake.call(wireHandle("ProjectService"))
		if errors.Is(err, errOriginal) {
			return original()
		}
		if err != nil {
			return nil, err
		}
		status, _ := value.(*pyjson.Object)
		return status, nil
	}
}

// copyWithFault installs the fake standing in for shutil.copyfile(source, backup): an
// exception it raises fails the copy (Python's OSError), "original" copies for real.
func copyWithFault(fake callback) {
	recovery.SetCopySeam(func(source, destination string, original func(string, string) error) error {
		_, err := fake.call(wirePath(source), wirePath(destination))
		if errors.Is(err, errOriginal) {
			return original(source, destination)
		}
		return err
	})
}
