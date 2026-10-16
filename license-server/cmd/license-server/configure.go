package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/config"
)

type prompter struct{ in *bufio.Reader }

func (p prompter) text(question, current string, secret bool) string {
	shown := current
	if secret && current != "" {
		shown = "keep current"
	}
	fmt.Printf("%s [%s]: ", question, shown)
	line, _ := p.in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return current
	}
	if line == "-" {
		return ""
	}
	return line
}

func (p prompter) number(question string, current int) int {
	for {
		answer := p.text(question, strconv.Itoa(current), false)
		value, err := strconv.Atoi(answer)
		if err == nil && value >= 0 {
			return value
		}
		fmt.Println("  Please type a whole number.")
	}
}

func (p prompter) choice(question, current string, options ...string) string {
	for {
		answer := strings.ToLower(p.text(question+" ("+strings.Join(options, "/")+")", current, false))
		for _, option := range options {
			if answer == option {
				return option
			}
		}
		fmt.Println("  Please type one of:", strings.Join(options, ", "))
	}
}

// configure asks only what a seller must decide; Enter keeps every current value.
func configure(args []string) error {
	dataDir, err := flags("configure", args, nil)
	if err != nil {
		return err
	}
	cfg, err := config.LoadFile(dataDir)
	if err != nil {
		return err
	}
	p := prompter{bufio.NewReader(os.Stdin)}
	fmt.Println("ScadaBench license server setup. Press Enter to keep the value in [brackets].")
	fmt.Println("Type - to clear a value.")
	fmt.Println()
	fmt.Println("== Website ==")
	cfg.PublicURL = strings.TrimRight(p.text("Public address customers use (https://license.example.com)", cfg.PublicURL, false), "/")
	cfg.ListenAddr = p.text("Local listen address (Cloudflare Tunnel points here)", cfg.ListenAddr, false)
	cfg.SupportEmail = p.text("Support email shown to customers", cfg.SupportEmail, false)
	fmt.Println("== Product ==")
	cfg.ProductName = p.text("Product name", cfg.ProductName, false)
	cfg.LicensePrice = p.text("Price (for example 149.00)", cfg.LicensePrice, false)
	if normalized, err := config.NormalizePrice(cfg.LicensePrice); err == nil {
		cfg.LicensePrice = normalized
	}
	cfg.LicenseCurrency = strings.ToUpper(p.text("Currency (USD, EUR, ...)", cfg.LicenseCurrency, false))
	cfg.TrialDays = p.number("Free trial days", cfg.TrialDays)
	cfg.MaxTransfersPerYear = p.number("Self-service computer moves per key per year (0 = only you can reset)", cfg.MaxTransfersPerYear)
	fmt.Println("== PayPal (developer.paypal.com > Apps & Credentials) ==")
	cfg.PayPalMode = p.choice("Mode", cfg.PayPalMode, "sandbox", "live")
	cfg.PayPalClientID = p.text("PayPal client ID", cfg.PayPalClientID, false)
	cfg.PayPalClientSecret = p.text("PayPal secret", cfg.PayPalClientSecret, true)
	fmt.Printf("  Webhook URL to add in PayPal: %s/api/v1/webhooks/paypal\n", cfg.PublicURL)
	fmt.Println("  Events: Checkout order approved, Payment capture completed/refunded/reversed.")
	cfg.PayPalWebhookID = p.text("PayPal webhook ID", cfg.PayPalWebhookID, false)
	fmt.Println("== Mailgun (optional; keys are also shown on the purchase page) ==")
	cfg.MailgunDomain = p.text("Mailgun sending domain (mg.example.com)", cfg.MailgunDomain, false)
	cfg.MailgunAPIKey = p.text("Mailgun API key", cfg.MailgunAPIKey, true)
	cfg.MailgunRegion = p.choice("Mailgun region", cfg.MailgunRegion, "us", "eu")
	cfg.MailgunFrom = p.text("From address (ScadaBench <licenses@example.com>)", cfg.MailgunFrom, false)
	cfg.EnsureSecrets()
	if err := cfg.Save(); err != nil {
		return err
	}
	signer, err := certs.Load(dataDir)
	if err != nil {
		if signer, err = certs.Generate(dataDir, false); err != nil {
			return err
		}
		fmt.Println("\nCreated a new signing key. BACK IT UP: run `license-server backup --out ~/license-backup.tar.gz`.")
	}
	fmt.Println("\nSaved", config.Path(dataDir))
	printWindowsSync(cfg, signer)
	return nil
}

func printWindowsSync(cfg config.Config, signer *certs.Signer) {
	fmt.Println("\nOn your Windows build PC, run this once in the ScadaBench (studio) repository:")
	fmt.Printf("  python studio\\tools\\configure_release.py --server-url %s --public-key %s\n",
		cfg.PublicURL, signer.PublicHex())
}

func status(args []string) error {
	dataDir, err := flags("status", args, nil)
	if err != nil {
		return err
	}
	cfg, err := config.Load(dataDir)
	if err != nil {
		return err
	}
	signer, err := certs.Load(dataDir)
	if err != nil {
		return err
	}
	ready := map[bool]string{true: "ready", false: "NOT SET UP"}
	fmt.Println("Data directory: ", dataDir)
	fmt.Println("Public address: ", cfg.PublicURL)
	fmt.Println("Listen address: ", cfg.ListenAddr)
	fmt.Println("Price:          ", cfg.LicensePrice, cfg.LicenseCurrency, "| trial days:", cfg.TrialDays)
	fmt.Println("PayPal checkout:", ready[cfg.PayPalReady()], "("+cfg.PayPalMode+")")
	fmt.Println("Email delivery: ", ready[cfg.MailReady()])
	fmt.Println("Sale alerts to: ", dash(cfg.NotifyEmail))
	printUsage(cfg)
	fmt.Println("Key id:         ", signer.KeyID)
	printWindowsSync(cfg, signer)
	return nil
}
