package platformapi

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/licensing"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

// fakeLicenseClient is gen_fakes.LicenseClient: canned answers per call name.
type fakeLicenseClient struct{ answers map[string][]any }

func (c *fakeLicenseClient) answer(name string) (any, error) {
	queue := c.answers[name]
	if len(queue) == 0 {
		return nil, &licensing.ServerError{Message: "no canned answer", Code: "offline"}
	}
	c.answers[name] = queue[1:]
	item := queue[0].(*pyjson.Object)
	if raised, found := item.Get("raise"); found {
		parts := raised.([]any)
		return nil, &licensing.ServerError{Message: parts[0].(string), Code: parts[1]}
	}
	value, _ := item.Get("return")
	return value, nil
}

func (c *fakeLicenseClient) StartTrial(string, *pyjson.Object, any, any) (any, error) {
	return c.answer("trial")
}
func (c *fakeLicenseClient) RequestTrialCode(string, string) (any, error) {
	return c.answer("email-code")
}
func (c *fakeLicenseClient) Activate(string, string) (any, error)    { return c.answer("activate") }
func (c *fakeLicenseClient) Verify(any, string, string) (any, error) { return c.answer("verify") }

// fakeServer is gen_fakes.UpdateServer: the update check's only HTTP call.
type fakeServer struct{ queue []any }

func (s *fakeServer) Do(request *http.Request) (*http.Response, error) {
	if len(s.queue) == 0 {
		return nil, errors.New("no route")
	}
	item := s.queue[0].(*pyjson.Object)
	s.queue = s.queue[1:]
	if failed, _ := item.Get("urlerror"); failed == true {
		return nil, errors.New("no route")
	}
	response := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(nil))}
	if code, found := item.Get("http"); found {
		response.StatusCode = int(code.(int64))
		return response, nil
	}
	body, _ := item.Get("body")
	response.Body = io.NopCloser(strings.NewReader(body.(string)))
	if etag, _ := item.Get("etag"); etag != nil {
		response.Header.Set("ETag", etag.(string))
	}
	return response, nil
}

// fakeWindows is gen_fakes.Windows: the state "net" steps set; "!Class:message" raises.
type fakeWindows struct {
	t   testing.TB
	env *pyjson.Object
}

func (w *fakeWindows) read(key string) (any, error) {
	value, _ := w.env.Get(key)
	if text, ok := value.(string); ok && strings.HasPrefix(text, "!") {
		kind, message, _ := strings.Cut(text[1:], ":")
		switch kind {
		case "RuntimeError":
			return nil, apperr.RuntimeError(message)
		case "ValueError":
			return nil, apperr.ValueError(message)
		case "KeyError":
			return nil, apperr.KeyError(message)
		}
		w.t.Fatalf("unknown fake error %q", text)
	}
	return value, nil
}

func (w *fakeWindows) adapters() ([]netcfg.Adapter, error) {
	value, err := w.read("adapters")
	if err != nil {
		return nil, err
	}
	found := []netcfg.Adapter{}
	for _, item := range value.([]any) {
		found = append(found, adapterFrom(item.(*pyjson.Object)))
	}
	return found, nil
}

func (w *fakeWindows) names(key string) func() ([]string, error) {
	return func() ([]string, error) {
		value, err := w.read(key)
		if err != nil {
			return nil, err
		}
		return texts(value), nil
	}
}

func (w *fakeWindows) blocking(string) ([]netcfg.BlockingRule, error) {
	value, err := w.read("blocking")
	if err != nil {
		return nil, err
	}
	rules := []netcfg.BlockingRule{}
	for _, item := range value.([]any) {
		rule := item.(*pyjson.Object)
		name, _ := rule.Get("name")
		display, _ := rule.Get("display")
		rules = append(rules, netcfg.BlockingRule{Name: name.(string), Display: display.(string)})
	}
	return rules, nil
}

func (w *fakeWindows) probe(addresses []string) (map[string]bool, error) {
	value, _ := w.read("probe")
	answering := set(texts(value))
	found := map[string]bool{}
	for _, address := range addresses {
		if answering[address] {
			found[address] = true
		}
	}
	return found, nil
}

func (w *fakeWindows) firewall([]string) (map[string]bool, map[string]bool, error) {
	value, _ := w.read("state")
	state := value.(*pyjson.Object)
	gateway, _ := state.Get("gateway")
	disabled, _ := state.Get("disabled")
	return set(texts(gateway)), set(texts(disabled)), nil
}

// elevate stands in for the UAC helper: it never starts a process; it writes the result
// file the current mode asks for next to the request (arguments: flag, request, digest,
// program).
func (w *fakeWindows) elevate(executable string, arguments []string) (uint32, error) {
	mode, _ := w.read("elevate")
	switch mode {
	case "cancel":
		return 0, netcfg.ErrElevationCancelled
	case "runtime":
		return 0, apperr.RuntimeError("The network helper did not finish within two minutes")
	case "none":
		return 0, nil
	}
	request := arguments[1]
	payload, err := os.ReadFile(request)
	if err != nil {
		w.t.Fatal(err)
	}
	decoded, _ := pyjson.Decode(payload)
	items, _ := decoded.(*pyjson.Object).Get("items")
	results := []any{}
	for n, item := range items.([]any) {
		key, _ := netcfg.ItemKey(item.(*pyjson.Object))
		failed := mode == "fail" && n == 1
		message := ""
		if failed {
			message = "Access denied"
		}
		results = append(results, pyjson.NewObject().Set("key", key).Set("ok", !failed).Set("message", message))
	}
	outcome := pyjson.NewObject().Set("refused", false).Set("results", results)
	if mode == "refused" {
		outcome = pyjson.NewObject().Set("refused", true).Set("reason", "Unknown helper action").Set("results", []any{})
	}
	data, _ := pyjson.Dumps(outcome)
	resultPath := strings.TrimSuffix(request, filepath.Ext(request)) + ".result.json"
	return 0, os.WriteFile(resultPath, data, 0o666)
}

// fakeRuntime is the coordinator's state and cycle_lock.
type fakeRuntime struct {
	state string
	lock  sync.Mutex
}

func (r *fakeRuntime) State() string          { return r.state }
func (r *fakeRuntime) CycleLock() sync.Locker { return &r.lock }

// licenseProxy lets "license" steps replace the service behind the routes.
type licenseProxy struct{ *licensing.Service }

func adapterFrom(object *pyjson.Object) netcfg.Adapter {
	field := func(key string) any { value, _ := object.Get(key); return value }
	adapter := netcfg.Adapter{GUID: field("guid").(string), Name: field("name").(string),
		Description: field("description").(string), Status: field("status").(string), MAC: field("mac").(string),
		Speed: field("speed").(string), Kind: field("kind").(string), DefaultRoute: field("default_route").(bool)}
	if index, ok := field("index").(int64); ok {
		adapter.Index = &index
	}
	for _, item := range field("addresses").([]any) {
		address := item.(*pyjson.Object)
		ip, _ := address.Get("ip")
		prefix, _ := address.Get("prefix")
		skip, _ := address.Get("skip_as_source")
		origin, _ := address.Get("origin")
		adapter.Addresses = append(adapter.Addresses, netcfg.Address{IP: ip.(string), Prefix: int(prefix.(int64)),
			SkipAsSource: skip.(bool), Origin: origin.(string)})
	}
	return adapter
}

func texts(value any) []string {
	found := []string{}
	list, _ := value.([]any)
	for _, item := range list {
		found = append(found, item.(string))
	}
	return found
}

func set(items []string) map[string]bool {
	found := map[string]bool{}
	for _, item := range items {
		found[item] = true
	}
	return found
}
