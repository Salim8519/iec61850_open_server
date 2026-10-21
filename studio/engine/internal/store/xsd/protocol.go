package xsd

import (
	"encoding/json"
	"fmt"

	"studio-engine/internal/scl"
)

// The studio_xsd protocol (studio/native/xsd/README.md), version 1.
const (
	protocolVersion = 1
	capability      = "xsd_validate_v1"
	maxErrors       = 100
)

type request struct {
	Version    int    `json:"version"`
	ID         string `json:"id"`
	Op         string `json:"op"`
	SchemaPath string `json:"schema_path,omitempty"`
	SourcePath string `json:"source_path,omitempty"`
}

type response struct {
	ID     string          `json:"id"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

// helloResult identifies the helper binary before it is trusted with a document.
type helloResult struct {
	Protocol        int      `json:"protocol"`
	Capabilities    []string `json:"capabilities"`
	Libxml2         string   `json:"libxml2"`
	Libxml2Compiled string   `json:"libxml2_compiled"`
	Build           string   `json:"build"`
	ParseOptions    int64    `json:"parse_options"`
}

// outcome is one validate answer: a verdict, or Doctype when lxml parsed a DOCTYPE.
type outcome struct {
	Doctype bool
	Verdict scl.SchemaVerdict
}

type validateResult struct {
	Phase      string `json:"phase"`
	Doctype    bool   `json:"doctype"`
	Status     string `json:"status"`
	ErrorCount int    `json:"error_count"`
	Errors     []struct {
		Line    *int    `json:"line"`
		Message *string `json:"message"`
	} `json:"errors"`
}

func checkHello(raw json.RawMessage, expectedBuild string) error {
	var hello helloResult
	if err := json.Unmarshal(raw, &hello); err != nil {
		return errProtocol("hello: " + err.Error())
	}
	if hello.Protocol != protocolVersion || !contains(hello.Capabilities, capability) {
		return errProtocol(fmt.Sprintf("protocol %d without %s", hello.Protocol, capability))
	}
	if hello.Libxml2 != Libxml2Version || hello.Libxml2Compiled != Libxml2Version {
		found := hello.Libxml2
		if found == Libxml2Version {
			found = hello.Libxml2Compiled
		}
		return errLibxml2(found)
	}
	if expectedBuild != "" && hello.Build != expectedBuild {
		return errBuild(hello.Build, expectedBuild)
	}
	return nil
}

func decodeOutcome(raw json.RawMessage) (outcome, error) {
	var result validateResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return outcome{}, errProtocol("validate: " + err.Error())
	}
	if result.Doctype {
		return outcome{Doctype: true}, nil
	}
	if result.Status != "valid" && result.Status != "invalid" {
		return outcome{}, errProtocol(fmt.Sprintf("validate status %q", result.Status))
	}
	if len(result.Errors) > maxErrors {
		return outcome{}, errProtocol(fmt.Sprintf("%d errors, more than %d", len(result.Errors), maxErrors))
	}
	verdict := scl.SchemaVerdict{Status: result.Status, Errors: []scl.SchemaError{}}
	for _, item := range result.Errors {
		if item.Line == nil || item.Message == nil {
			return outcome{}, errProtocol("validate error without line or message")
		}
		verdict.Errors = append(verdict.Errors, scl.SchemaError{Line: *item.Line, Message: *item.Message})
	}
	return outcome{Verdict: verdict}, nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
