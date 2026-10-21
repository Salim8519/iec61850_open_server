// scadabench_hostkey is the VM host helper (ADR-0099): it lets the virtual machines on this licensed
// PC use its license, offline. It is started at Windows logon by a per-user Run value that exists
// only while "Let my virtual machines use this license" is on (Settings > License), answers proof
// requests on TCP port 47862 only on the adapters of virtual switches, and refuses to serve unless
// this PC's machine ID, certificate and DPAPI key agree. It has no window.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"studio-engine/internal/licensing"
	"studio-engine/internal/licensing/hostkey"
)

// testSeam is nil in every release build. The test build (seam_test.go, build tag hostkeytest)
// sets it so an end-to-end test can serve on a loopback address of its choosing and with its own
// signing key; nothing of that exists in the shipped program.
var testSeam func(*helperConfig)

// helperConfig is what main fills in and the test seam may change.
type helperConfig struct {
	Home     string
	Options  licensing.HostOptions
	Wanted   func() bool
	Extra    func() []hostkey.Endpoint
	Adapters func() ([]hostkey.Adapter, error)
}

func main() {
	home := licensing.DefaultHome()
	logFile := openLog(home)
	if logFile != nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}
	release, ok := hostkey.Acquire()
	if !ok {
		return // another helper already serves this Windows session
	}
	defer release()
	sharing := hostkey.Sharing{Run: hostkey.SystemRunKey(), Process: hostkey.SystemProcess()}
	config := helperConfig{Home: home, Wanted: sharing.Wanted, Adapters: hostkey.SystemAdapters,
		Options: licensing.HostOptions{Home: home, Protector: hostkey.System(), Registry: licensing.WindowsRegistry{}}}
	if testSeam != nil {
		testSeam(&config)
	}
	if !config.Wanted() {
		log.Print("Sharing is switched off; the helper ends.")
		return
	}
	responder := licensing.NewHostResponder(config.Options)
	h := &helper{responder: responder, adapters: config.Adapters, wanted: config.Wanted, extra: config.Extra,
		logf: log.Printf, check: func() error { _, err := responder.Check(); return err }}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	h.run(ctx)
}

// openLog appends to vm_host.log in the license folder, starting afresh when it grew past 256 KB.
// The log holds no keys, certificates or machine IDs: only what was served and why not.
func openLog(home string) *os.File {
	path := filepath.Join(home, "vm_host.log")
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if info, err := os.Stat(path); err == nil && info.Size() > 256*1024 {
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}
	if err := os.MkdirAll(home, 0o777); err != nil {
		return nil
	}
	file, err := os.OpenFile(path, flags, 0o666)
	if err != nil {
		return nil
	}
	return file
}
