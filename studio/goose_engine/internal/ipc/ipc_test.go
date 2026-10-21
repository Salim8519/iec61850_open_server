package ipc_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"

	"studio-goose-engine/internal/ipc"
)

func frame(t *testing.T, message any) []byte {
	t.Helper()
	payload, _ := json.Marshal(message)
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, uint32(len(payload)))
	return append(header, payload...)
}

func replies(t *testing.T, out *bytes.Buffer) []map[string]any {
	t.Helper()
	var all []map[string]any
	for out.Len() > 0 {
		size := binary.LittleEndian.Uint32(out.Next(4))
		var message map[string]any
		if err := json.Unmarshal(out.Next(int(size)), &message); err != nil {
			t.Fatal(err)
		}
		all = append(all, message)
	}
	return all
}

func TestServeAnswersOpsReportsErrorsAndStops(t *testing.T) {
	in := bytes.Buffer{}
	in.Write(frame(t, map[string]any{"version": 1, "id": "a", "op": "echo", "value": 7}))
	in.Write(frame(t, map[string]any{"version": 1, "id": "b", "op": "fail"}))
	in.Write(frame(t, map[string]any{"version": 1, "id": "c", "op": "missing"}))
	in.Write(frame(t, map[string]any{"version": 1, "id": "d", "op": "stop"}))
	in.Write(frame(t, map[string]any{"version": 1, "id": "e", "op": "echo"}))
	out := bytes.Buffer{}
	stopped := 0
	handlers := map[string]ipc.Handler{
		"echo": func(p json.RawMessage) (any, error) {
			var v struct{ Value int }
			return v.Value, json.Unmarshal(p, &v)
		},
		"fail": func(json.RawMessage) (any, error) { return nil, errors.New("boom") },
	}
	if err := ipc.Serve(&in, ipc.NewWriter(&out), handlers, func() { stopped++ }); err != nil {
		t.Fatal(err)
	}
	got := replies(t, &out)
	if len(got) != 4 || stopped != 1 {
		t.Fatalf("replies %v stopped %d", got, stopped)
	}
	if got[0]["result"] != 7.0 || got[1]["error"] != "boom" || got[2]["ok"] != false || got[3]["id"] != "d" {
		t.Fatalf("unexpected replies %v", got)
	}
}

func TestReadRequestRejectsEmptyAndOversizedFrames(t *testing.T) {
	for _, size := range []uint32{0, ipc.MaxFrame + 1} {
		header := make([]byte, 4)
		binary.LittleEndian.PutUint32(header, size)
		if _, err := ipc.ReadRequest(bytes.NewReader(header)); err == nil {
			t.Errorf("size %d accepted", size)
		}
	}
}
