package xsd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The test binary doubles as a fake studio_xsd: with STUDIO_XSD_FAKE set it speaks the
// helper protocol instead of running tests. The document text selects the behaviour:
// CRASH exits mid-request, SLEEP hangs, DOCTYPE reports a DOCTYPE, VALID is valid and
// anything else is invalid with one error. Every validate appends a line to
// STUDIO_XSD_FAKE_LOG so tests can count helper work.
func TestMain(m *testing.M) {
	if os.Getenv("STUDIO_XSD_FAKE") != "" {
		os.Exit(fakeHelper())
	}
	os.Exit(m.Run())
}

func fakeHelper() int {
	for {
		payload, err := readFrame(os.Stdin)
		if err != nil {
			return 0
		}
		var message request
		if err := json.Unmarshal(payload, &message); err != nil {
			return 2
		}
		reply := map[string]any{"id": message.ID, "ok": true}
		switch message.Op {
		case "hello":
			reply["result"] = map[string]any{"protocol": 1, "capabilities": []string{capability},
				"libxml2":          envOr("STUDIO_XSD_FAKE_LIBXML2", Libxml2Version),
				"libxml2_compiled": envOr("STUDIO_XSD_FAKE_LIBXML2", Libxml2Version),
				"build":            envOr("STUDIO_XSD_FAKE_BUILD", "dev"), "parse_options": 4278272}
		case "stop":
			_ = writeFrame(os.Stdout, reply)
			return 0
		case "validate":
			result, exit := fakeValidate(message)
			if exit {
				return 3
			}
			if result == nil {
				reply = map[string]any{"id": message.ID, "ok": false, "error": "cannot read source"}
			} else {
				reply["result"] = result
			}
		default:
			reply = map[string]any{"id": message.ID, "ok": false, "error": "Unknown operation " + message.Op}
		}
		if err := writeFrame(os.Stdout, reply); err != nil {
			return 4
		}
	}
}

func fakeValidate(message request) (map[string]any, bool) {
	source, err := os.ReadFile(message.SourcePath)
	if err != nil {
		return nil, false
	}
	if log := os.Getenv("STUDIO_XSD_FAKE_LOG"); log != "" {
		file, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(file, "%s %d\n", filepath.Base(filepath.Dir(message.SchemaPath)), len(source))
			file.Close()
		}
	}
	switch {
	case bytes.Contains(source, []byte("CRASH")):
		return nil, true
	case bytes.Contains(source, []byte("SLEEP")):
		time.Sleep(30 * time.Second)
	case bytes.Contains(source, []byte("DOCTYPE")):
		return map[string]any{"phase": "parse", "doctype": true}, false
	case bytes.Contains(source, []byte("VALID")):
		return map[string]any{"phase": "schema", "doctype": false, "status": "valid",
			"error_count": 0, "errors": []any{}}, false
	}
	schema := filepath.Base(filepath.Dir(message.SchemaPath))
	return map[string]any{"phase": "schema", "doctype": false, "status": "invalid", "error_count": 1,
		"errors": []any{map[string]any{"line": 2, "message": fmt.Sprintf("fake %s %d bytes", schema, len(source))}}}, false
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
