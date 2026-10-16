package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"substation-license-server/internal/api"
	"substation-license-server/internal/config"
	"substation-license-server/internal/store"
)

// adminCommand works with the server stopped (direct database access) or
// running (it then calls the local admin API with the admin token).
func adminCommand(command string, args []string) error {
	var request api.AdminRequest
	dataDir, err := flags(command, args, func(set *flag.FlagSet) {
		set.StringVar(&request.Email, "email", "", "customer email")
		set.StringVar(&request.Type, "type", "lifetime", "lifetime or trial")
		set.IntVar(&request.Days, "days", 0, "trial key length in days")
		set.StringVar(&request.HWID, "hwid", "", "bind to this Machine ID now (offline delivery)")
		set.StringVar(&request.Key, "key", "", "license key")
		set.StringVar(&request.Reason, "reason", "", "revocation reason")
	})
	if err != nil {
		return err
	}
	needsKey := command == "revoke" || command == "reset-hwid" || command == "certificate" || command == "delete"
	if needsKey && request.Key == "" {
		return errors.New("--key is required")
	}
	cfg, err := config.Load(dataDir)
	if err != nil {
		return err
	}
	result, err := runLocal(cfg, command, request)
	if errors.Is(err, store.ErrDatabaseBusy) {
		result, err = runRemote(cfg, command, request)
	}
	if err != nil {
		return err
	}
	pretty, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(pretty))
	return nil
}

func runLocal(cfg config.Config, command string, request api.AdminRequest) (any, error) {
	db, err := store.Open(dbPath(cfg.DataDir))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	server, logger, err := buildServer(cfg, db)
	if err != nil {
		return nil, err
	}
	defer logger.Close()
	return server.RunAdmin(command, request)
}

func callAdmin(cfg config.Config, command string, request api.AdminRequest) (*http.Response, error) {
	body, _ := json.Marshal(request)
	address := cfg.ListenAddr
	if strings.HasPrefix(address, ":") || strings.HasPrefix(address, "0.0.0.0:") {
		address = "127.0.0.1:" + address[strings.LastIndex(address, ":")+1:]
	}
	httpRequest, err := http.NewRequest(http.MethodPost, "http://"+address+"/api/v1/admin/"+command, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+cfg.AdminToken)
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("server holds the database but is not reachable: %w", err)
	}
	return response, nil
}

func remoteBackup(dataDir string, target io.Writer) error {
	cfg, err := config.Load(dataDir)
	if err != nil {
		return err
	}
	response, err := callAdmin(cfg, "backup", api.AdminRequest{})
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server refused the backup (%d)", response.StatusCode)
	}
	_, err = io.Copy(target, response.Body)
	return err
}

func runRemote(cfg config.Config, command string, request api.AdminRequest) (any, error) {
	response, err := callAdmin(cfg, command, request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var result any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, err
	}
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("server refused: %v", result)
	}
	return result, nil
}

// backup writes database (hot copy), config.json and signing.key into one tar.gz.
func backup(args []string) error {
	var out *string
	dataDir, err := flags("backup", args, func(set *flag.FlagSet) {
		out = set.String("out", "license-backup-"+time.Now().Format("20060102")+".tar.gz", "output file")
	})
	if err != nil {
		return err
	}
	var database bytes.Buffer
	db, err := store.Open(dbPath(dataDir))
	if err == nil {
		err = db.Backup(&database)
		db.Close()
	} else if errors.Is(err, store.ErrDatabaseBusy) {
		err = remoteBackup(dataDir, &database)
	}
	if err != nil {
		return err
	}
	file, err := os.OpenFile(*out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	zipped := gzip.NewWriter(file)
	archive := tar.NewWriter(zipped)
	entries := map[string][]byte{"licenses.db": database.Bytes()}
	for _, name := range []string{"config.json", "signing.key"} {
		if raw, err := os.ReadFile(filepath.Join(dataDir, name)); err == nil {
			entries[name] = raw
		}
	}
	for name, raw := range entries {
		header := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(raw)), ModTime: time.Now()}
		if err := archive.WriteHeader(header); err != nil {
			return err
		}
		if _, err := io.Copy(archive, bytes.NewReader(raw)); err != nil {
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	if err := zipped.Close(); err != nil {
		return err
	}
	fmt.Println("Backup written:", *out, "- copy it off this server and keep it private.")
	return nil
}
