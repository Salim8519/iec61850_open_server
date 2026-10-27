package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/config"
)

// set changes settings without questions (used by the Windows deploy command).
// Secrets never appear on the command line: with --secrets-stdin they are read as
// "paypal_client_secret=..." / "mailgun_api_key=..." / "lemon_signing_secret=..." /
// "admin_gateway_secret=..." lines
// from standard input.
func set(args []string) error {
	text := map[string]*string{}
	var trialDays *int
	var secretsStdin *bool
	names := []string{"listen", "public-url", "price", "currency", "support-email", "paypal-mode",
		"paypal-client-id", "paypal-webhook-id", "mailgun-domain", "mailgun-region", "mailgun-from", "notify-email", "seller-name",
		"admin-path", "admin-origin", "marketing-from", "marketing-domain", "lemon-checkout-url", "lemon-variant-id", "lemon-store-id",
		"lemon-test-mode"}
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
	testMode := ""
	targets := map[string]*string{"listen": &cfg.ListenAddr, "public-url": &cfg.PublicURL, "price": &cfg.LicensePrice,
		"currency": &cfg.LicenseCurrency, "support-email": &cfg.SupportEmail, "paypal-mode": &cfg.PayPalMode,
		"paypal-client-id": &cfg.PayPalClientID, "paypal-webhook-id": &cfg.PayPalWebhookID,
		"mailgun-domain": &cfg.MailgunDomain, "mailgun-region": &cfg.MailgunRegion, "mailgun-from": &cfg.MailgunFrom,
		"notify-email": &cfg.NotifyEmail, "seller-name": &cfg.SellerName,
		"admin-path": &cfg.AdminPath, "admin-origin": &cfg.AdminOrigin,
		"marketing-from": &cfg.MarketingFrom, "marketing-domain": &cfg.MarketingDomain,
		"lemon-checkout-url": &cfg.LemonCheckoutURL, "lemon-variant-id": &cfg.LemonVariantID, "lemon-store-id": &cfg.LemonStoreID, "lemon-test-mode": &testMode}
	for name, value := range text {
		if v := strings.TrimSpace(*value); v != "" {
			*targets[name] = v
		}
	}
	switch testMode {
	case "on":
		cfg.LemonAcceptTestMode = true
	case "off":
		cfg.LemonAcceptTestMode = false
	case "":
	default:
		return errors.New("--lemon-test-mode must be on or off")
	}
	cfg.PublicURL = strings.TrimRight(cfg.PublicURL, "/")
	cfg.LicenseCurrency = strings.ToUpper(cfg.LicenseCurrency)
	if cfg.LicensePrice, err = config.NormalizePrice(cfg.LicensePrice); err != nil {
		return err
	}
	if cfg.PayPalMode != "sandbox" && cfg.PayPalMode != "live" {
		return errors.New("--paypal-mode must be sandbox or live")
	}
	if err := checkAdminPath(cfg.AdminPath); err != nil {
		return err
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
		"| Lemon Squeezy", ready[cfg.LemonReady()], "| PayPal", ready[cfg.PayPalReady()], "("+cfg.PayPalMode+") | email", ready[cfg.MailReady()])
	return nil
}

var adminPathPattern = regexp.MustCompile(`^/[a-z0-9][a-z0-9-]{2,39}$`)

// checkAdminPath keeps the owner page off the public routes.
func checkAdminPath(path string) error {
	if path == "" {
		return nil
	}
	if !adminPathPattern.MatchString(path) {
		return errors.New("--admin-path must look like /yourname (3-40 lowercase letters, digits or -)")
	}
	for _, taken := range []string{"/api", "/buy", "/activate", "/static", "/unsubscribe"} {
		if strings.HasPrefix(path, taken) {
			return fmt.Errorf("--admin-path cannot start with %s", taken)
		}
	}
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
		case "lemon_api_key":
			cfg.LemonAPIKey = strings.TrimSpace(value)
		case "lemon_signing_secret":
			cfg.LemonSigningSecret = strings.TrimSpace(value)
		case "admin_gateway_secret":
			cfg.AdminGatewaySecret = strings.TrimSpace(value)
		default:
			return fmt.Errorf("unknown secret %q", key)
		}
	}
	return scanner.Err()
}
