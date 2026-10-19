// Package config loads and saves the server's single JSON configuration file.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds every operator setting. Secrets never leave this file except
// through the admin's own terminal.
type Config struct {
	ListenAddr      string `json:"listen_addr"`
	PublicURL       string `json:"public_url"`
	TrustCloudflare bool   `json:"trust_cloudflare"`
	DataDir         string `json:"data_dir"`

	TrialDays           int    `json:"trial_days"`
	TrialIPEmailAfter   int    `json:"trial_ip_email_after"`
	TrialIPWindowDays   int    `json:"trial_ip_window_days"`
	LicensePrice        string `json:"license_price"`
	LicenseCurrency     string `json:"license_currency"`
	ProductName         string `json:"product_name"`
	MaxTransfersPerYear int    `json:"max_transfers_per_year"`
	SupportEmail        string `json:"support_email"`
	NotifyEmail         string `json:"notify_email"`
	SellerName          string `json:"seller_name"`

	PayPalMode         string `json:"paypal_mode"`
	PayPalClientID     string `json:"paypal_client_id"`
	PayPalClientSecret string `json:"paypal_client_secret"`
	PayPalWebhookID    string `json:"paypal_webhook_id"`

	MailgunDomain string `json:"mailgun_domain"`
	MailgunAPIKey string `json:"mailgun_api_key"`
	MailgunRegion string `json:"mailgun_region"`
	MailgunFrom   string `json:"mailgun_from"`

	AdminToken    string `json:"admin_token"`
	PowSecret     string `json:"pow_secret"`
	PowDifficulty int    `json:"pow_difficulty"`
	RateGeneral   int    `json:"rate_general_per_minute"`
	RateSensitive int    `json:"rate_sensitive_per_minute"`

	LogDir          string `json:"log_dir"`
	LogMaxMegabytes int    `json:"log_max_megabytes"`
	LogMaxFiles     int    `json:"log_max_files"`
	MaxDataMB       int    `json:"max_data_megabytes"`
}

// Defaults returns a safe starting configuration rooted at dataDir.
func Defaults(dataDir string) Config {
	return Config{
		ListenAddr: "127.0.0.1:8090", PublicURL: "http://127.0.0.1:8090", TrustCloudflare: true,
		DataDir: dataDir, TrialDays: 7, TrialIPEmailAfter: 3, TrialIPWindowDays: 30, LicensePrice: "149.00", LicenseCurrency: "USD",
		ProductName: "ScadaBench Lifetime License", MaxTransfersPerYear: 2,
		PayPalMode: "sandbox", MailgunRegion: "us", PowDifficulty: 18, RateGeneral: 10, RateSensitive: 5,
		LogDir: filepath.Join(dataDir, "logs"), LogMaxMegabytes: 5, LogMaxFiles: 3, MaxDataMB: 45,
	}
}

// Path returns the config file location inside dataDir.
func Path(dataDir string) string { return filepath.Join(dataDir, "config.json") }

// LoadFile reads config.json (or defaults) without environment overrides; use it
// when the result is saved back, so overrides never leak into the file.
func LoadFile(dataDir string) (Config, error) {
	cfg := Defaults(dataDir)
	raw, err := os.ReadFile(Path(dataDir))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return cfg, err
	}
	if err == nil {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return cfg, errors.New("config.json is not valid JSON: " + err.Error())
		}
	}
	cfg.DataDir = dataDir
	cfg.EnsureSecrets()
	return cfg, nil
}

// Load returns the effective configuration: config.json plus LICENSE_* overrides.
func Load(dataDir string) (Config, error) {
	cfg, err := LoadFile(dataDir)
	if err == nil {
		applyEnv(&cfg)
	}
	return cfg, err
}

// Init makes sure config.json exists with generated secrets.
func Init(dataDir string) error {
	cfg, err := LoadFile(dataDir)
	if err != nil {
		return err
	}
	return cfg.Save()
}

// EnsureSecrets fills any missing random secret so a fresh install is never weak.
func (c *Config) EnsureSecrets() {
	if c.AdminToken == "" {
		c.AdminToken = RandomHex(32)
	}
	if c.PowSecret == "" {
		c.PowSecret = RandomHex(32)
	}
}

// Save writes config.json readable only by the service user.
func (c Config) Save() error {
	if err := os.MkdirAll(c.DataDir, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(c.DataDir), append(raw, '\n'), 0o600)
}

// PayPalBase returns the REST endpoint for the configured PayPal mode.
func (c Config) PayPalBase() string {
	if c.PayPalMode == "live" {
		return "https://api-m.paypal.com"
	}
	return "https://api-m.sandbox.paypal.com"
}

// PayPalReady reports whether checkout can be offered.
func (c Config) PayPalReady() bool {
	return c.PayPalClientID != "" && c.PayPalClientSecret != ""
}

// MailReady reports whether license emails can be sent.
func (c Config) MailReady() bool {
	return c.MailgunDomain != "" && c.MailgunAPIKey != "" && c.MailgunFrom != ""
}

// TrialIPWindow is the period in which new trials from one IP are counted; more than
// TrialIPEmailAfter of them ask the next trial for an email (never a refusal).
func (c Config) TrialIPWindow() time.Duration {
	days := c.TrialIPWindowDays
	if days <= 0 {
		days = 30
	}
	return time.Duration(days) * 24 * time.Hour
}

// RandomHex returns n random bytes as lowercase hex.
func RandomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic("system random source failed: " + err.Error())
	}
	return hex.EncodeToString(buf)
}

func applyEnv(c *Config) {
	text := map[string]*string{
		"LICENSE_LISTEN_ADDR": &c.ListenAddr, "LICENSE_PUBLIC_URL": &c.PublicURL,
		"LICENSE_PRICE": &c.LicensePrice, "LICENSE_CURRENCY": &c.LicenseCurrency,
		"PAYPAL_MODE": &c.PayPalMode, "PAYPAL_CLIENT_ID": &c.PayPalClientID,
		"PAYPAL_CLIENT_SECRET": &c.PayPalClientSecret, "PAYPAL_WEBHOOK_ID": &c.PayPalWebhookID,
		"MAILGUN_DOMAIN": &c.MailgunDomain, "MAILGUN_API_KEY": &c.MailgunAPIKey,
		"MAILGUN_REGION": &c.MailgunRegion, "MAILGUN_FROM": &c.MailgunFrom,
		"LICENSE_ADMIN_TOKEN": &c.AdminToken,
	}
	for name, target := range text {
		if value, ok := os.LookupEnv(name); ok {
			*target = strings.TrimSpace(value)
		}
	}
	numbers := map[string]*int{"LICENSE_TRIAL_DAYS": &c.TrialDays, "LICENSE_POW_DIFFICULTY": &c.PowDifficulty,
		"LICENSE_RATE_GENERAL": &c.RateGeneral, "LICENSE_RATE_SENSITIVE": &c.RateSensitive,
		"LICENSE_TRIAL_IP_EMAIL_AFTER": &c.TrialIPEmailAfter, "LICENSE_TRIAL_IP_WINDOW_DAYS": &c.TrialIPWindowDays}
	for name, target := range numbers {
		if value, err := strconv.Atoi(os.Getenv(name)); err == nil {
			*target = value
		}
	}
}
