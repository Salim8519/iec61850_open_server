// Command studio_server is the Go engine's backend, started like the Python server mode
// (launcher/entrypoint.py): same arguments, data-folder lease, launcher.json and printed
// lines, beginning with the readiness line "ScadaBench: http://127.0.0.1:<port>". The
// release build is ScadaBench.exe (ADR-0083) and adds the packaged app window.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"studio-engine/internal/app"
	"studio-engine/internal/desktop"
	"studio-engine/internal/launcher"
)

// packaged is "1" in the packaged build (-ldflags "-X main.packaged=1"): only then is the
// license enforced, as Python enforced it only when frozen.
var packaged = ""

func main() { os.Exit(run(os.Args[1:])) }

// parityRun is set only in the parity test build (-tags parity, parity.go): it serves the
// harness CLI (--parity-seams) and answers handled=false for every other command line.
var parityRun func(args []string) (code int, handled bool)

func run(args []string) int {
	if parityRun != nil {
		if code, handled := parityRun(args); handled {
			return code
		}
	}
	if len(args) > 0 && args[0] == "--network-helper" {
		return networkHelper(args[1:])
	}
	if len(args) > 0 && args[0] == "--fold-journal" {
		return foldJournal(args[1:])
	}
	desktop.AttachParentTerminal()
	options, err := launcher.ParseOptions(args)
	var usage *launcher.UsageError
	if errors.As(err, &usage) {
		fmt.Fprintln(os.Stderr, launcher.Usage)
		if usage.Message == "help" {
			return 0
		}
		fmt.Fprintf(os.Stderr, "studio_server: error: %s\n", usage.Message)
		return 2
	}
	if options.Dev && packaged == "1" {
		// Developer mode uses a fixed session secret and token; the installed app never runs it.
		fmt.Fprintln(os.Stderr, "Developer mode (--dev) is not available in the installed ScadaBench. "+
			"Run it from source with studio/tools/run_dev.py instead.")
		return 1
	}
	if packaged == "1" {
		desktop.MigrateLegacyFolder(os.Getenv("LOCALAPPDATA"))
		announceSignedModule()
	}
	paths := resolvePaths(options)
	if options.SelfTest {
		return selfTest(paths)
	}
	lease, err := launcher.AcquireLease(paths.data)
	if err != nil && !options.Dev {
		// Another ScadaBench owns this data folder: bring its window forward instead.
		return desktop.ShowRunningInstance(paths.data, err)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer lease.Release()
	log := desktop.OpenLog(paths.data)
	defer log.Close()
	return serve(options, paths, log)
}

func serve(options launcher.Options, paths paths, log *desktop.Logger) int {
	token, secret, origin := launcher.LaunchToken(), launcher.SecretKey(), ""
	if options.Dev {
		token, secret, origin = options.DevToken, launcher.DevSecret, trimSlash(options.FrontendURL)
	}
	engine, err := app.Compose(app.Options{DataDir: paths.data, NativeDir: paths.native, FrontendDir: paths.frontend,
		SchemasDir: paths.schemas, XSDSeed: seedFiles(), Token: token, SecretKey: secret, AdminOrigin: origin,
		Version: paths.version, Enforce: packaged == "1", NativeSeal: parseSeal(nativeSeal), Logf: engineLog(log)})
	if err != nil {
		return startFailed(log, paths.data, err)
	}
	defer engine.Close()
	listener, err := net.Listen("tcp4", "127.0.0.1:"+strconv.Itoa(options.Port))
	if err != nil && options.Dev {
		fmt.Fprintf(os.Stderr, "Port %d is busy. Stop the other ScadaBench backend before starting dev mode.\n", options.Port)
		return 1
	}
	if err != nil {
		if listener, err = net.Listen("tcp4", "127.0.0.1:0"); err != nil {
			return startFailed(log, paths.data, err)
		}
	}
	url := "http://127.0.0.1:" + strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	_ = launcher.WriteRecord(paths.data, url, 0)
	fmt.Println("ScadaBench: " + url)
	if options.Dev {
		if !frontendReady(options.FrontendURL, 15*time.Second) {
			fmt.Printf("Vite is not reachable at %s; start it and open the printed address.\n", options.FrontendURL)
		}
		fmt.Printf("Vite frontend: %s/#session=%s\n", options.FrontendURL, token)
		fmt.Println("Open the address above in your browser (ScadaBench never opens tabs automatically).")
		return listen(engine, listener, nil)
	}
	target := url + "#session=" + token
	fmt.Println("Open in your browser: " + target)
	closed := make(chan struct{})
	stopShow := startWindow(options, paths.data, url, target, log, closed)
	defer stopShow()
	return listen(engine, listener, closed)
}

// listen serves until Ctrl+C (or a termination signal) or until the app window closes, then
// stops accepting requests, ends open event streams and lets the caller close the engine
// and release the lease. windowClosed is nil when there is no window to watch.
func listen(engine *app.Engine, listener net.Listener, windowClosed <-chan struct{}) int {
	server := &http.Server{Handler: engine.Handler, ReadHeaderTimeout: 30 * time.Second}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-stop:
		case <-windowClosed:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if server.Shutdown(ctx) != nil {
			_ = server.Close() // event streams never go idle
		}
	}()
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	<-stopped // no request is still running when the engine closes
	return 0
}

// seedFiles lists STUDIO_XSD_SEED: verdict files (Python lxml verdicts) imported into the
// XSD cache while studio_xsd.exe is not built (development only).
func seedFiles() []string {
	var files []string
	for _, file := range filepath.SplitList(os.Getenv("STUDIO_XSD_SEED")) {
		if strings.TrimSpace(file) != "" {
			files = append(files, file)
		}
	}
	return files
}

func trimSlash(text string) string {
	for len(text) > 0 && text[len(text)-1] == '/' {
		text = text[:len(text)-1]
	}
	return text
}
