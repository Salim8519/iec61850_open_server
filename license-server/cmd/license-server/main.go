// Command license-server runs and administers the ScadaBench license service.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"substation-license-server/internal/api"
	"substation-license-server/internal/certs"
	"substation-license-server/internal/config"
	"substation-license-server/internal/logging"
	"substation-license-server/internal/mail"
	"substation-license-server/internal/paypal"
	"substation-license-server/internal/store"
)

const usage = `ScadaBench license server

Usage: license-server <command> [options]

Everyday commands:
  serve                     run the server (systemd does this for you)
  configure                 answer a few questions; Enter keeps the [current] value
  set --listen A --public-url U --price P ...   change settings without questions
  status                    show settings summary and the public key for the Windows app
  issue --email E [--type lifetime|trial] [--days 7] [--hwid H]
                            create a key; with --hwid also prints an offline license.cert
  revoke --key K [--reason R]
  reset-hwid --key K        let a key be activated on a new computer
  reset-trial --hwid H      let this Machine ID start a free trial again (testing)
  delete --key K            remove a key completely (testing; use revoke for customers)
  certificate --key K       print license.cert for an activated key (email it to the customer)
  list                      list keys and the number of trial machines
  payments [--json]         every recorded payment, refund and reversal, with buyer details
  release < release.json    publish the newest app version the Windows app checks (JSON on stdin)
  admin-setup --password-stdin   create or replace the owner page login (password on stdin);
                            prints the authenticator key and recovery codes once
  admin-status              say whether the owner login exists (no secrets shown)
  backup --out FILE.tar.gz  save database, settings and signing key (keep it offline!)
  keygen [--force]          create the Ed25519 signing key (configure does this once)

All commands accept --data DIR (default: $LICENSE_DATA_DIR, /var/lib/substation-license, or ./data).
`

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
		fmt.Print(usage)
		return
	}
	command, args := os.Args[1], os.Args[2:]
	var err error
	switch command {
	case "serve":
		err = serve(args)
	case "configure":
		err = configure(args)
	case "set":
		err = set(args)
	case "status":
		err = status(args)
	case "keygen":
		err = keygen(args)
	case "backup":
		err = backup(args)
	case "payments":
		err = payments(args)
	case "release":
		err = publishRelease(args)
	case "admin-setup":
		err = adminSetup(args)
	case "admin-status":
		err = adminStatus(args)
	case "issue", "revoke", "reset-hwid", "reset-trial", "delete", "list", "certificate":
		err = adminCommand(command, args)
	default:
		err = fmt.Errorf("unknown command %q; run `license-server help`", command)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func defaultDataDir() string {
	if dir := os.Getenv("LICENSE_DATA_DIR"); dir != "" {
		return dir
	}
	if info, err := os.Stat("/var/lib/substation-license"); err == nil && info.IsDir() {
		return "/var/lib/substation-license"
	}
	return "data"
}

func flags(name string, args []string, define func(*flag.FlagSet)) (string, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	dataDir := set.String("data", defaultDataDir(), "data directory")
	if define != nil {
		define(set)
	}
	if err := set.Parse(args); err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(*dataDir)
	return absolute, err
}

func dbPath(dataDir string) string { return filepath.Join(dataDir, "licenses.db") }

func buildServer(cfg config.Config, db *store.Store) (*api.Server, *logging.Capped, error) {
	signer, err := certs.Load(cfg.DataDir)
	if err != nil {
		return nil, nil, err
	}
	logger, err := logging.Open(cfg.LogDir, "server.log", cfg.LogMaxMegabytes, cfg.LogMaxFiles)
	if err != nil {
		return nil, nil, err
	}
	server := api.New(cfg, db, signer, logger)
	if err := server.LoadRelease(); err != nil {
		logger.Event("release_unreadable", map[string]any{"error": err.Error()})
	}
	if cfg.PayPalReady() {
		server.Payments = paypal.New(cfg.PayPalBase(), cfg.PayPalClientID, cfg.PayPalClientSecret)
	}
	if cfg.MailReady() {
		server.Mailer = mail.New(cfg.MailgunDomain, cfg.MailgunAPIKey, cfg.MailgunFrom, cfg.MailgunRegion)
		// News may use its own sending domain so bulk mail never hurts license emails.
		domain, from := cfg.MailgunDomain, cfg.MailgunFrom
		if cfg.MarketingDomain != "" {
			domain = cfg.MarketingDomain
		}
		if cfg.MarketingFrom != "" {
			from = cfg.MarketingFrom
		}
		server.News = mail.New(domain, cfg.MailgunAPIKey, from, cfg.MailgunRegion)
	}
	return server, logger, nil
}

func serve(args []string) error {
	dataDir, err := flags("serve", args, nil)
	if err != nil {
		return err
	}
	if err := config.Init(dataDir); err != nil {
		return err
	}
	cfg, err := config.Load(dataDir)
	if err != nil {
		return err
	}
	db, err := store.Open(dbPath(dataDir))
	if errors.Is(err, store.ErrDatabaseBusy) {
		return errors.New("another license-server is already running on this data directory")
	}
	if err != nil {
		return err
	}
	defer db.Close()
	server, logger, err := buildServer(cfg, db)
	if err != nil {
		return err
	}
	defer logger.Close()
	httpServer := &http.Server{Addr: cfg.ListenAddr, Handler: server.Handler(),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)
	}()
	logger.Event("start", map[string]any{"addr": cfg.ListenAddr, "kid": server.Signer.KeyID,
		"checkout": cfg.PayPalReady(), "mail": cfg.MailReady()})
	fmt.Printf("License server listening on http://%s (key id %s)\n", cfg.ListenAddr, server.Signer.KeyID)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func keygen(args []string) error {
	var force *bool
	dataDir, err := flags("keygen", args, func(set *flag.FlagSet) { force = set.Bool("force", false, "replace key") })
	if err != nil {
		return err
	}
	signer, err := certs.Generate(dataDir, *force)
	if err != nil {
		return err
	}
	fmt.Println("New signing key created:", certs.KeyPath(dataDir))
	fmt.Println("Public key (put this in the Windows app):", signer.PublicHex())
	return nil
}
