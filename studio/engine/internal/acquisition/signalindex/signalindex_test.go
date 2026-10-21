package signalindex

import (
	"errors"
	"sort"
	"strconv"
	"sync"
	"testing"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// answer renders a Search result the way the Flask route and its error handlers do.
func answer(t *testing.T, result *pyjson.Object, err error) (string, string) {
	t.Helper()
	if err == nil {
		body, encodeErr := pyjson.Flask(result)
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		return "200 " + string(body), ""
	}
	var failure *apperr.Error
	if !errors.As(err, &failure) {
		t.Fatalf("not an apperr: %v", err)
	}
	switch failure.Kind {
	case apperr.Studio:
		details := failure.Details
		if details == nil {
			details = []any{}
		}
		body, _ := pyjson.Flask(pyv.O("error", failure.Message, "code", failure.Code, "details", details))
		return "400 " + string(body), "StudioError"
	case apperr.Value:
		body, _ := pyjson.Flask(pyv.O("error", failure.Message))
		return "400 " + string(body), "ValueError"
	}
	t.Fatalf("unexpected kind %v", failure.Kind)
	return "", ""
}

func TestSignalsMatchPythonRoute(t *testing.T) {
	indexes := map[string]*Index{}
	queries := 0
	for _, c := range goldentest.Load(t, "signals_http.json") {
		switch pyv.Str(c.Args[0]) {
		case "ied":
			indexes[pyv.Str(c.Args[1])] = Build(c.Obj(2))
		case "query":
			ix := indexes[pyv.Str(c.Args[1])]
			if ix == nil {
				t.Fatalf("%s: IED not recorded before its queries", c.Name)
			}
			result, err := ix.Search(ParseQuery(ParseArgs(pyv.Str(c.Args[2]))), nil)
			text, class := answer(t, result, err)
			goldentest.Text(t, c, text)
			if want := c.Args[3]; (want == nil && class != "") || (want != nil && pyv.Str(want) != class) {
				t.Errorf("%s: raised %q, Python raised %v", c.Name, class, want)
			}
			queries++
		case "args":
			values := ParseArgs(pyv.Str(c.Args[1]))
			keys := make([]string, 0, len(values))
			for key := range values {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			pairs := []any{}
			for _, key := range keys {
				pairs = append(pairs, []any{key, pyv.Strings(values[key])})
			}
			body, _ := pyjson.Dumps(pairs)
			goldentest.Text(t, c, string(body))
		}
	}
	if queries < 500 {
		t.Fatalf("only %d recorded queries", queries)
	}
}

func TestValueReaderOverridesStoredValues(t *testing.T) {
	ied := pyv.O("id", "x", "signals", []any{
		pyv.O("ref", "LD/A.q", "fc", "ST", "kind", "Quality", "value", int64(0), "description", "",
			"quality_ref", "LD/A.q", "timestamp_ref", "LD/A.t"),
		pyv.O("ref", "LD/A.t", "fc", "ST", "kind", "Timestamp", "value", int64(5), "description", ""),
	}, "datasets", []any{}, "reports", []any{})
	ix := Build(ied)
	live := mapReader{"LD/A.q": int64(4)}
	result, err := ix.Search(ParseQuery(ParseArgs("bad_quality=true")), live)
	if err != nil || pyv.Get(result, "total") != int64(1) {
		t.Fatalf("live bad quality: %v %v", result, err)
	}
	item := pyv.Obj(pyv.GetList(result, "items")[0])
	if pyv.Get(item, "quality") != int64(4) || pyv.Get(item, "timestamp") != int64(0) {
		t.Fatalf("live values: %v", pyv.Repr(item))
	}
	if stored, _ := ix.Search(ParseQuery(ParseArgs("bad_quality=true")), nil); pyv.Get(stored, "total") != int64(0) {
		t.Fatal("stored values must not be bad")
	}
}

type mapReader map[string]any

func (m mapReader) Value(ref string) (any, bool) {
	value, found := m[ref]
	return value, found
}

func TestCacheRebuildsOnlyForANewModelObject(t *testing.T) {
	var cache Cache
	first := pyv.O("id", "a", "signals", []any{})
	ix := cache.For(first)
	if cache.For(first) != ix {
		t.Fatal("same model object must reuse the index")
	}
	second := pyv.O("id", "a", "signals", []any{})
	if cache.For(second) == ix || cache.For(second).IED() != second {
		t.Fatal("a new model object must rebuild the index")
	}
	var group sync.WaitGroup
	for index := 0; index < 16; index++ {
		group.Add(1)
		go func(n int) {
			defer group.Done()
			cache.For(pyv.O("id", strconv.Itoa(n%4), "signals", []any{}))
		}(index)
	}
	group.Wait()
	cache.Forget("a")
	if cache.For(second) == nil {
		t.Fatal("rebuild after Forget")
	}
}
