package native

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func TestWorkersRunInsideTheKillOnCloseJob(t *testing.T) {
	client, _ := startTemplate(t, nil, "127.63.0.31", 21031)
	if err := adopt(client.Pid()); err != nil {
		t.Fatalf("worker is not in the server's job: %v", err)
	}
}

func TestLiveHelloAndRead(t *testing.T) {
	live, err := StartLive(nativeBin(t, "studio_live.exe"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()
	hello, err := live.Hello()
	if err != nil || !HasCapabilities(hello, LiveRequired) {
		t.Fatalf("hello %v %v", hello, err)
	}
	_, _ = startTemplate(t, nil, "127.63.0.32", 21032)
	if _, err := live.Connect("127.63.0.32", 21032); err != nil {
		t.Fatal(err)
	}
	read, err := live.Read([]any{pyjson.NewObject().Set("ref", "BayACTRL/XCBR1.Pos.stVal").Set("fc", "ST")})
	if err != nil {
		t.Fatal(err)
	}
	values, _ := read.Get("values")
	first := values.([]any)[0].(*pyjson.Object)
	if value, ok := first.Get("value"); !ok || value == nil {
		encoded, _ := pyjson.Compact(first)
		t.Fatalf("read %s", encoded)
	}
	if _, err := live.Disconnect(); err != nil {
		t.Fatal(err)
	}
}

func TestGooseEngineHelloAndOpen(t *testing.T) {
	goose, err := StartGoose(nativeBin(t, "studio_goose.exe"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer goose.Close()
	opened, err := goose.Open([]any{pyjson.NewObject().Set("id", "ied1").Set("gocb_refs", []any{"IED1LD0/LLN0$GO$gcb1"})})
	if err != nil {
		t.Fatal(err)
	}
	sources, _ := opened.Get("sources")
	mac, _ := sources.(*pyjson.Object).Get("ied1")
	if text, _ := mac.(string); !strings.HasPrefix(text, "02-") {
		t.Fatalf("source MAC %v", opened)
	}
	if _, err := goose.Stats(); err != nil {
		t.Fatal(err)
	}
}
