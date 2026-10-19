package archive

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// TestDecodeVectorsFromPython replays archive.decode outcomes recorded by
// testdata/crosscheck.py (CPython 3.12): StudioError texts, UnicodeDecodeError texts of
// json.loads(bytes), and the documents of accepted UTF-8/16/32 payloads.
func TestDecodeVectorsFromPython(t *testing.T) {
	data, err := os.ReadFile("../testdata/fixtures/decode_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	value, err := pyjson.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	for index, item := range value.([]any) {
		vector := item.(*pyjson.Object)
		payloadHex, _ := vector.Get("payload")
		payload, _ := hex.DecodeString(payloadHex.(string))
		doc, err := Decode(payload)
		if ok, found := vector.Get("ok"); found {
			want, _ := pyjson.Dumps(ok)
			got, _ := pyjson.Dumps(doc)
			if err != nil || !bytes.Equal(got, want) {
				t.Errorf("vector %d: got %s (%v) want %s", index, got, err, want)
			}
			continue
		}
		kindValue, _ := vector.Get("kind")
		message, _ := vector.Get("message")
		kind := apperr.Value
		if kindValue == "StudioError" {
			kind = apperr.Studio
		}
		appErr, isAppErr := err.(*apperr.Error)
		if !isAppErr || appErr.Kind != kind || appErr.Message != message {
			t.Errorf("vector %d: got %v want %s %q", index, err, kindValue, message)
		}
	}
}
