package acquisition

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"studio-engine/internal/acquisition/livedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

var errLost = errors.New("Connection to the device was lost")

// run is RealSession._run: connect, and on failure retry after 2, 4, ... 30 s.
func (s *RealSession) run() {
	defer close(s.done)
	delay := 2
	host, _ := s.get("host").(string)
	port := s.get("port")
	if host == "" {
		s.set("no_address", "This IED has no address in the SCD. Type the device address first.")
		return
	}
	for !s.isStopping() {
		s.mu.Lock()
		s.resetLostLocked()
		s.mu.Unlock()
		if err := s.session(host, port); err != nil {
			if s.isStopping() {
				break
			}
			s.set("retrying", fmt.Sprintf("%s. Trying again in %d s.", err.Error(), delay))
		} else {
			delay = 2
		}
		s.closeProcess()
		if s.wait(time.Duration(delay) * time.Second) {
			break
		}
		delay = min(delay*2, 30)
	}
}

// wait sleeps for d or until Stop; it reports true when stopping.
func (s *RealSession) wait(d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-s.stopping:
		return true
	case <-timer.C:
		return false
	}
}

func (s *RealSession) lostChannel() chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lost
}

func (s *RealSession) session(host string, port any) error {
	s.set("connecting", fmt.Sprintf("Connecting to %s:%s", host, pyv.PyStr(port)))
	process, err := s.deps.Start(s.deps.LiveExecutable, s.onEvent)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.process = process
	s.mu.Unlock()
	if s.isStopping() {
		return errors.New("stopping")
	}
	hello, err := process.Request("hello", 10*time.Second, pyjson.NewObject())
	if err != nil {
		return err
	}
	if !hasAll(pyv.GetList(pyv.Obj(hello), "capabilities"), liveCapabilities) {
		return errors.New("The live client is outdated; rebuild the native programs")
	}
	if _, err := process.Request("connect", 15*time.Second, pyv.O("ip", host, "port", port)); err != nil {
		return err
	}
	s.set("discovering", "Reading the device data model")
	discovery, err := process.Request("discover", 120*time.Second, pyjson.NewObject())
	if err != nil {
		return err
	}
	s.put("comparison", livedom.Compare(s.ied, pyv.Obj(discovery)))
	access := pyv.GetStr(pyv.GetObj(s.ied, "live"), "access")
	blocks, err := s.readBlocks(process, access)
	if err != nil {
		return err
	}
	if err := s.chooseAccess(process, blocks, access); err != nil {
		return err
	}
	health, err := readHealth(process, s.ied, blocks, access, s.deps.Clock.Now)
	if err != nil {
		return err
	}
	s.put("health", health)
	return s.live()
}

func hasAll(items []any, wanted []string) bool {
	have := map[string]bool{}
	for _, item := range items {
		if text, ok := item.(string); ok {
			have[text] = true
		}
	}
	for _, name := range wanted {
		if !have[name] {
			return false
		}
	}
	return true
}

// readBlocks asks the device for every SCD report block when access is auto_reports.
func (s *RealSession) readBlocks(process Worker, access string) (*pyjson.Object, error) {
	blocks := pyjson.NewObject()
	var refs []any
	for _, report := range pyv.Objects(pyv.GetList(s.ied, "reports")) {
		refs = append(refs, pyv.Get(report, "ref"))
	}
	if len(refs) == 0 || access != "auto_reports" {
		return blocks, nil
	}
	answer, err := process.Request("rcb_status", 30*time.Second, pyv.O("rcbs", refs))
	if err != nil {
		return nil, err
	}
	for _, item := range pyv.Objects(pyv.GetList(pyv.Obj(answer), "blocks")) {
		blocks.Set(pyv.GetStr(item, "rcb"), item)
	}
	return blocks, nil
}

// chooseAccess reserves the free report blocks and decides what is polled.
func (s *RealSession) chooseAccess(process Worker, blocks *pyjson.Object, access string) error {
	chosen := livedom.ChooseAccess(s.ied, blocks, access)
	notes := pyv.GetList(chosen, "notes")
	used := []any{}
	for _, rcb := range pyv.GetList(chosen, "reports") {
		result, err := process.Request("rcb_reserve", 30*time.Second, pyv.O("rcb", rcb))
		if err != nil {
			return err
		}
		if pyv.Truthy(pyv.Get(pyv.Obj(result), "reserved")) {
			used = append(used, rcb)
			continue
		}
		text := pyv.PyStr(rcb)
		notes = append(notes, "Report block "+text[strings.LastIndex(text, ".")+1:]+" became busy; polling instead.")
	}
	chosen.Set("notes", notes)
	chosen.Set("reports", used)
	usedSet := map[string]bool{}
	for _, rcb := range used {
		usedSet[pyv.PyStr(rcb)] = true
	}
	covered := map[string]bool{}
	for _, report := range pyv.Objects(pyv.GetList(s.ied, "reports")) {
		if !usedSet[pyv.GetStr(report, "ref")] {
			continue
		}
		for _, dataset := range pyv.Objects(pyv.GetList(s.ied, "datasets")) {
			if pyv.Equal(pyv.Get(dataset, "ref"), pyv.Get(report, "dataset_ref")) {
				for _, ref := range pyv.GetList(dataset, "signal_refs") {
					covered[pyv.PyStr(ref)] = true
				}
			}
		}
	}
	polling := []any{}
	for _, item := range s.watched {
		if !covered[pyv.GetStr(pyv.Obj(item), "ref")] {
			polling = append(polling, item)
		}
	}
	s.put("access", chosen, "polling", polling)
	return nil
}

// live is the connected loop: poll the uncovered points every poll_ms and re-read the mode
// every 10 s until Stop or a lost connection.
func (s *RealSession) live() error {
	polling := pyv.List(s.get("polling"))
	used := len(pyv.GetList(pyv.Obj(s.get("access")), "reports")) > 0
	how := "Connected. Use 'Read these now' below to read values from the device"
	switch {
	case used && len(polling) > 0:
		how = "Receiving live values by report, and polling the rest"
	case used:
		how = "Receiving live values by report"
	case len(polling) > 0:
		how = "Receiving live values by polling"
	}
	s.set("live", how)
	pollMS, _ := pyv.Int64(pyv.Get(pyv.GetObj(s.ied, "live"), "poll_ms"))
	lost := s.lostChannel()
	modeDue := time.Now().Add(ModeRefresh)
	for !s.isStopping() && !closed(lost) {
		if len(polling) > 0 {
			if _, err := s.Read(polling, "live_poll"); err != nil {
				return err
			}
		}
		if !time.Now().Before(modeDue) {
			if err := s.refreshMode(); err != nil {
				return err
			}
			modeDue = time.Now().Add(ModeRefresh)
		}
		select {
		case <-s.stopping:
		case <-lost:
		case <-time.After(time.Duration(pollMS) * time.Millisecond):
		}
	}
	if closed(lost) {
		s.values.MarkStale(s.ID(), errLost.Error())
		return errLost
	}
	return nil
}

func closed(channel chan struct{}) bool {
	select {
	case <-channel:
		return true
	default:
		return false
	}
}

func (s *RealSession) refreshMode() error {
	s.mu.Lock()
	process := s.process
	s.mu.Unlock()
	if process == nil {
		return errLost
	}
	mode, err := readMode(process, s.ied, s.deps.Clock.Now)
	if err != nil {
		return err
	}
	s.mu.Lock()
	health := pyv.Shallow(pyv.GetObj(s.status, "health"))
	health.Set("mode", mode)
	s.status.Set("health", health)
	s.mu.Unlock()
	return nil
}
