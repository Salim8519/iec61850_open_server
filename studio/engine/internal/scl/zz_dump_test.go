package scl

// TEMPORARY (never committed): dump every SCL reader's answer per corpus file for an old/new diff.
import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

type dumpAccept struct{}

func (dumpAccept) Validate([]byte, string) (SchemaVerdict, error) { return SchemaVerdict{Status: "valid"}, nil }

func TestZZDumpCorpus(t *testing.T) {
	out := os.Getenv("DUMP_OUT")
	if out == "" {
		t.Skip("DUMP_OUT not set")
	}
	_ = os.MkdirAll(out, 0o755)
	var files []string
	for _, root := range filepath.SplitList(os.Getenv("STUDIO_SCL_CORPUS")) {
		_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				switch strings.ToLower(filepath.Ext(p)) {
				case ".scd", ".cid", ".icd", ".iid":
					files = append(files, p)
				}
			}
			return nil
		})
	}
	sort.Strings(files)
	dump := func(v any, err error) string {
		if err != nil {
			return "ERR " + err.Error()
		}
		b, e := pyjson.Dumps(v)
		if e != nil {
			return "DUMPERR " + e.Error()
		}
		return string(b)
	}
	for _, path := range files {
		source, err := os.ReadFile(path)
		if err != nil || len(source) > 100<<20 {
			continue
		}
		sum := sha256.Sum256([]byte(path))
		var b strings.Builder
		fmt.Fprintf(&b, "FILE %s\n", path)
		models, perr := Preview(source)
		fmt.Fprintf(&b, "PREVIEW %s\n", dump(objects(models), perr))
		counter := 0
		im := &Importer{Validator: dumpAccept{}, NewID: func() string { counter++; return fmt.Sprint("id", counter) }}
		if perr == nil {
			for _, m := range models {
				name, _ := m.Get("name")
				counter = 0
				built, err := im.ImportNamed(source, "x.scd", Selection{Names: []any{name}})
				fmt.Fprintf(&b, "IMPORT %v %s\n", name, dump(objects(built), err))
				ext, err := ExtendedControls(source, fmt.Sprint(name))
				fmt.Fprintf(&b, "EXT %v %s\n", name, dump(ext, err))
				classes, err := DataClasses(source, fmt.Sprint(name))
				keys := []string{}
				for k, v := range classes {
					keys = append(keys, k+"="+v)
				}
				sort.Strings(keys)
				fmt.Fprintf(&b, "CLASSES %v %v %s\n", name, err, strings.Join(keys, ","))
			}
		}
		gs, err := ReadGooseSources([]GooseDocument{{Text: string(source)}})
		fmt.Fprintf(&b, "GOOSE %s\n", dump(gs, err))
		flow, err := ReadFlow(string(source))
		if err == nil {
			fmt.Fprintf(&b, "FLOW %s\n", dump(pyjson.NewObject().Set("ieds", toAny(flow.IEDs)), nil))
			names := []string{}
			for n := range flow.Subscriptions {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				s := flow.Subscriptions[n]
				fmt.Fprintf(&b, "FLOWSUB %s %s\n", n, dump([]any{s.Subscriptions, s.Problems}, nil))
			}
		} else {
			fmt.Fprintf(&b, "FLOW ERR %s\n", err)
		}
		_ = os.WriteFile(filepath.Join(out, hex.EncodeToString(sum[:8])+".txt"), []byte(b.String()), 0o644)
	}
}

func toAny(list []*pyjson.Object) []any {
	out := make([]any, len(list))
	for i, v := range list {
		out[i] = v
	}
	return out
}
