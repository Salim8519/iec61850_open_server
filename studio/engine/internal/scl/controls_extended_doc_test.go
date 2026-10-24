package scl

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"studio-engine/internal/pyjson"
)

// sclFiles are the public fixtures, plus folders named in STUDIO_SCL_CORPUS (the local
// vendor_cids and test_cids, never committed; ADR-0116 ran 185 files this way).
func sclFiles(t *testing.T) []string {
	t.Helper()
	files, roots := []string{}, []string{}
	for _, folder := range []string{"studio/tests/fixtures", "scd"} {
		roots = append(roots, filepath.Join(repoRoot, folder))
	}
	if extra := os.Getenv("STUDIO_SCL_CORPUS"); extra != "" { // e.g. the main checkout's local files
		roots = append(roots, filepath.SplitList(extra)...)
	}
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".scd", ".cid", ".icd", ".iid", ".ssd", ".sed":
				files = append(files, path)
			}
			return nil
		})
	}
	return files
}

func answer(list []any, err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	data, dumpErr := pyjson.Dumps(list)
	if dumpErr != nil {
		return "dump error: " + dumpErr.Error()
	}
	return string(data)
}

// TestDocumentReadMatchesExtendedControls: reading a file once for all its IEDs and building
// only typed-control objects gives byte-identical answers and errors to ExtendedControls.
func TestDocumentReadMatchesExtendedControls(t *testing.T) {
	files, compared, typed, refused := sclFiles(t), 0, 0, 0
	for _, path := range files {
		source, err := os.ReadFile(path)
		if err != nil || len(source) > 100<<20 {
			continue
		}
		results, docErr := extendedForDocument(source)
		names := []string{"", "NoSuchIED"}
		if doc, err := ParseDocument(source, false); err == nil {
			for _, ied := range doc.Root.FindAll("IED") {
				names = append(names, ied.GetOr("name", ""))
			}
		}
		for _, name := range names {
			want := answer(ExtendedControls(source, name))
			var got string
			if docErr != nil {
				got = answer(nil, docErr)
			} else if result, found := results[name]; found {
				got = answer(result.list, result.err)
			} else {
				got = answer([]any{}, nil)
			}
			if got != want && strings.HasPrefix(want, "error: ") && docErr == nil && refusedByImport(source, name) {
				refused++ // never in a project, so never asked at Run
				continue
			}
			if got != want {
				t.Fatalf("%s IED %q:\n got %.300s\nwant %.300s", path, name, got, want)
			}
			compared++
			if strings.HasPrefix(want, "[{") {
				typed++
			}
		}
	}
	t.Logf("%d files, %d IED answers identical (%d with typed controls); %d differ only for IEDs import refuses",
		len(files), compared, typed, refused)
	if compared == 0 {
		t.Fatal("no SCL files found")
	}
}

type acceptAll struct{}

func (acceptAll) Validate([]byte, string) (SchemaVerdict, error) { return SchemaVerdict{Status: "valid"}, nil }

// refusedByImport: the importer refuses this IED, so no project holds it.
func refusedByImport(source []byte, name string) bool {
	_, err := (&Importer{Validator: acceptAll{}}).ImportNamed(source, "x.scd", Selection{Names: []any{name}})
	return err != nil
}

// TestDocumentReadOncePerText: IEDs of one file asking together share one read.
func TestDocumentReadOncePerText(t *testing.T) {
	documentCalls.mu.Lock()
	documentCalls.calls = map[string]*documentCall{}
	documentCalls.mu.Unlock()
	source := string(typedSample(t))
	want := answer(ExtendedControls([]byte(source), "SampleTyped"))
	var group sync.WaitGroup
	answers := make([]string, 8)
	for index := range answers {
		group.Add(1)
		go func() {
			defer group.Done()
			answers[index] = answer(extendedFromDocument(source, "SampleTyped"))
		}()
	}
	group.Wait()
	for _, got := range answers {
		if got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	}
	documentCalls.mu.Lock()
	defer documentCalls.mu.Unlock()
	reads := 0
	for range documentCalls.calls {
		reads++
	}
	if reads != 1 {
		t.Fatalf("%d reads of one text", reads)
	}
}
