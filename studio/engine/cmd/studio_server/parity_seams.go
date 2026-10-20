//go:build parity

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"studio-engine/internal/licensing"
	"studio-engine/internal/updates"
)

// paritySeams is the seams file of tests/parity/server_seams.py (data seams only).
type paritySeams struct {
	Token       *string                    `json:"token"`
	SecretKey   *string                    `json:"secret_key"`
	AdminOrigin *string                    `json:"admin_origin"`
	ParityKey   string                     `json:"parity_key"`
	License     map[string]json.RawMessage `json:"license"`
	Updates     map[string]json.RawMessage `json:"updates"`
	Faults      map[string]json.RawMessage `json:"faults"`
}

type licenseSeam struct {
	Home          string            `json:"home"`
	DataDir       string            `json:"data_dir"`
	Enforce       bool              `json:"enforce"`
	HWID          *string           `json:"hwid"`
	Clock         *float64          `json:"clock"`
	Keys          map[string]string `json:"keys"`
	NativeFolder  *string           `json:"native_folder"`
	ExpectedFiles map[string]string `json:"expected_files"`
}

type updateSeam struct {
	Directory string            `json:"directory"`
	Server    *string           `json:"server"`
	Current   string            `json:"current"`
	Keys      map[string]string `json:"keys"`
	Clock     *float64          `json:"clock"`
	Opener    json.RawMessage   `json:"opener"`
}

func readSeams(path string) (paritySeams, error) {
	var seams paritySeams
	data, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(data, &seams)
	}
	if err == nil && len(seams.Faults) > 0 {
		err = fmt.Errorf("the Go parity build does not implement fault seams (%d given)", len(seams.Faults))
	}
	return seams, err
}

// decodeSeam re-reads a seam object into its typed form; callback ("object") seams are
// not implemented in Go yet.
func decodeSeam(raw map[string]json.RawMessage, into any) (bool, error) {
	if raw == nil {
		return false, nil
	}
	if _, isObject := raw["object"]; isObject {
		return false, fmt.Errorf("the Go parity build does not implement object seams")
	}
	data, _ := json.Marshal(raw)
	return true, json.Unmarshal(data, into)
}

func hexKeys(table map[string]string) (map[string][]byte, error) {
	if table == nil {
		return nil, nil
	}
	keys := map[string][]byte{}
	for kid, value := range table {
		key, err := hex.DecodeString(value)
		if err != nil {
			return nil, err
		}
		keys[kid] = key
	}
	return keys, nil
}

func licenseFromSeam(raw map[string]json.RawMessage) (*licensing.Service, error) {
	var seam licenseSeam
	if present, err := decodeSeam(raw, &seam); !present || err != nil {
		return nil, err
	}
	keys, err := hexKeys(seam.Keys)
	if err != nil {
		return nil, err
	}
	options := licensing.Options{Home: seam.Home, DataDir: seam.DataDir, Enforce: seam.Enforce, Keys: keys,
		ExpectedFiles: seam.ExpectedFiles}
	if seam.NativeFolder != nil {
		options.NativeFolder = *seam.NativeFolder
	}
	if seam.HWID != nil {
		hwid := *seam.HWID
		options.HWID = func() (string, error) { return hwid, nil }
	}
	if seam.Clock != nil {
		clock := *seam.Clock
		options.Clock = func() float64 { return clock }
	}
	return licensing.New(options), nil
}

func updatesFromSeam(raw map[string]json.RawMessage) (*updates.Service, error) {
	var seam updateSeam
	if present, err := decodeSeam(raw, &seam); !present || err != nil {
		return nil, err
	}
	if len(seam.Opener) > 0 && string(seam.Opener) != "null" {
		return nil, fmt.Errorf("the Go parity build does not implement the update opener callback")
	}
	keys, err := hexKeys(seam.Keys)
	if err != nil {
		return nil, err
	}
	server := ""
	if seam.Server != nil {
		server = *seam.Server
	}
	options := updates.Options{Directory: seam.Directory, Current: seam.Current, Keys: keys,
		ServerURL: func() (string, error) { return server, nil }}
	if seam.Clock != nil {
		clock := *seam.Clock
		options.Clock = func() float64 { return clock }
	}
	return updates.New(options), nil
}
