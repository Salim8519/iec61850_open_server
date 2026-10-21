package netcfg

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// outcome is vector_common.outcome: Python's exact JSON text, or the exception it raised.
type outcome struct {
	OK      *string         `json:"ok"`
	Error   string          `json:"error"`
	Message string          `json:"message"`
	Code    any             `json:"code"`
	Details json.RawMessage `json:"details"`
}

// loadVectors reads testdata/generated/<name> into target.
func loadVectors(t *testing.T, name string, target any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "generated", name))
	if err != nil {
		t.Fatalf("vectors missing (run testdata/make_*.py): %v", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

// py decodes a vector value with Python's json semantics.
func py(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	if len(raw) == 0 {
		return nil
	}
	value, err := pyjson.Decode(raw)
	if err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return value
}

func pyObject(t *testing.T, raw json.RawMessage) *pyjson.Object {
	t.Helper()
	object, _ := py(t, raw).(*pyjson.Object)
	return object
}

func dumpsText(t *testing.T, value any) string {
	t.Helper()
	data, err := pyjson.Dumps(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// pythonClass names the Python exception an error stands for.
func pythonClass(err error) string {
	var known *apperr.Error
	if errors.As(err, &known) {
		return map[apperr.Kind]string{apperr.Studio: "StudioError", apperr.Value: "ValueError",
			apperr.Key: "KeyError", apperr.Runtime: "RuntimeError"}[known.Kind]
	}
	var refused *HelperRefused
	if errors.As(err, &refused) {
		return "HelperRefused"
	}
	return "error"
}

// checkOutcome compares a Go result (text) or error with Python's. Exception texts must be
// equal for StudioError, KeyError and HelperRefused, and for ValueError when exact is set;
// Python's TypeError/AttributeError only require an error.
func checkOutcome(t *testing.T, label, got string, err error, want outcome, exact bool) {
	t.Helper()
	if want.OK != nil {
		if err != nil {
			t.Errorf("%s: unexpected error %v", label, err)
		} else if got != *want.OK {
			t.Errorf("%s:\n got %s\nwant %s", label, got, *want.OK)
		}
		return
	}
	if err == nil {
		t.Errorf("%s: want %s %q, got %s", label, want.Error, want.Message, got)
		return
	}
	switch want.Error {
	case "TypeError", "AttributeError", "UnicodeDecodeError", "JSONDecodeError":
		return
	}
	class := pythonClass(err)
	if want.Error == "AddressValueError" || want.Error == "NetmaskValueError" {
		want.Error, exact = "ValueError", false // ipaddress's own subclasses and texts
	}
	if class != want.Error {
		t.Errorf("%s: class %s (%v), want %s %q", label, class, err, want.Error, want.Message)
		return
	}
	if (exact || class != "ValueError") && err.Error() != want.Message {
		t.Errorf("%s: message\n got %q\nwant %q", label, err.Error(), want.Message)
	}
}

// adapterFrom turns a normalized adapter dictionary (the Python fakes) into an Adapter.
func adapterFrom(object *pyjson.Object) Adapter {
	adapter := Adapter{GUID: field(object, "guid", ""), Name: field(object, "name", ""),
		Description: field(object, "description", ""), Status: field(object, "status", ""),
		MAC: field(object, "mac", ""), Speed: field(object, "speed", ""), Kind: field(object, "kind", ""),
		DefaultRoute: truthy(get(object, "default_route"))}
	if index, ok := intField(object, "index"); ok {
		adapter.Index = &index
	}
	for _, item := range objects(get(object, "addresses")) {
		prefix, _ := intField(item, "prefix")
		adapter.Addresses = append(adapter.Addresses, Address{IP: field(item, "ip", ""), Prefix: int(prefix),
			SkipAsSource: truthy(get(item, "skip_as_source")), Origin: field(item, "origin", "")})
	}
	return adapter
}

func adaptersFrom(value any) []Adapter {
	found := []Adapter{}
	for _, object := range objects(value) {
		found = append(found, adapterFrom(object))
	}
	return found
}

// iedFrom turns {"id", "name", "scl_address", "port"} into a PlanIED.
func iedFrom(object *pyjson.Object) PlanIED {
	ied := PlanIED{ID: field(object, "id", ""), Name: field(object, "name", ""), Port: 102}
	if port, ok := intField(object, "port"); ok {
		ied.Port = int(port)
	}
	if address, ok := get(object, "scl_address").(*pyjson.Object); ok {
		if ip := get(address, "ip"); truthy(ip) {
			ied.SCLIP = pyStr(ip)
		}
		if subnet := get(address, "subnet"); truthy(subnet) {
			ied.SCLSubnet = pyStr(subnet)
		}
	}
	return ied
}

func stringList(value any) []string {
	found := []string{}
	list, _ := value.([]any)
	for _, item := range list {
		found = append(found, pyStr(item))
	}
	return found
}
