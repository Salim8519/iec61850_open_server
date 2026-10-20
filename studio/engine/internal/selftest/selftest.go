// Package selftest is "ScadaBench.exe --self-test" for the Go engine (launcher/self_test.py):
// the composed engine runs in process on temporary data inside the selected data folder,
// imports the SelfTest template with an MV array, runs it on loopback, reads values back
// with the independent studio_probe.exe, exports and re-imports an edited SCL, then a
// second engine recovers the stopped project, deletes and restores the IED. It prints the
// same JSON line as Python.
package selftest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"studio-engine/internal/app"
	"studio-engine/internal/launcher"
	"studio-engine/internal/pyjson"
)

// Options are the Settings and create_app arguments the check composes the engine with.
type Options struct {
	DataDir     string // the selected data folder; the temporary folder is made inside it
	NativeDir   string // folder of studio_ied.exe and studio_probe.exe
	FrontendDir string // frontend.pak or dist folder; empty: none
	SchemasDir  string // bundled XSD folders (SCL2003, SCL2007B, SCL2007B4)
	XSDHelper   string // studio_xsd.exe; empty means <NativeDir>/studio_xsd.exe
	XSDSeed     []string
	Version     string
	Enforce     bool   // license enforcement (the packaged app)
	LicenseHome string // "" means licensing.DefaultHome()
	Logf        func(format string, args ...any)
}

// Run is entrypoint.py's --self-test branch: the JSON line on stdout and exit code 0, or
// the error on stderr and exit code 1 (Python's traceback and exit status 1).
func Run(options Options, stdout, stderr io.Writer) int {
	text, err := Check(options)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, text)
	return 0
}

// Check is self_test.check(settings) followed by json.dumps: the exact text Python printed.
func Check(options Options) (text string, err error) {
	if err := os.MkdirAll(options.DataDir, 0o755); err != nil {
		return "", err
	}
	temporary, err := os.MkdirTemp(options.DataDir, "studio-check-")
	if err != nil {
		return "", err
	}
	defer func() {
		if removal := removeAll(temporary); removal != nil && err == nil {
			text, err = "", removal
		}
	}()
	directory, err := resolve(temporary)
	if err != nil {
		return "", err
	}
	base, err := resolve(options.DataDir)
	if err != nil {
		return "", err
	}
	if !within(directory, base) {
		return "", errors.New("Self-test directory escaped the selected data folder")
	}
	check := &run{options: options, directory: directory}
	if err := check.licensedRun(); err != nil {
		return "", err
	}
	if err := check.recoveryRun(); err != nil {
		return "", err
	}
	result := pyjson.NewObject().Set("ok", true).Set("schema", "SCL2007B4").Set("mms_value", 42.5).
		Set("array_value", 17.25).Set("edited_export_verified", true).Set("stopped_recovery_verified", true).
		Set("ied_deletion_verified", true).Set("saved_panel_and_pulse_verified", true).
		Set("customer_data_used", false).Set("clean_machine_verified", false)
	encoded, err := pyjson.Dumps(result)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// run is one check on its temporary folder.
type run struct {
	options   Options
	directory string
	identity  any // the project id the recovered engine must open
}

// compose is create_app(Settings(directory, native, frontend), token): a fresh secret key
// per engine, as create_app made one when none was given.
func (r *run) compose(token string) (*app.Engine, error) {
	o := r.options
	return app.Compose(app.Options{DataDir: r.directory, NativeDir: o.NativeDir, FrontendDir: o.FrontendDir,
		SchemasDir: o.SchemasDir, XSDHelper: o.XSDHelper, XSDSeed: o.XSDSeed, Token: token,
		SecretKey: launcher.SecretKey(), Version: o.Version, Enforce: o.Enforce, LicenseHome: o.LicenseHome,
		Logf: o.Logf})
}

// resolve is Path.resolve(): absolute, links followed.
func resolve(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absolute)
}

// removeAll is TemporaryDirectory's cleanup, patient with Windows: a file a worker or a
// scanner just closed can stay "delete pending" for a moment.
func removeAll(dir string) error {
	var err error
	for attempt := 0; attempt < 100; attempt++ {
		if err = os.RemoveAll(dir); err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("remove the self-test folder %s: %w", dir, err)
}
