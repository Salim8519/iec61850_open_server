package licensing

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// Doer sends one HTTP request; *http.Client satisfies it and tests pass httptest clients.
type Doer interface {
	Do(request *http.Request) (*http.Response, error)
}

// HTTPClient is client.LicenseClient over net/http. Go's TLS on Windows verifies the
// server with the system certificate verifier, as truststore does for Python (ADR-0056).
type HTTPClient struct {
	ServerURL string
	Doer      Doer
}

// NewHTTPClient returns a client for serverURL (a trailing "/" is dropped).
func NewHTTPClient(serverURL string, doer Doer) *HTTPClient {
	return &HTTPClient{ServerURL: strings.TrimRight(serverURL, "/"), Doer: doer}
}

// DefaultClientFactory is LicenseClient(server_url) with the 20 s timeout.
func DefaultClientFactory(serverURL string) Client {
	return NewHTTPClient(serverURL, &http.Client{Timeout: ClientTimeoutSeconds * time.Second})
}

func offline() error { return &ServerError{Message: OfflineText, Code: "offline"} }

// call is LicenseClient._call: JSON in, decoded JSON out, readable ServerErrors.
func (c *HTTPClient) call(method, path string, body *pyjson.Object) (any, error) {
	if !allowedServer(c.ServerURL) {
		return nil, &ServerError{Message: NoServerText, Code: "no_server"}
	}
	var reader io.Reader
	if body != nil {
		data, err := pyjson.Dumps(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequest(method, c.ServerURL+path, reader)
	if err != nil {
		return nil, offline()
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "ScadaBench")
	request.Header.Set("Accept-Encoding", "identity")
	response, err := c.Doer.Do(request)
	if err != nil {
		return nil, offline()
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, httpFailure(response)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxAnswerBytes))
	if err != nil {
		return nil, offline()
	}
	value, err := pytext.LoadsBytes(data)
	if err != nil {
		return nil, offline()
	}
	return value, nil
}

// httpFailure is the HTTPError branch: the server's {"error", "code"} when readable.
func httpFailure(response *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBytes))
	detail, err := pytext.LoadsBytes(data)
	object, isObject := detail.(*pyjson.Object)
	if err == nil && !isObject {
		return uncaught("license server error detail is not an object")
	}
	message := fmt.Sprintf("License server error %d", response.StatusCode)
	if text, found := lookup(object, "error"); found && truthy(text) {
		if scalar, ok := scalarText(text); ok {
			message = scalar
		}
	}
	code, found := lookup(object, "code")
	if !found {
		code = defaultErrCode
	}
	return &ServerError{Message: message, Code: code}
}

// proof is LicenseClient._proof: fetch a challenge and solve its puzzle.
func (c *HTTPClient) proof() (*pyjson.Object, error) {
	answer, err := c.call("GET", "/api/v1/challenge", nil)
	if err != nil {
		return nil, err
	}
	challenge, _ := answer.(*pyjson.Object)
	nonce, hasNonce := lookup(challenge, "nonce")
	rawDifficulty, hasDifficulty := lookup(challenge, "difficulty")
	if !hasNonce || !hasDifficulty {
		return nil, uncaught("license server challenge is incomplete")
	}
	difficulty, err := pyInt(rawDifficulty)
	nonceText, isText := scalarText(nonce)
	if err != nil || !isText || !isASCII(nonceText) {
		return nil, uncaught("license server challenge is unreadable")
	}
	counter, err := Solve(nonceText, difficulty)
	if err != nil {
		return nil, err
	}
	timestamp, hasTimestamp := lookup(challenge, "timestamp")
	mac, hasMac := lookup(challenge, "mac")
	if !hasTimestamp || !hasMac {
		return nil, uncaught("license server challenge is incomplete")
	}
	return pyjson.NewObject().Set("pow_nonce", nonce).Set("pow_ts", timestamp).
		Set("pow_difficulty", rawDifficulty).Set("pow_mac", mac).Set("pow_counter", counter), nil
}

// certificateRequest posts body plus a solved proof and returns answer["certificate"].
func (c *HTTPClient) certificateRequest(path string, body *pyjson.Object) (any, error) {
	proof, err := c.proof()
	if err != nil {
		return nil, err
	}
	for _, key := range proof.Keys() {
		value, _ := proof.Get(key)
		body.Set(key, value)
	}
	answer, err := c.call("POST", path, body)
	if err != nil {
		return nil, err
	}
	certificate, found := lookup(answerObject(answer), "certificate")
	if !found {
		return nil, uncaught("'certificate'")
	}
	return certificate, nil
}

func isASCII(text string) bool {
	for index := 0; index < len(text); index++ {
		if text[index] >= 0x80 {
			return false
		}
	}
	return true
}

func answerObject(answer any) *pyjson.Object {
	object, _ := answer.(*pyjson.Object)
	return object
}

// StartTrial asks for a trial certificate bound to hwid.
func (c *HTTPClient) StartTrial(hwid string) (any, error) {
	return c.certificateRequest("/api/v1/trial", pyjson.NewObject().Set("hwid", hwid))
}

// Activate binds licenseKey to hwid.
func (c *HTTPClient) Activate(licenseKey, hwid string) (any, error) {
	body := pyjson.NewObject().Set("license_key", licenseKey).Set("hwid", hwid)
	return c.certificateRequest("/api/v1/activate", body)
}

// Transfer moves licenseKey to hwid.
func (c *HTTPClient) Transfer(licenseKey, hwid string) (any, error) {
	body := pyjson.NewObject().Set("license_key", licenseKey).Set("hwid", hwid)
	return c.certificateRequest("/api/v1/transfer", body)
}

// Verify asks whether the license is still valid on hwid.
func (c *HTTPClient) Verify(licenseKey any, hwid string) (any, error) {
	return c.call("POST", "/api/v1/verify", pyjson.NewObject().Set("license_key", licenseKey).Set("hwid", hwid))
}
