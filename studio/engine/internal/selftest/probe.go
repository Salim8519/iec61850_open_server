package selftest

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// probeTimeout is subprocess.run(..., timeout=10).
const probeTimeout = 10 * time.Second

// readNative is self_test.read_native: an independent MMS read of reference (functional
// constraint MX) by studio_probe.exe beside the IED worker, without a console window. It
// returns float(json.loads(stdout)["value"]).
func readNative(nativeDir, host, port, reference string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, filepath.Join(nativeDir, "studio_probe.exe"), host, port, "read", reference, "MX")
	hideConsole(command)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return 0, errors.New("studio_probe.exe did not answer within 10 seconds")
	}
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		return 0, err
	}
	if err != nil {
		return 0, failed("studio_probe.exe read "+reference+" failed", stdout.String()+stderr.String())
	}
	decoded, err := pyjson.Decode(stdout.Bytes())
	if err != nil {
		return 0, err
	}
	object, ok := decoded.(*pyjson.Object)
	if !ok {
		return 0, errors.New("studio_probe.exe printed no JSON object")
	}
	value, ok := object.Get("value")
	if !ok {
		return 0, errors.New("studio_probe.exe printed no value") // Python's KeyError: 'value'
	}
	return pyval.ToFloat(value)
}
