package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"substation-license-server/internal/release"
)

// siteVersionPattern finds the installer the download page links, which is the
// version scadabench.com offers (website/deploy.cmd checks the page links it).
var siteVersionPattern = regexp.MustCompile(`ScadaBench-Setup-([0-9]{1,4}\.[0-9]{1,4}\.[0-9]{1,4})\.exe`)

// siteWatch remembers the website's version for five minutes, so opening the
// page often never hammers the website.
type siteWatch struct {
	mu      sync.Mutex
	version string
	problem string
	checked time.Time
	fetch   func(ctx context.Context, url string) (string, error)
}

func fetchPage(ctx context.Context, url string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "ScadaBench owner page")
	response, err := (&http.Client{Timeout: 8 * time.Second}).Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("the download page answered " + response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	return string(body), err
}

func (w *siteWatch) current(ctx context.Context, url string, now time.Time) (string, string, time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.checked.IsZero() && now.Sub(w.checked) < 5*time.Minute {
		return w.version, w.problem, w.checked
	}
	fetch := w.fetch
	if fetch == nil {
		fetch = fetchPage
	}
	page, err := fetch(ctx, url)
	w.version, w.problem, w.checked = "", "", now
	if err != nil {
		w.problem = "Could not read the website: " + err.Error()
	} else if found := siteVersionPattern.FindStringSubmatch(page); found != nil {
		w.version = found[1]
	} else {
		w.problem = "The download page links no installer."
	}
	return w.version, w.problem, w.checked
}

// ownerVersions compares what the website offers with what the update check tells
// installed apps. Apps send nothing about themselves (ADR-0071), so the version each
// customer runs is unknown by design.
func (s *Server) ownerVersions(r *http.Request, _ *ownerCall) (any, error) {
	info, published, err := release.Load(s.Config.DataDir)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(s.Config.AdminOrigin, "/") + "/download/"
	site, problem, checked := s.owner.site.current(r.Context(), url, s.Now())
	result := map[string]any{"website": site, "website_problem": problem, "website_checked": checked.Unix(),
		"website_page": url, "go": runtime.Version(), "in_step": published && site != "" && info.Version == site}
	if published {
		result["release"] = info
	}
	if program, err := os.Executable(); err == nil {
		if stat, err := os.Stat(program); err == nil {
			result["server_built"] = stat.ModTime().Unix()
		}
	}
	return result, nil
}
