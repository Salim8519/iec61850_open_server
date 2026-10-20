package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
	"studio-engine/internal/tools/pytext"
)

// readerState is the device being read and the table of its values.
type readerState struct {
	state     string
	message   string
	target    model.Target
	system    []any
	engineID  string
	walkCount int64
	table     map[string]*pyjson.Object // by OID
	order     []string                  // OIDs in MIB order
	polled    []string
	problem   string
}

func (r readerState) describe() string {
	if r.target.Host == "" {
		return ""
	}
	who := "community " + r.target.Community
	if r.target.Version == "v3" {
		who = "user " + r.target.User.Name
	}
	return fmt.Sprintf("%s:%d (%s, %s)", r.target.Host, r.target.Port, r.target.Version, who)
}

// openTimeout covers v3 discovery and the system group with the target's retries.
func openTimeout(target model.Target) time.Duration {
	each := time.Duration(max(target.TimeoutMS, 2000)) * time.Millisecond
	return each*time.Duration(3*(max(target.Retries, 1)+1)) + 5*time.Second
}

// Connect opens the device, reads its system group, then walks it in the background and
// polls the key values (owner: walk all, key points on top).
func (s *Service) Connect(target model.Target) (Status, error) {
	config := s.Config()
	config.Reader.Target = target
	if err := s.save(config); err != nil {
		return Status{}, err
	}
	s.mu.Lock()
	s.reader = readerState{state: "connecting", target: target, table: map[string]*pyjson.Object{}}
	s.mu.Unlock()
	s.changed(ChangeStatus)
	answer, err := s.call("read_open", map[string]any{"target": target}, openTimeout(target))
	if err != nil {
		s.mu.Lock()
		s.reader.state, s.reader.message = "failed", err.Error()
		s.mu.Unlock()
		s.changed(ChangeStatus)
		s.releaseIfIdle()
		return s.Status(), apperr.ValueError(err.Error())
	}
	system, _ := field(answer, "system").([]any)
	s.mu.Lock()
	s.reader.system = s.MIBs.Decorate(system)
	s.reader.engineID = pytext.Str(field(answer, "engine_id"))
	s.mu.Unlock()
	go s.walk("")
	return s.Status(), nil
}

// Walk reads the whole device again (or the branch under root).
func (s *Service) Walk(root string) (Status, error) {
	s.mu.Lock()
	state := s.reader.state
	s.mu.Unlock()
	if state != "connected" && state != "failed" {
		return Status{}, apperr.ValueError("Connect to a device first")
	}
	go s.walk(root)
	return s.Status(), nil
}

func (s *Service) walk(root string) {
	s.mu.Lock()
	s.reader.state, s.reader.walkCount, s.reader.message = "walking", 0, ""
	target := s.reader.target
	s.mu.Unlock()
	s.call("read_poll", map[string]any{"oids": []string{}}, 10*time.Second) // a walk and a poll do not mix
	s.changed(ChangeStatus)
	answer, err := s.call("read_walk", map[string]any{"root": root}, 30*time.Minute+openTimeout(target))
	s.mu.Lock()
	if err != nil {
		s.reader.state, s.reader.message = "failed", err.Error()
		s.mu.Unlock()
		s.changed(ChangeStatus)
		return
	}
	values, _ := field(answer, "values").([]any)
	if root == "" {
		s.reader.table = map[string]*pyjson.Object{}
	}
	for _, row := range s.MIBs.Decorate(values) {
		if object, isObject := row.(*pyjson.Object); isObject {
			s.reader.table[pytext.Str(field(object, "oid"))] = object
		}
	}
	s.reader.order = sortedOIDs(s.reader.table)
	s.reader.walkCount = int64(len(values))
	s.reader.state, s.reader.message = "connected", pytext.Str(field(answer, "problem"))
	s.mu.Unlock()
	s.changed(ChangeValues)
	s.repoll()
}

func sortedOIDs(table map[string]*pyjson.Object) []string {
	type keyed struct {
		text string
		oid  proto.OID
	}
	items := make([]keyed, 0, len(table))
	for text := range table {
		oid, _ := proto.ParseOID(text)
		items = append(items, keyed{text, oid})
	}
	sort.Slice(items, func(a, b int) bool { return items[a].oid.Compare(items[b].oid) < 0 })
	order := make([]string, len(items))
	for index, item := range items {
		order[index] = item.text
	}
	return order
}

// Disconnect stops reading.
func (s *Service) Disconnect() Status {
	s.mu.Lock()
	connected := s.process != nil && s.reader.state != "disconnected"
	s.reader = readerState{state: "disconnected"}
	s.mu.Unlock()
	if connected {
		s.call("read_close", nil, 30*time.Second)
	}
	s.changed(ChangeStatus)
	s.changed(ChangeValues)
	s.releaseIfIdle()
	return s.Status()
}

// SetDevice writes one value to the device being read (after the page's confirmation).
func (s *Service) SetDevice(point model.Point) ([]any, error) {
	if _, err := proto.ParseValue(point.Type, point.Value, point.Hex); err != nil {
		return nil, apperr.ValueError(err.Error())
	}
	answer, err := s.call("read_set", map[string]any{"point": point}, 30*time.Second)
	if err != nil {
		return nil, apperr.ValueError(err.Error())
	}
	values, _ := field(answer, "values").([]any)
	rows := s.MIBs.Decorate(values)
	s.mu.Lock()
	if s.reader.table == nil {
		s.reader.table = map[string]*pyjson.Object{}
	}
	for _, row := range rows {
		if object, isObject := row.(*pyjson.Object); isObject {
			s.reader.table[pytext.Str(field(object, "oid"))] = object
		}
	}
	s.mu.Unlock()
	s.changed(ChangeValues)
	return rows, nil
}

// trimmed is a user-typed OID list without blanks and duplicates.
func trimmed(oids []string) []string {
	seen, out := map[string]bool{}, []string{}
	for _, oid := range oids {
		oid = strings.TrimPrefix(strings.TrimSpace(oid), ".")
		if oid != "" && !seen[oid] {
			seen[oid] = true
			out = append(out, oid)
		}
	}
	return out
}
