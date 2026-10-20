package main

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"studio-engine/internal/testutil"
)

func testPayload() fstest.MapFS {
	return fstest.MapFS{
		keyFile:                             {Data: []byte("0123456789abcdef\n")},
		"ScadaBench.exe":                    {Data: []byte("engine")},
		"ScadaBench.exe.sig":                {Data: []byte("sig")},
		"_internal/native/studio_ied.exe":   {Data: []byte("ied")},
		"_internal/schemas/SCL2007B4/a.xsd": {Data: []byte("<xs/>")},
	}
}

func TestStageCopiesOnceAndReuses(t *testing.T) {
	data := testutil.TempDir(t)
	folder, err := stage(testPayload(), data)
	if err != nil {
		t.Fatal(err)
	}
	if folder != filepath.Join(data, "app", "0123456789abcdef") {
		t.Fatalf("folder %s", folder)
	}
	if body, _ := os.ReadFile(filepath.Join(folder, "_internal", "native", "studio_ied.exe")); string(body) != "ied" {
		t.Fatalf("worker copy %q", body)
	}
	if _, err := os.Stat(filepath.Join(folder, keyFile)); !os.IsNotExist(err) {
		t.Fatal("the key file must not be copied")
	}
	marker := filepath.Join(folder, "ScadaBench.exe")
	stamp, _ := os.Stat(marker)
	again, err := stage(testPayload(), data)
	if err != nil || again != folder {
		t.Fatalf("second stage %s %v", again, err)
	}
	if now, _ := os.Stat(marker); !now.ModTime().Equal(stamp.ModTime()) {
		t.Fatal("a complete copy must be reused, not rewritten")
	}
}

func TestStageRepairsAnIncompleteCopy(t *testing.T) {
	data := testutil.TempDir(t)
	folder, err := stage(testPayload(), data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "ScadaBench.exe"), []byte("cut"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := stage(testPayload(), data); err != nil {
		t.Fatal(err)
	}
	if body, _ := os.ReadFile(filepath.Join(folder, "ScadaBench.exe")); string(body) != "engine" {
		t.Fatalf("repaired copy %q", body)
	}
	entries, _ := os.ReadDir(filepath.Join(data, "app"))
	if len(entries) != 1 {
		t.Fatalf("leftover staging folders: %v", entries)
	}
}

func TestStageRefusesAPayloadWithoutKey(t *testing.T) {
	payload := testPayload()
	delete(payload, keyFile)
	if _, err := stage(payload, testutil.TempDir(t)); err == nil {
		t.Fatal("a build without payload.key must not stage")
	}
	payload[keyFile] = &fstest.MapFile{Data: []byte("../../evil")}
	if _, err := stage(payload, testutil.TempDir(t)); err == nil {
		t.Fatal("a malformed key must not name a folder")
	}
}

func TestEngineArguments(t *testing.T) {
	cases := map[string][2][]string{
		"double-click": {nil, {"--data-dir", `D:\p\ScadaBench-data`}},
		"self-test":    {{"--self-test"}, {"--data-dir", `D:\p\ScadaBench-data`, "--self-test"}},
		"own data":     {{"--data-dir", "X"}, {"--data-dir", "X"}},
		"own data =":   {{"--data-dir=X"}, {"--data-dir=X"}},
		"helper":       {{"--network-helper", "a"}, {"--network-helper", "a"}},
	}
	for name, c := range cases {
		got := engineArguments(c[0], `D:\p\ScadaBench-data`)
		if len(got) != len(c[1]) {
			t.Fatalf("%s: %q", name, got)
		}
		for index := range got {
			if got[index] != c[1][index] {
				t.Fatalf("%s: %q", name, got)
			}
		}
	}
}
