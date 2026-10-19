// Command studio_server is the Go engine's backend, started like the Python server mode
// (launcher/entrypoint.py): same arguments, data-folder lease, launcher.json and printed
// lines, beginning with the readiness line "ScadaBench: http://127.0.0.1:<port>".
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"studio-engine/internal/app"
	"studio-engine/internal/launcher"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) > 0 && args[0] == "--network-helper" {
		fmt.Fprintln(os.Stderr, "The network helper is not part of the Go engine yet; use the Python engine.")
		return 1
	}
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
	if options.SelfTest {
		fmt.Fprintln(os.Stderr, "--self-test is not part of the Go engine yet; use the Python engine.")
		return 1
	}
	paths := resolvePaths(options)
	lease, err := launcher.AcquireLease(paths.data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer lease.Release()
	return serve(options, paths)
}

func serve(options launcher.Options, paths paths) int {
	token, secret, origin := launcher.LaunchToken(), launcher.SecretKey(), ""
	if options.Dev {
		token, secret, origin = options.DevToken, launcher.DevSecret, trimSlash(options.FrontendURL)
	}
	engine, err := app.New(app.Config{DataDir: paths.data, FrontendDir: paths.frontend, Token: token,
		SecretKey: secret, AdminOrigin: origin})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:"+strconv.Itoa(options.Port))
	if err != nil && options.Dev {
		fmt.Fprintf(os.Stderr, "Port %d is busy. Stop the other ScadaBench backend before starting dev mode.\n", options.Port)
		return 1
	}
	if err != nil {
		if listener, err = net.Listen("tcp4", "127.0.0.1:0"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	url := "http://127.0.0.1:" + strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	if err := os.MkdirAll(paths.data, 0o755); err == nil {
		_ = launcher.WriteRecord(paths.data, url, 0)
	}
	fmt.Println("ScadaBench: " + url)
	if options.Dev {
		if !frontendReady(options.FrontendURL, 15*time.Second) {
			fmt.Printf("Vite is not reachable at %s; start it and open the printed address.\n", options.FrontendURL)
		}
		fmt.Printf("Vite frontend: %s/#session=%s\n", options.FrontendURL, token)
		fmt.Println("Open the address above in your browser (ScadaBench never opens tabs automatically).")
	} else {
		fmt.Printf("Open in your browser: %s#session=%s\n", url, token)
	}
	server := &http.Server{Handler: engine.Handler, ReadHeaderTimeout: 30 * time.Second}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func trimSlash(text string) string {
	for len(text) > 0 && text[len(text)-1] == '/' {
		text = text[:len(text)-1]
	}
	return text
}
