package selftest

import (
	"path/filepath"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// CheckError is a failed self-test check: an AssertionError in self_test.py. Check names
// what failed (Python printed it only in its traceback); Detail is the message Python's
// assert carried, empty for a bare assert.
type CheckError struct {
	Check  string
	Detail string
}

func (e *CheckError) Error() string {
	if e.Detail == "" {
		return "Self-test check failed: " + e.Check
	}
	return "Self-test check failed: " + e.Check + ": " + e.Detail
}

func failed(check, detail string) error { return &CheckError{Check: check, Detail: detail} }

// expectStatus is `assert response.status_code == want` (with the JSON body as the message
// when Python gave one).
func expectStatus(got response, want int, check string, withBody bool) error {
	if got.status == want {
		return nil
	}
	if withBody {
		return failed(check, pyval.Repr(got.json()))
	}
	return failed(check, "")
}

// within is Path.is_relative_to on Windows paths: a lexical, case-insensitive prefix test.
func within(path, folder string) bool {
	path, folder = filepath.Clean(path), filepath.Clean(folder)
	if strings.EqualFold(path, folder) {
		return true
	}
	prefix := folder
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return len(path) > len(prefix) && strings.EqualFold(path[:len(prefix)], prefix)
}

// firstIED is project["ieds"][0] (nil when the list is empty).
func firstIED(project *pyjson.Object) *pyjson.Object {
	list := pyval.List(project, "ieds")
	if len(list) == 0 {
		return nil
	}
	ied, _ := list[0].(*pyjson.Object)
	return ied
}

// emptyList is `value == []`.
func emptyList(value any) bool {
	list, ok := value.([]any)
	return ok && len(list) == 0
}
