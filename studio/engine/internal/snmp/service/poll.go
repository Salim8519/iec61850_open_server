package service

import (
	"strings"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// keyPrefixes are the values shown on top and polled after a walk: the system group, port
// names and status, entity and sensor values, UPS battery and alarms, NTP status and CPU load.
var keyPrefixes = []string{
	"1.3.6.1.2.1.1.",            // system
	"1.3.6.1.2.1.2.2.1.2.",      // ifDescr
	"1.3.6.1.2.1.2.2.1.7.",      // ifAdminStatus
	"1.3.6.1.2.1.2.2.1.8.",      // ifOperStatus
	"1.3.6.1.2.1.31.1.1.1.1.",   // ifName
	"1.3.6.1.2.1.47.1.1.1.1.7.", // entPhysicalName
	"1.3.6.1.2.1.99.1.1.1.4.",   // entPhySensorValue
	"1.3.6.1.2.1.33.1.2.",       // upsBattery
	"1.3.6.1.2.1.33.1.6.1.",     // upsAlarmsPresent
	"1.3.6.1.2.1.197.1.2.",      // ntpEntStatus
	"1.3.6.1.2.1.25.3.3.1.2.",   // hrProcessorLoad
}

// maxPolled caps how many values are re-read each interval.
const maxPolled = 500

// MinPollMS is the fastest interval the page offers.
const MinPollMS = 200

func isKey(oid string) bool {
	for _, prefix := range keyPrefixes {
		if strings.HasPrefix(oid, prefix) {
			return true
		}
	}
	return false
}

// repoll polls the key values found by the walk plus the watched OIDs.
func (s *Service) repoll() {
	config := s.Config()
	s.mu.Lock()
	if s.reader.state != "connected" {
		s.mu.Unlock()
		return
	}
	oids := trimmed(config.Reader.Watched)
	for _, oid := range s.reader.order {
		if len(oids) >= maxPolled {
			break
		}
		if isKey(oid) {
			oids = append(oids, oid)
		}
	}
	oids = trimmed(oids)
	s.reader.polled = oids
	s.mu.Unlock()
	_, err := s.call("read_poll", map[string]any{"oids": oids, "every_ms": config.Reader.EveryMS}, 10*time.Second)
	if err != nil {
		s.mu.Lock()
		s.reader.problem = err.Error()
		s.mu.Unlock()
	}
	s.changed(ChangeStatus)
}

// mergeValues takes a poll round's changed values into the table.
func (s *Service) mergeValues(event *pyjson.Object) {
	values, _ := field(event, "values").([]any)
	rows := s.MIBs.Decorate(values)
	s.mu.Lock()
	if s.reader.table == nil {
		s.mu.Unlock()
		return
	}
	added := false
	for _, row := range rows {
		object, isObject := row.(*pyjson.Object)
		if !isObject {
			continue
		}
		oid := pytext.Str(field(object, "oid"))
		if _, known := s.reader.table[oid]; !known {
			added = true
		}
		s.reader.table[oid] = object
	}
	if added {
		s.reader.order = sortedOIDs(s.reader.table)
	}
	problem := pytext.Str(field(event, "problem"))
	changedProblem := problem != s.reader.problem
	s.reader.problem = problem
	s.mu.Unlock()
	if len(rows) > 0 {
		s.changed(ChangeValues)
	}
	if changedProblem {
		s.changed(ChangeStatus)
	}
}

// SetPoll changes how often the shown values are re-read.
func (s *Service) SetPoll(everyMS int) (Status, error) {
	if everyMS < MinPollMS || everyMS > 3600000 {
		return Status{}, apperr.ValueError("Choose an interval from 0.2 seconds to 1 hour")
	}
	config := s.Config()
	config.Reader.EveryMS = everyMS
	if err := s.save(config); err != nil {
		return Status{}, err
	}
	s.repoll()
	return s.Status(), nil
}

// Watch sets the OIDs polled besides the key values.
func (s *Service) Watch(oids []string) (Status, error) {
	config := s.Config()
	config.Reader.Watched = trimmed(oids)
	if len(config.Reader.Watched) > maxPolled {
		return Status{}, apperr.ValueError("Watch at most 500 values")
	}
	if err := s.save(config); err != nil {
		return Status{}, err
	}
	s.repoll()
	return s.Status(), nil
}

// Values is the reader's table in MIB order; key values are marked.
func (s *Service) Values() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	watched := map[string]bool{}
	for _, oid := range s.reader.polled {
		watched[oid] = true
	}
	rows := make([]any, 0, len(s.reader.order))
	for _, oid := range s.reader.order {
		row := s.reader.table[oid]
		if row == nil {
			continue
		}
		copied := pyjson.NewObject()
		for _, key := range row.Keys() {
			value, _ := row.Get(key)
			copied.Set(key, value)
		}
		rows = append(rows, copied.Set("key", isKey(oid)).Set("polled", watched[oid]))
	}
	return rows
}
