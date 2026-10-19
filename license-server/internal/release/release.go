// Package release holds the newest published ScadaBench version that the Windows app
// checks against. The owner sets it at each website deploy; the server answers with one
// precomputed signed body, so a check costs no signing, encoding or disk work.
package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Download is one published file and its SHA-256, so the app can check what it fetched.
type Download struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// Info is the release record. Field order is fixed, so the JSON bytes are stable.
type Info struct {
	Version    string    `json:"version"`
	Date       string    `json:"date"`
	MinVersion string    `json:"min_version,omitempty"`
	Notes      string    `json:"notes"`
	Page       string    `json:"page"`
	Installer  Download  `json:"installer"`
	Portable   *Download `json:"portable,omitempty"`
}

var (
	versionPattern = regexp.MustCompile(`^[0-9]{1,4}\.[0-9]{1,4}\.[0-9]{1,4}$`)
	datePattern    = regexp.MustCompile(`^20[0-9]{2}-[01][0-9]-[0-3][0-9]$`)
	hexPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	// Only the product's own sites may be named, so a leaked admin step cannot point
	// every app at another download host.
	allowedHosts = map[string]bool{"scadabench.com": true, "www.scadabench.com": true, "download.scadabench.com": true}
)

const maxNotes = 600

// FileName is the release record inside the data directory.
const FileName = "release.json"

func checkURL(name, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || !allowedHosts[parsed.Host] || parsed.User != nil {
		return fmt.Errorf("%s must be an https link on scadabench.com", name)
	}
	return nil
}

// Validate refuses anything the app could misread or that points outside the product's sites.
func (i *Info) Validate() error {
	i.Notes = strings.TrimSpace(i.Notes)
	switch {
	case !versionPattern.MatchString(i.Version):
		return errors.New("version must look like 0.1.14")
	case !datePattern.MatchString(i.Date):
		return errors.New("date must look like 2026-10-19")
	case i.MinVersion != "" && !versionPattern.MatchString(i.MinVersion):
		return errors.New("min_version must look like 0.1.14")
	case utf8.RuneCountInString(i.Notes) > maxNotes:
		return fmt.Errorf("notes must be at most %d characters", maxNotes)
	case !hexPattern.MatchString(i.Installer.SHA256):
		return errors.New("installer sha256 must be 64 lowercase hex characters")
	case i.Portable != nil && !hexPattern.MatchString(i.Portable.SHA256):
		return errors.New("portable sha256 must be 64 lowercase hex characters")
	}
	if err := checkURL("page", i.Page); err != nil {
		return err
	}
	if err := checkURL("installer url", i.Installer.URL); err != nil {
		return err
	}
	if i.Portable != nil {
		return checkURL("portable url", i.Portable.URL)
	}
	return nil
}

// Parse reads and validates one release record (from the owner's deploy step).
func Parse(raw []byte) (Info, error) {
	var info Info
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&info); err != nil {
		return Info{}, errors.New("release record must be JSON with version, date, notes, page and installer")
	}
	return info, info.Validate()
}

// Save writes the record atomically (temporary file, then rename).
func Save(dataDir string, info Info) error {
	if err := info.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dataDir, FileName)
	if err := os.WriteFile(path+".tmp", append(raw, '\n'), 0o640); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}

// Load returns the saved record; ok is false when none was published yet.
func Load(dataDir string) (Info, bool, error) {
	raw, err := os.ReadFile(filepath.Join(dataDir, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return Info{}, false, nil
	}
	if err != nil {
		return Info{}, false, err
	}
	info, err := Parse(raw)
	return info, err == nil, err
}

// Signed is the ready-to-send answer: the exact payload string, its signature and a
// strong ETag, built once at start so every request just copies bytes.
type Signed struct {
	Body []byte
	ETag string
}

// Sign builds the answer with sign(payload) from the server's key (certs.SignRelease).
func Sign(info Info, keyID string, sign func([]byte) string) (Signed, error) {
	payload, err := json.Marshal(info)
	if err != nil {
		return Signed{}, err
	}
	body, err := json.Marshal(map[string]string{"payload": string(payload), "signature": sign(payload), "kid": keyID})
	if err != nil {
		return Signed{}, err
	}
	digest := sha256.Sum256(body)
	return Signed{Body: body, ETag: `"` + hex.EncodeToString(digest[:12]) + `"`}, nil
}
