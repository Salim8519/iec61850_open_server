//go:build parity

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"studio-engine/internal/app"
	"studio-engine/internal/launcher"
)

// The parity build (go build -tags parity -o studio/engine/build/studio_server_parity.exe)
// serves tests/parity's engine CLI: --port --no-browser --data-dir --native-exe --frontend
// --parity-seams. Like parity/server.py it takes no lease and writes no launcher.json.
// /api/v1/_parity/* answers shutdown and the operations in parity_ops.go; write-through
// project views (project/set, project/update), runtime/state writes, supervisor reads,
// network/fault seams and service stops are not ported.

func init() { parityRun = runParity }

const parityPrefix = "/api/v1/_parity/"

func runParity(args []string) (int, bool) {
	handled := false
	for _, arg := range args {
		handled = handled || arg == "--parity-seams" || strings.HasPrefix(arg, "--parity-seams=")
	}
	if !handled {
		return 0, false
	}
	set := flag.NewFlagSet("studio_server_parity", flag.ContinueOnError)
	port := set.Int("port", 0, "")
	set.Bool("no-browser", false, "")
	dataDir, nativeExe := set.String("data-dir", "", ""), set.String("native-exe", "", "")
	frontend, seamsFile := set.String("frontend", "", ""), set.String("parity-seams", "", "")
	if err := set.Parse(args); err != nil {
		return 2, true
	}
	seams, err := readSeams(*seamsFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1, true
	}
	engine, err := composeParity(seams, *dataDir, *nativeExe, *frontend)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1, true
	}
	defer engine.Close()
	listener, err := net.Listen("tcp4", "127.0.0.1:"+strconv.Itoa(*port))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1, true
	}
	stop := make(chan struct{})
	server := &http.Server{Handler: parityHandler{engine: engine, key: seams.ParityKey, stop: stop}}
	go func() {
		<-stop
		_ = server.Close()
	}()
	fmt.Printf("ScadaBench: http://127.0.0.1:%d\n", listener.Addr().(*net.TCPAddr).Port)
	_ = server.Serve(listener)
	return 0, true
}

func composeParity(seams paritySeams, dataDir, nativeExe, frontend string) (*app.Engine, error) {
	paths := resolvePaths(launcher.Options{DataDir: dataDir, NativeExe: nativeExe})
	var gate *objectLicense
	if object, isObject := seams.License["object"]; isObject {
		found, err := newObjectLicense(object)
		if err != nil {
			return nil, err
		}
		gate, seams.License = found, nil
	}
	license, err := licenseFromSeam(seams.License)
	if err != nil {
		return nil, err
	}
	updateService, err := updatesFromSeam(seams.Updates)
	if err != nil {
		return nil, err
	}
	options := app.Options{DataDir: dataDir, NativeDir: paths.native, FrontendDir: frontend, SchemasDir: paths.schemas,
		XSDSeed: seedFiles(), Token: launcher.LaunchToken(), SecretKey: launcher.SecretKey(), Version: paths.version,
		LicenseService: license, UpdateService: updateService}
	if gate != nil {
		options.License = gate // the duck-typed service's is_licensed() is the 402 gate
	}
	if fake, found := seams.Faults[recoverFault]; found {
		options.RecoverStartup = recoverWithFault(fake)
	}
	if _, err := os.Stat(frontend); err != nil {
		options.FrontendDir = "" // Settings.frontend may point at a missing build
	}
	if seams.Token != nil {
		options.Token = *seams.Token
	}
	if seams.SecretKey != nil {
		options.SecretKey = *seams.SecretKey
	}
	if seams.AdminOrigin != nil {
		options.AdminOrigin = strings.TrimRight(*seams.AdminOrigin, "/")
	}
	return app.Compose(options)
}

// parityHandler is parity/inspect_api.InspectApp: parity headers are trusted only with the
// run's key and removed before the product handler sees the request.
type parityHandler struct {
	engine *app.Engine
	key    string
	stop   chan struct{}
}

func (h parityHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	trusted := h.key != "" && r.Header.Get("X-Parity-Key") == h.key
	remote := r.Header.Get("X-Parity-Remote-Addr")
	r.Header.Del("X-Parity-Key")
	r.Header.Del("X-Parity-Remote-Addr")
	if operation, inspect := strings.CutPrefix(r.URL.Path, parityPrefix); inspect {
		body, _ := io.ReadAll(r.Body)
		op, known := parityOps[r.Method+" "+operation]
		switch {
		case !trusted:
			reply(w, http.StatusForbidden, map[string]any{"error": "parity key required"})
		case r.Method == "POST" && operation == "shutdown":
			reply(w, http.StatusOK, map[string]any{"value": nil})
			go func() { close(h.stop) }()
		case known:
			runParityOp(w, h.engine, op, body, r)
		default:
			reply(w, http.StatusNotFound, map[string]any{"error": "unknown parity operation " + r.Method + " " + operation})
		}
		return
	}
	if remote != "" && trusted {
		r.RemoteAddr = remote
	}
	h.engine.Handler.ServeHTTP(w, r)
}

func reply(w http.ResponseWriter, status int, payload map[string]any) {
	data, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
