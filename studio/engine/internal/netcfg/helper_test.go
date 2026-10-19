package netcfg

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type helperVectors struct {
	Validation []struct {
		outcome
		Payload string `json:"payload"`
		Digest  string `json:"digest"`
	} `json:"validation"`
	Runner []struct {
		Payload string `json:"payload"`
		Calls   []struct {
			Argv    []string `json:"argv"`
			Timeout float64  `json:"timeout"`
		} `json:"calls"`
		Exit   int     `json:"exit"`
		Raised string  `json:"raised"`
		Result *string `json:"result"`
	} `json:"runner"`
	HelperCommand []any    `json:"helper_command"`
	ListToCmdline [][2]any `json:"list2cmdline"`
}

const vectorProgram = `C:\Studio\native\studio_ied.exe`

// helperAnswers are make_helper.ANSWERS: canned PowerShell outcomes, used in order.
var helperAnswers = []Output{{0, "", ""},
	{1, "", "New-NetIPAddress : The object already exists.\r\n  + CategoryInfo : x\r\n"},
	{1, "out only\nlast out\n", ""}, {0, "warning\n", "  \n"}, {1, "", repeatText("é", 600) + "\n"},
	{5, "", "line\x1cnext\u2028final  "}}

func repeatText(text string, count int) string {
	out := ""
	for index := 0; index < count; index++ {
		out += text
	}
	return out
}

func TestHelperValidationMatchesPython(t *testing.T) {
	var v helperVectors
	loadVectors(t, "helper.json", &v)
	for index, c := range v.Validation {
		payload, _ := base64.StdEncoding.DecodeString(c.Payload)
		request, err := ValidateRequest(payload, c.Digest, vectorProgram)
		got := ""
		if err == nil {
			got = dumpsText(t, request)
		}
		want := c.outcome
		if want.Error == "AttributeError" || want.Error == "TypeError" {
			if !errors.Is(err, errHelperCrash) {
				t.Errorf("validation %d %s: want a crash (%s), got %v", index, payload, want.Error, err)
			}
			continue
		}
		checkOutcome(t, fmt.Sprintf("validation %d %s", index, payload), got, err, want, true)
	}
}

func TestHelperRunnerMatchesPython(t *testing.T) {
	var v helperVectors
	loadVectors(t, "helper.json", &v)
	folder := t.TempDir()
	for index, c := range v.Runner {
		payload, _ := base64.StdEncoding.DecodeString(c.Payload)
		request := filepath.Join(folder, fmt.Sprintf("case-%d.json", index))
		os.WriteFile(request, payload, 0o666)
		answers := append([]Output(nil), helperAnswers...)
		calls := []Command{}
		fake := ExecutorFunc(func(command Command) (Output, error) {
			calls = append(calls, command)
			answer := answers[0]
			answers = answers[1:]
			return answer, nil
		})
		exit := HelperMain([]string{request, sha256Hex(payload), vectorProgram}, fake)
		if exit != c.Exit {
			t.Errorf("runner %d: exit %d want %d", index, exit, c.Exit)
		}
		if len(calls) != len(c.Calls) {
			t.Fatalf("runner %d: %d commands want %d", index, len(calls), len(c.Calls))
		}
		for number, call := range calls {
			want := c.Calls[number]
			if !reflect.DeepEqual(call.Argv, want.Argv) || call.Timeout != time.Duration(want.Timeout)*time.Second {
				t.Errorf("runner %d call %d:\n got %q %v\nwant %q %v", index, number, call.Argv, call.Timeout, want.Argv, want.Timeout)
			}
		}
		data, err := os.ReadFile(withSuffix(request, ".result.json"))
		switch {
		case c.Result == nil && err == nil:
			t.Errorf("runner %d: result file written where Python crashed", index)
		case c.Result != nil:
			want, _ := base64.StdEncoding.DecodeString(*c.Result)
			if string(data) != string(want) {
				t.Errorf("runner %d result:\n got %s\nwant %s", index, data, want)
			}
		}
	}
}

func TestHelperCommandAndQuoting(t *testing.T) {
	var v helperVectors
	loadVectors(t, "helper.json", &v)
	exe, args := HelperCommand(v.HelperCommand[0].(string), `C:\data dir\network\requests\apply-ab.json`,
		repeatText("f", 64), vectorProgram)
	if got := append([]any{exe}, []any{toAny(args)}...); fmt.Sprint(got) != fmt.Sprint(v.HelperCommand) {
		t.Errorf("helper command %v want %v", got, v.HelperCommand)
	}
	for _, pair := range v.ListToCmdline {
		args := []string{}
		for _, item := range pair[0].([]any) {
			args = append(args, item.(string))
		}
		if got := ListToCmdline(args); got != pair[1] {
			t.Errorf("list2cmdline %q: %q want %q", args, got, pair[1])
		}
	}
	if withSuffix(`C:\x\apply-1.json`, ".result.json") != `C:\x\apply-1.result.json` ||
		withSuffix(`C:\x\.json`, ".r") != `C:\x\.json.r` || withSuffix(`C:\x\a.`, ".r") != `C:\x\a..r` {
		t.Error("with_suffix differs from pathlib")
	}
}

func toAny(values []string) []any {
	items := make([]any, len(values))
	for index, value := range values {
		items[index] = value
	}
	return items
}
