package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// licenseFiles are the files the Studio app keeps in its license folder.
var licenseFiles = []string{"license.cert", "license_state.json", ".time_anchor"}

// licenseHome is the installed app's folder, same as service.default_home().
func licenseHome() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, "AppData", "Local")
	}
	return filepath.Join(base, "ScadaBench")
}

func readPayload(folder string) (*Payload, error) {
	return readCertificate(filepath.Join(folder, "license.cert"))
}

// readCertificate reads the payload of any license.cert-format file.
func readCertificate(path string) (*Payload, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bundle struct {
		Payload string `json:"payload"`
	}
	var payload Payload
	if json.Unmarshal(raw, &bundle) != nil || json.Unmarshal([]byte(bundle.Payload), &payload) != nil {
		return nil, errors.New("the license file on this PC is damaged")
	}
	return &payload, nil
}

// moveAside moves the license files into removed-<time>; nothing is deleted.
// It returns the folder used, or "" when there was nothing to move.
func moveAside(folder string, names []string, now time.Time) (string, error) {
	var present []string
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(folder, name)); err == nil {
			present = append(present, name)
		}
	}
	if len(present) == 0 {
		return "", nil
	}
	target := filepath.Join(folder, "removed-"+now.Format("20060102-150405"))
	for suffix := 2; ; suffix++ {
		if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
			break
		}
		target = filepath.Join(folder, "removed-"+now.Format("20060102-150405")+"-"+strconv.Itoa(suffix))
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		return "", err
	}
	for _, name := range present {
		if err := os.Rename(filepath.Join(folder, name), filepath.Join(target, name)); err != nil {
			return target, err
		}
	}
	return target, nil
}

// installCertificate writes license.cert the way the Studio app saves it. An old
// certificate and state file are moved aside first; the clock anchor is kept.
func installCertificate(folder string, certificate any, now time.Time) (string, error) {
	raw, err := json.Marshal(certificate)
	if err != nil {
		return "", err
	}
	var bundle struct {
		Payload   string `json:"payload"`
		Signature string `json:"signature"`
	}
	if json.Unmarshal(raw, &bundle) != nil || bundle.Payload == "" || bundle.Signature == "" {
		return "", errors.New("the server did not return a license file")
	}
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return "", err
	}
	moved, err := moveAside(folder, []string{"license.cert", "license_state.json"}, now)
	if err != nil {
		return moved, err
	}
	text, _ := json.MarshalIndent(bundle, "", "  ")
	temporary := filepath.Join(folder, "license.cert.tmp")
	if err := os.WriteFile(temporary, text, 0o600); err != nil {
		return moved, err
	}
	return moved, os.Rename(temporary, filepath.Join(folder, "license.cert"))
}
