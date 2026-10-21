package xsd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/scl"
	"studio-engine/internal/testutil"
)

// dumpEntry is one document recorded by testdata/parity_dump.py with Python's verdict.
type dumpEntry struct {
	File    string `json:"file"`
	Schema  string `json:"schema"`
	Label   string `json:"label"`
	Doctype bool   `json:"doctype"`
	Status  string `json:"status"`
	Errors  []struct {
		Line    int    `json:"line"`
		Message string `json:"message"`
	} `json:"errors"`
}

// TestHelperMatchesPython replays every recorded document through the real studio_xsd.exe
// (STUDIO_XSD_HELPER) with an empty verdict cache and compares status, lines and messages
// with lxml's byte for byte. STUDIO_XSD_PARITY lists dump folders (os.PathListSeparator).
// Opt-in: it needs the built helper and the dumps, which are not committed.
func TestHelperMatchesPython(t *testing.T) {
	helper, folders := os.Getenv("STUDIO_XSD_HELPER"), os.Getenv("STUDIO_XSD_PARITY")
	if helper == "" || folders == "" {
		t.Skip("set STUDIO_XSD_HELPER and STUDIO_XSD_PARITY (see testdata/parity_dump.py)")
	}
	total, starts, failures := 0, 0, []string{}
	for _, folder := range filepath.SplitList(folders) {
		// A fresh cache per folder: a document recorded twice still reaches the helper.
		validator := New(Config{HelperPath: helper, SchemasDir: bundledSchemas, DataDir: testutil.TempDir(t)})
		entries := readDump(t, folder)
		same, invalid, doctype := 0, 0, 0
		for _, entry := range entries {
			source, err := os.ReadFile(filepath.Join(folder, entry.File))
			if err != nil {
				t.Fatal(err)
			}
			got, err := validator.Validate(source, entry.Schema)
			if problem := compare(entry, got, err); problem != "" {
				failures = append(failures, fmt.Sprintf("%s [%s %s]: %s", folder, entry.Label, entry.File, problem))
				continue
			}
			same++
			if entry.Doctype {
				doctype++
			} else if entry.Status == "invalid" {
				invalid++
			}
		}
		total += len(entries)
		starts += validator.HelperStarts()
		validator.Close()
		t.Logf("%s: %d/%d identical (%d invalid, %d DOCTYPE)", folder, same, len(entries), invalid, doctype)
	}
	t.Logf("helper starts: %d, documents: %d, differences: %d", starts, total, len(failures))
	for index, failure := range failures {
		if index == 40 {
			t.Errorf("... and %d more", len(failures)-40)
			break
		}
		t.Error(failure)
	}
}

func readDump(t *testing.T, folder string) []dumpEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(folder, "entries.json"))
	if err != nil {
		t.Fatal(err)
	}
	var entries []dumpEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("%s: %v", folder, err)
	}
	return entries
}

// compare returns "" when the Go verdict is Python's, else what differs.
func compare(want dumpEntry, got scl.SchemaVerdict, err error) string {
	if want.Doctype {
		var appErr *apperr.Error
		if errors.As(err, &appErr) && appErr.Kind == apperr.Value && strings.Contains(appErr.Message, "DTD") {
			return ""
		}
		return fmt.Sprintf("want the DOCTYPE ValueError, got %+v / %v", got, err)
	}
	if err != nil {
		return "helper error: " + err.Error()
	}
	if got.Status != want.Status {
		return fmt.Sprintf("status %q, Python %q", got.Status, want.Status)
	}
	if len(got.Errors) != len(want.Errors) {
		return fmt.Sprintf("%d errors, Python %d (first: %+v vs %+v)", len(got.Errors), len(want.Errors),
			first(got.Errors), want.Errors)
	}
	for index, item := range want.Errors {
		if got.Errors[index].Line != item.Line || got.Errors[index].Message != item.Message {
			return fmt.Sprintf("error %d: line %d %q, Python line %d %q", index,
				got.Errors[index].Line, got.Errors[index].Message, item.Line, item.Message)
		}
	}
	return ""
}

func first(errors []scl.SchemaError) any {
	if len(errors) == 0 {
		return nil
	}
	return errors[0]
}
