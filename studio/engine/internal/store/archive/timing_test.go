package archive

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// publicSCL reads the public sample files (never test_cids or customer SCL).
func publicSCL(t testing.TB) []string {
	root := filepath.Join("..", "..", "..", "..", "..")
	names, _ := filepath.Glob(filepath.Join(root, "scd", "*.scd"))
	names = append(names, filepath.Join(root, "marketing", "videos", "demo", "scadabench_demo.scd"))
	var texts []string
	for _, name := range names {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Skipf("public SCL not available: %v", err)
		}
		texts = append(texts, strings.TrimPrefix(string(data), bom))
	}
	return texts
}

// bom is U+FEFF, removed like Python's utf-8-sig decoding of an SCL upload.
const bom = "\xef\xbb\xbf"

// largeSCL repeats the public files into six distinct sources of about 22 MB in total.
func largeSCL(t testing.TB) []string {
	base := strings.Join(publicSCL(t), "\n")
	var texts []string
	for index := 0; index < 6; index++ {
		var builder strings.Builder
		for builder.Len() < 22<<20/6 {
			fmt.Fprintf(&builder, "<!-- copy %d/%d -->\n%s", index, builder.Len(), base)
		}
		texts = append(texts, builder.String())
	}
	return texts
}

func withDiagram(doc *pyjson.Object, edges int) *pyjson.Object {
	changed := pyjson.NewObject()
	for _, key := range doc.Keys() {
		value, _ := doc.Get(key)
		changed.Set(key, value)
	}
	items := make([]any, edges)
	for index := range items {
		items[index] = pyjson.NewObject().Set("id", fmt.Sprintf("e%d", index)).Set("source", "a").Set("target", "b")
	}
	return changed.Set("diagram", pyjson.NewObject().Set("nodes", []any{}).Set("edges", items))
}

func measure(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// TestTimingReport logs the timings quoted in the change record (go test -run Timing -v).
func TestTimingReport(t *testing.T) {
	if testing.Short() {
		t.Skip("timing report")
	}
	for _, set := range []struct {
		name  string
		texts []string
	}{{"public 4 SCD", publicSCL(t)}, {"22 MB SCL", largeSCL(t)}} {
		doc := testDoc(set.texts...)
		var cold, warm []byte
		encoder := NewEncoder()
		full := measure(func() { cold, _ = Encode(doc) })
		_, _ = encoder.Encode(doc)
		edited := withDiagram(doc, 200)
		cached := measure(func() { warm, _ = encoder.Encode(edited) })
		var decoded *pyjson.Object
		decode := measure(func() { decoded, _ = Decode(cold) })
		dumps := measure(func() { _, _ = pyjson.Dumps(doc) })
		data, _ := unzip(t, warm)
		want, _ := pyjson.Dumps(edited)
		if !bytes.Equal(data, want) || decoded == nil {
			t.Fatal("timing run produced a wrong archive")
		}
		t.Logf("%s: json %d bytes, archive %d bytes; full encode %v, cached-segment encode %v, decode %v, pyjson.Dumps %v",
			set.name, len(want), len(cold), full, cached, decode, dumps)
	}
}

func BenchmarkCachedEncode22MB(b *testing.B) {
	doc := testDoc(largeSCL(b)...)
	encoder := NewEncoder()
	_, _ = encoder.Encode(doc)
	edited := withDiagram(doc, 200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := encoder.Encode(edited); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCachedEncodePublic(b *testing.B) {
	doc := testDoc(publicSCL(b)...)
	encoder := NewEncoder()
	_, _ = encoder.Encode(doc)
	edited := withDiagram(doc, 200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := encoder.Encode(edited); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodePublic(b *testing.B) {
	payload, _ := Encode(testDoc(publicSCL(b)...))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Decode(payload); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFullEncode22MB(b *testing.B) {
	doc := testDoc(largeSCL(b)...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Encode(doc); err != nil {
			b.Fatal(err)
		}
	}
}
