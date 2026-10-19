package scl

import (
	"encoding/base64"
	"testing"

	"studio-engine/internal/scl/pyrules"
)

// TestDecodeUTF8SigMatchesPython replays bytes.decode("utf-8-sig") results from CPython.
func TestDecodeUTF8SigMatchesPython(t *testing.T) {
	for index, item := range readGoldenList(t, "utf8_errors.json") {
		input, err := base64.StdEncoding.DecodeString(text(item, "input"))
		if err != nil {
			t.Fatal(err)
		}
		got := "OK"
		if _, err := pyrules.DecodeUTF8Sig(input); err != nil {
			got = err.Error()
		}
		if want := text(item, "result"); got != want {
			t.Errorf("case %d %x: got %q want %q", index, input, got, want)
		}
	}
}
