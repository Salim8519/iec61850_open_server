package netcfg

import (
	"errors"
	"os"
	"path/filepath"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

func conflict(message string, fingerprint any) error {
	err := apperr.Coded(message, "conflict")
	if fingerprint != nil {
		err.Details = pyjson.NewObject().Set("fingerprint", fingerprint)
	}
	return err
}

// ownedAdds is [item for item in items if item["action"] == "add" and item["owned"]].
func ownedAdds(items []*pyjson.Object) []*pyjson.Object {
	found := []*pyjson.Object{}
	for _, item := range items {
		if get(item, "action") == "add" && truthy(get(item, "owned")) {
			found = append(found, item)
		}
	}
	return found
}

// Apply is service.apply: re-check the reviewed preview, run the helper for the owned
// additions, then switch the project to lab mode when everything succeeded.
func (s *NetworkService) Apply(fingerprint, confirm any) (*pyjson.Object, error) {
	if !isTrue(confirm) {
		return nil, apperr.New("Review the change list and confirm it before applying")
	}
	key, _ := fingerprint.(string)
	s.mu.Lock()
	stored, found := s.previews[key]
	s.mu.Unlock()
	if _, isText := fingerprint.(string); !found || !isText {
		return nil, conflict("This preview has expired. Review the changes again.", nil)
	}
	s.clearCache()
	_, fresh, err := s.preview(stored.guid, stored.strategy, stored.manual)
	if err != nil {
		return nil, err
	}
	if fresh.fingerprint != key {
		return nil, conflict("The network changed since you reviewed it. Review the new change list.", fresh.fingerprint)
	}
	if !fresh.ready {
		return nil, apperr.New("Fix the highlighted address problems before applying")
	}
	results := []any{}
	if adds := ownedAdds(fresh.items); len(adds) > 0 {
		if results, err = s.RunHelper("apply", adds, key); err != nil {
			return nil, err
		}
	}
	failed := false
	for _, entry := range results {
		ok, found := entry.(*pyjson.Object).Get("ok")
		if !found {
			return nil, apperr.KeyError("ok")
		}
		failed = failed || !truthy(ok)
	}
	if !failed {
		ieds, snmp := splitRows(fresh.rows)
		if s.SNMP != nil && len(snmp) > 0 { // saved first: the devices move when lab mode starts
			if err := s.SNMP.SetSNMPLabIPs(snmp); err != nil {
				return nil, err
			}
		}
		if ieds == nil {
			ieds = []Row{}
		}
		if err := s.Projects.EnterLabMode(ieds, fresh.adapter); err != nil {
			return nil, err
		}
	}
	s.clearCache()
	message := "Lab network is ready. Press Run to start the IEDs on the network."
	if failed {
		message = "Some changes failed; nothing else was switched. See each line for the reason."
	}
	return pyjson.NewObject().Set("results", results).Set("applied", !failed).Set("message", message), nil
}

// RunHelper is service._run_helper: write the request file, start the elevated helper, read
// its result, journal every item and return one line per item with its description.
func (s *NetworkService) RunHelper(action string, items []*pyjson.Object, fingerprint string) ([]any, error) {
	folder := filepath.Join(s.Directory, "network", "requests")
	if err := os.MkdirAll(folder, 0o777); err != nil {
		return nil, err
	}
	request := filepath.Join(folder, action+"-"+s.NewID()+".json")
	clean := make([]*pyjson.Object, len(items))
	for index, item := range items {
		clean[index] = copyObject(item)
		clean[index].Delete("description")
	}
	payload, err := pyjson.Dumps(pyjson.NewObject().Set("action", action).Set("fingerprint", fingerprint).
		Set("items", itemsAny(clean)))
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(request, payload, 0o666); err != nil {
		return nil, err
	}
	executable, arguments := HelperCommand(s.Executable, request, sha256Hex(payload), s.Program)
	if _, err := s.Elevator.Elevate(executable, arguments); err != nil {
		if errors.Is(err, ErrElevationCancelled) {
			return nil, apperr.New(err.Error())
		}
		return nil, err
	}
	byKey, err := readHelperResult(withSuffix(request, ".result.json"))
	if err != nil {
		return nil, err
	}
	return s.journalResults(action, clean, fingerprint, byKey)
}

// readHelperResult reads <request>.result.json into {key: entry}.
func readHelperResult(path string) (map[string]*pyjson.Object, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, apperr.New("The network helper finished without a result; nothing was recorded as changed")
	}
	if err != nil {
		return nil, err
	}
	value, err := pyjson.Decode(data)
	if err != nil {
		return nil, apperr.ValueError(err.Error())
	}
	outcome, _ := value.(*pyjson.Object)
	if outcome == nil {
		return nil, apperr.ValueError("helper result is not an object")
	}
	if truthy(get(outcome, "refused")) {
		reason, found := outcome.Get("reason")
		if !found {
			reason = ""
		}
		return nil, apperr.New("The network helper refused the request: " + pyStr(reason))
	}
	results, found := outcome.Get("results")
	if !found {
		return nil, apperr.KeyError("results")
	}
	byKey := map[string]*pyjson.Object{}
	for _, raw := range objects(results) {
		key, found := raw.Get("key")
		if !found {
			return nil, apperr.KeyError("key")
		}
		byKey[pyStr(key)] = raw
	}
	return byKey, nil
}

func (s *NetworkService) journalResults(action string, clean []*pyjson.Object, fingerprint string,
	byKey map[string]*pyjson.Object) ([]any, error) {
	label, done := Describe, "applied"
	if action != "apply" {
		label, done = DescribeRemoval, "removed"
	}
	lines := []any{}
	for _, item := range clean {
		key, err := ItemKey(item)
		if err != nil {
			return nil, err
		}
		entry, found := byKey[key]
		if !found {
			entry = pyjson.NewObject().Set("ok", false).Set("message", "No result")
		}
		ok, hasOK := entry.Get("ok")
		if !hasOK {
			return nil, apperr.KeyError("ok")
		}
		event := "failed"
		if truthy(ok) {
			event = done
		}
		message, hasMessage := entry.Get("message")
		if !hasMessage {
			message = ""
		}
		if _, err := s.Journal.Append(event, item, fingerprint, message); err != nil {
			return nil, err
		}
		text, err := label(item)
		if err != nil {
			return nil, err
		}
		line := pyjson.NewObject()
		if found {
			line = copyObject(entry)
		}
		lines = append(lines, line.Set("description", text))
	}
	return lines, nil
}
