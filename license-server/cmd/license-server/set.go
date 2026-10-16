package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/config"
)

// set changes settings without questions (used by the Windows deploy command).
// Secrets never appear on the command line: with --secrets-stdin they are read as
// "paypal_client_secret=..." / "mailgun_api_key=..." lines from standard input.
func set(args []string) error {
	text := map[string]*string{}
	var trialDays *int
	var secretsStdin *bool
	names := []string{"listen", "public-url", "price", "currency", "support-email", "paypal-mode",
		"paypal-client-id", "paypal-webhook-id", "mailgun-domain", "mailgun-region", "mailgun-from", "notify-email", "seller-name"}
	dataDir, err := flags("set", args, func(fs *flag.FlagSet) {
		for _, name := range names {
			text[name] = fs.String(name, "", name)
		}
		trialDays = fs.Int("trial-days", 0, "free trial length in days")
		secretsStdin = fs.Bool("secrets-stdin", false, "read secrets from standard input")
	})
	if err != nil {
		return err
	}
	cfg, err := config.LoadFile(dataDir)
	if err != nil {
		return err
	}
	targets := map[string]*string{"listen": &cfg.ListenAddr, "public-url": &cfg.PublicURL, "price": &cfg.LicensePrice,
		"currency": &cfg.LicenseCurrency, "support-email": &cfg.SupportEmail, "paypal-mode": &cfg.PayPalMode,
		"paypal-client-id": &cfg.PayPalClientID, "paypal-webhook-id": &cfg.PayPalWebhookID,
		"mailgun-domain": &cfg.MailgunDomain, "mailgun-region": &cfg.MailgunRegion, "mailgun-from": &cfg.MailgunFrom,
		"notify-email": &cfg.NotifyEmail, "seller-name": &cfg.SellerName}
	for name, value := range text {
		if v := strings.TrimSpace(*value); v != "" {
			*targets[name] = v
		}
	}
	cfg.PublicURL = strings.TrimRight(cfg.PublicURL, "/")
	cfg.LicenseCurrency = strings.ToUpper(cfg.LicenseCurrency)
	if cfg.LicensePrice, err = config.NormalizePrice(cfg.LicensePrice); err != nil {
		return err
	}
	if cfg.PayPalMode != "sandbox" && cfg.PayPalMode != "live" {
		return errors.New("--paypal-mode must be sandbox or live")
	}
	if *trialDays > 0 {
		cfg.TrialDays = *trialDays
	}
	if *secretsStdin {
		if err := readSecrets(&cfg); err != nil {
			return err
		}
	}
	if err := cfg.Save(); err != nil {
		return err
	}
	if _, err := certs.Load(dataDir); err != nil {
		if _, err := certs.Generate(dataDir, false); err != nil {
			return err
		}
	}
	ready := map[bool]string{true: "ready", false: "not set up"}
	fmt.Println("Saved: listen", cfg.ListenAddr, "| public", cfg.PublicURL, "| price", cfg.LicensePrice, cfg.LicenseCurrency,
		"| PayPal", ready[cfg.PayPalReady()], "("+cfg.PayPalMode+") | email", ready[cfg.MailReady()])
	return nil
}

func readSecrets(cfg *config.Config) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), string(rune(0xFEFF))))
		key, value, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(value) == "" {
			continue
		}
		switch strings.TrimSpace(key) {
		case "paypal_client_secret":
			cfg.PayPalClientSecret = strings.TrimSpace(value)
		case "mailgun_api_key":
			cfg.MailgunAPIKey = strings.TrimSpace(value)
		default:
			return fmt.Errorf("unknown secret %q", key)
		}
	}
	return scanner.Err()
}
