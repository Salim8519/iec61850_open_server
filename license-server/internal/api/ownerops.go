package api

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// problemKinds are the server log events the Issues view shows by default.
var problemKinds = map[string]bool{"internal_error": true, "paypal_error": true, "mail_error": true,
	"owner_mail_error": true, "webhook_rejected": true, "webhook_error": true, "paypal_amount_mismatch": true,
	"template_error": true, "backup_error": true, "data_budget_reached": true, "release_unreadable": true,
	"sales_record_error": true, "audit_error": true, "subscribe_error": true}

// readLogEvents returns every event in the rotated server logs, oldest first.
func readLogEvents(dir string) []map[string]any {
	names, _ := filepath.Glob(filepath.Join(dir, "server.log*"))
	sort.Slice(names, func(i, j int) bool { return logAge(names[i]) > logAge(names[j]) })
	events := []map[string]any{}
	for _, name := range names {
		file, err := os.Open(name)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64<<10), 1<<20)
		for scanner.Scan() {
			var event map[string]any
			if json.Unmarshal(scanner.Bytes(), &event) == nil {
				events = append(events, event)
			}
		}
		file.Close()
	}
	return events
}

// logAge orders server.log.2 (oldest) before server.log.1 before server.log.
func logAge(name string) int {
	suffix := strings.TrimPrefix(filepath.Ext(name), ".")
	age := 0
	for _, digit := range suffix {
		if digit < '0' || digit > '9' {
			return 0
		}
		age = age*10 + int(digit-'0')
	}
	return age
}

func (s *Server) ownerIssues(r *http.Request, _ *ownerCall) (any, error) {
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	everything := r.URL.Query().Get("all") == "1"
	events := readLogEvents(s.Config.LogDir)
	counts := map[string]int{}
	shown := []map[string]any{}
	for i := len(events) - 1; i >= 0; i-- {
		kind, _ := events[i]["kind"].(string)
		if problemKinds[kind] {
			counts[kind]++
		}
		if !everything && !problemKinds[kind] {
			continue
		}
		if query != "" {
			raw, _ := json.Marshal(events[i])
			if !strings.Contains(strings.ToLower(string(raw)), query) {
				continue
			}
		}
		if len(shown) < 400 {
			shown = append(shown, events[i])
		}
	}
	return map[string]any{"events": shown, "problem_counts": counts, "scanned": len(events)}, nil
}

// diskFile is one file in the data folder, with whether the page may delete it.
type diskFile struct {
	Name      string `json:"name"`
	Bytes     int64  `json:"bytes"`
	What      string `json:"what"`
	Removable bool   `json:"removable"`
}

var rotatedLog = regexp.MustCompile(`^server\.log\.[0-9]+$`)

// describe says what a data file is in plain words. Only rotated server logs may be
// removed; the database, signing key, settings, sales and audit records never.
func describe(name string) (string, bool) {
	base := filepath.Base(name)
	switch {
	case rotatedLog.MatchString(base):
		return "Older server log (safe to remove)", true
	case base == "server.log":
		return "Current server log", false
	case base == "licenses.db":
		return "License database (keys, trials, orders)", false
	case strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".pub"):
		return "Signing key (never removed)", false
	case base == "sales.jsonl":
		return "Payment record", false
	case base == "admin-audit.jsonl":
		return "Owner action record", false
	case base == "config.json" || base == "admin-account.json":
		return "Settings", false
	}
	return "Other server file", false
}

func (s *Server) dataFiles() []diskFile {
	files := []diskFile{}
	filepath.WalkDir(s.Config.DataDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		relative, _ := filepath.Rel(s.Config.DataDir, path)
		what, removable := describe(path)
		removable = removable && filepath.Dir(path) == filepath.Clean(s.Config.LogDir)
		files = append(files, diskFile{Name: filepath.ToSlash(relative), Bytes: info.Size(), What: what, Removable: removable})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Bytes > files[j].Bytes })
	return files
}

func (s *Server) ownerDisk(r *http.Request, _ *ownerCall) (any, error) {
	total, free, err := diskSpace(s.Config.DataDir)
	result := map[string]any{"files": s.dataFiles(), "data_bytes": FolderBytes(s.Config.DataDir),
		"budget_bytes": s.Config.DataBudgetBytes()}
	if err == nil {
		result["disk_total"], result["disk_free"] = total, free
	}
	return result, nil
}

// ownerClean removes older server logs only. With preview it lists them first.
func (s *Server) ownerClean(r *http.Request, call *ownerCall) (any, error) {
	var request struct {
		Preview bool `json:"preview"`
	}
	decode(r, &request)
	var freed int64
	removed := []string{}
	for _, file := range s.dataFiles() {
		if !file.Removable {
			continue
		}
		if !request.Preview {
			if err := os.Remove(filepath.Join(s.Config.DataDir, filepath.FromSlash(file.Name))); err != nil {
				continue
			}
		}
		freed += file.Bytes
		removed = append(removed, file.Name)
	}
	if !request.Preview {
		s.ownerAudit("clean", call.ip, strings.Join(removed, ", "), true)
	}
	return map[string]any{"files": removed, "bytes": freed, "preview": request.Preview}, nil
}
