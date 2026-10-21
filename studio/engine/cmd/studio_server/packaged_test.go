package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPackagedBuildNeedsItsSignatureFile(t *testing.T) {
	folder := t.TempDir()
	executable := filepath.Join(folder, "ScadaBench.exe")
	for _, name := range []string{"STUDIO_MODULE_PATH", "STUDIO_MODULE_SIG"} {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}
	refused := func(reason string) {
		t.Helper()
		if err := announceModule(executable); err == nil || err.Error() != signatureMissing {
			t.Fatalf("%s: got %v, want the signature message", reason, err)
		}
		if os.Getenv("STUDIO_MODULE_PATH") != "" || os.Getenv("STUDIO_MODULE_SIG") != "" {
			t.Fatalf("%s: the native workers must not be pointed at an unsigned module", reason)
		}
	}
	refused("no signature file")
	if err := os.WriteFile(executable+".sig", nil, 0o666); err != nil {
		t.Fatal(err)
	}
	refused("empty signature file")
	os.Remove(executable + ".sig")
	if err := os.Mkdir(executable+".sig", 0o777); err != nil {
		t.Fatal(err)
	}
	refused("a folder where the signature should be")
	os.Remove(executable + ".sig")
	if err := os.WriteFile(executable+".sig", []byte("signature"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := announceModule(executable); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("STUDIO_MODULE_PATH") != executable || os.Getenv("STUDIO_MODULE_SIG") != executable+".sig" {
		t.Fatal("the signed module was not announced to the native workers")
	}
	os.Unsetenv("STUDIO_MODULE_PATH")
	os.Unsetenv("STUDIO_MODULE_SIG")
}

// TestPackagedExeRefusesToStartWithoutItsSignature builds the packaged variant (-X
// main.packaged=1) and starts it beside no .sig file: it must stop with the message and
// never serve; the same exe built for development is unaffected.
func TestPackagedExeRefusesToStartWithoutItsSignature(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the server")
	}
	folder := t.TempDir()
	binary := filepath.Join(folder, "ScadaBench.exe")
	command := exec.Command("go", "build", "-ldflags", "-X main.packaged=1", "-o", binary, ".")
	command.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
	run := exec.Command(binary, "--no-browser", "--data-dir", filepath.Join(folder, "data"), "--port", "0")
	run.Env = append(os.Environ(), "LOCALAPPDATA="+filepath.Join(folder, "local"))
	var stderr bytes.Buffer
	run.Stderr = &stderr
	err := run.Run()
	exit, isExit := err.(*exec.ExitError)
	if !isExit || exit.ExitCode() != 1 || !strings.Contains(stderr.String(), signatureMissing) {
		t.Fatalf("exit %v, stderr %q", err, stderr.String())
	}
	if _, statErr := os.Stat(filepath.Join(folder, "data", "launcher.json")); statErr == nil {
		t.Fatal("the unsigned packaged exe started serving")
	}
}
