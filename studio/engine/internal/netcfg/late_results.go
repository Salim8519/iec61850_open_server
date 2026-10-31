package netcfg

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"studio-engine/internal/pyjson"
)

// lateResults guards the catch-up of helper results the app stopped waiting for (ADR-0131).
var lateResults = struct {
	sync.Mutex
	waiting map[string]bool // request files whose RunHelper still waits; never caught up
	checked map[string]bool // result files already journaled or skipped; not read again
}{waiting: map[string]bool{}, checked: map[string]bool{}}

func waitFor(request string, waiting bool) {
	lateResults.Lock()
	defer lateResults.Unlock()
	if waiting {
		lateResults.waiting[request] = true
	} else {
		delete(lateResults.waiting, request)
	}
}

// catchUp journals every apply or remove request whose helper wrote its result after the app
// had stopped waiting. Without this the changes Windows made stay unknown: Clean up cannot list
// them and the lab page cannot move them (the owner removed addresses by hand). A request with
// any journal record of its fingerprint is already journaled; one RunHelper still waits for is
// left to it. Records are only appended.
func (s *NetworkService) catchUp() error {
	lateResults.Lock()
	defer lateResults.Unlock()
	folder := filepath.Join(s.Directory, "network", "requests")
	found, _ := filepath.Glob(filepath.Join(folder, "*.result.json"))
	results := []string{}
	for _, result := range found {
		request := strings.TrimSuffix(result, ".result.json") + ".json"
		if !lateResults.checked[result] && !lateResults.waiting[request] {
			results = append(results, result)
		}
	}
	if len(results) == 0 {
		return nil
	}
	records, err := s.Journal.Records()
	if err != nil {
		return err
	}
	journaled := map[string]bool{}
	for _, record := range records {
		journaled[pyStr(get(record, "fingerprint"))] = true
	}
	sort.Strings(results)
	for _, result := range results {
		request := strings.TrimSuffix(result, ".result.json") + ".json"
		action := strings.SplitN(filepath.Base(request), "-", 2)[0]
		lateResults.checked[result] = true
		if action != "apply" && action != "remove" {
			continue
		}
		payload, err := os.ReadFile(request)
		if err != nil {
			continue
		}
		parsed, err := pyjson.Decode(payload)
		body, _ := parsed.(*pyjson.Object)
		if err != nil || body == nil || get(body, "action") != action {
			continue
		}
		fingerprint := pyStr(get(body, "fingerprint"))
		if fingerprint == "" || journaled[fingerprint] {
			continue
		}
		byKey, err := readHelperResult(result)
		if err != nil {
			continue // refused or unreadable: Windows changed nothing that can be journaled
		}
		items := objects(get(body, "items"))
		if _, err := s.journalResults(action, items, fingerprint, byKey); err != nil {
			return err
		}
		journaled[fingerprint] = true
	}
	return nil
}
