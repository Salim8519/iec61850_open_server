package archive

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"

	"studio-engine/internal/pyjson"
)

// testDoc builds a compact format-15 document with the given SCL texts (one IED each).
func testDoc(texts ...string) *pyjson.Object {
	node := pyjson.NewObject().Set("id", "cb1").Set("type", "equipment").
		Set("position", pyjson.NewObject().Set("x", 360.0).Set("y", 1e-05)).
		Set("data", pyjson.NewObject().Set("label", "CB é\U0001F600 <&> \x7f").Set("rotation", int64(0)))
	diagram := pyjson.NewObject().Set("nodes", []any{node}).Set("edges", []any{})
	network := pyjson.NewObject().Set("mode", "local").Set("nominal_hz", 50.0)
	doc := pyjson.NewObject().Set("id", "0123456789abcdef0123456789abcdef").Set("name", "Demo").
		Set("diagram", diagram).Set("sources", []any{}).Set("scenarios", []any{}).
		Set("test_panel", []any{}).Set("network", network).Set("gateway", pyjson.NewObject()).
		Set("format", int64(15))
	ieds := []any{}
	table := pyjson.NewObject()
	for index, text := range texts {
		digest := sha256Hex(text)
		table.Set(digest, text)
		ieds = append(ieds, pyjson.NewObject().Set("id", sha256Hex(text)[:32]).
			Set("name", "IED"+string(rune('A'+index))).Set("port", int64(102)).Set("source_id", digest))
	}
	return doc.Set("ieds", ieds).Set("scl_sources", table)
}

func sha256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// unzip returns the project.json bytes after archive/zip verified size and CRC.
func unzip(t testing.TB, payload []byte) ([]byte, *zip.File) {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 1 {
		t.Fatalf("%d entries", len(reader.File))
	}
	stream, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	return data, reader.File[0]
}

// zipOf builds an archive with Go's writer (for decode tests with odd members).
func zipOf(t testing.TB, members map[string][]byte, order ...string) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for _, name := range order {
		w, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(members[name])
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
