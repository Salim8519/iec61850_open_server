// Package agent runs simulated SNMP devices (ADR-0092): one UDP socket per device that answers
// GET, GETNEXT, GETBULK and SET in v1, v2c and v3 (USM) and sends traps and informs.
package agent

import (
	"fmt"
	"math"
	"sort"
	"time"

	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

// entry is one simulated OID with its parsed value. since is when a counter last had Value.
type entry struct {
	oid   proto.OID
	point model.Point
	value proto.Value
	since time.Time
}

// store is a device's points ordered by OID.
type store struct {
	entries []*entry
	index   map[string]*entry
	started time.Time
}

// newStore parses and orders points; the error names the first bad point.
func newStore(points []model.Point, now time.Time) (*store, error) {
	s := &store{index: map[string]*entry{}, started: now}
	for _, point := range points {
		oid, err := proto.ParseOID(point.OID)
		if err != nil {
			return nil, fmt.Errorf("point %s: %v", point.OID, err)
		}
		value, err := proto.ParseValue(point.Type, point.Value, point.Hex)
		if err != nil {
			return nil, fmt.Errorf("point %s: %v", point.OID, err)
		}
		key := oid.String()
		if _, duplicate := s.index[key]; duplicate {
			return nil, fmt.Errorf("point %s is listed twice", key)
		}
		item := &entry{oid: oid, point: point, value: value, since: now}
		s.entries = append(s.entries, item)
		s.index[key] = item
	}
	sort.Slice(s.entries, func(a, b int) bool { return s.entries[a].oid.Compare(s.entries[b].oid) < 0 })
	return s, nil
}

// get is the exact point, or nil.
func (s *store) get(oid proto.OID) *entry { return s.index[oid.String()] }

// next is the first point after oid, or nil at the end of the view.
func (s *store) next(oid proto.OID) *entry {
	at := sort.Search(len(s.entries), func(i int) bool { return s.entries[i].oid.Compare(oid) > 0 })
	if at == len(s.entries) {
		return nil
	}
	return s.entries[at]
}

// missing is the v2 exception for a GET of an OID that has no point: noSuchInstance when
// the object exists with other instances, else noSuchObject.
func (s *store) missing(oid proto.OID) proto.Value {
	if len(oid) > 1 {
		parent := oid[:len(oid)-1]
		if after := s.next(parent); after != nil && after.oid.HasPrefix(parent) {
			return proto.NoSuchInstance
		}
	}
	return proto.NoSuchObject
}

// current is a point's value now: uptime counts hundredths of a second since the device
// started (plus the stored offset), a counter grows by its rate and wraps like its type.
func (s *store) current(item *entry, now time.Time) proto.Value {
	value := item.value
	switch item.point.Behaviour {
	case model.BehaviourUptime:
		ticks := uint64(now.Sub(s.started) / (10 * time.Millisecond))
		value.Uint = (value.Uint + ticks) & math.MaxUint32
		if value.Type == proto.TagInteger {
			value.Int = int64(value.Uint & math.MaxInt32)
		}
	case model.BehaviourCounter:
		grown := uint64(math.Max(0, item.point.Rate*now.Sub(item.since).Seconds()))
		switch value.Type {
		case proto.TagCounter64:
			value.Uint += grown
		case proto.TagCounter32, proto.TagGauge32, proto.TagTimeTicks:
			value.Uint = (value.Uint + grown) & math.MaxUint32
		case proto.TagInteger:
			value.Int = int64(int32(value.Int + int64(grown)))
		}
	}
	return value
}

// set stores a new value (from a SET or the page); counters continue from it.
func (s *store) set(item *entry, value proto.Value, now time.Time) {
	value.Bytes = append([]byte(nil), value.Bytes...) // received bytes live in the socket buffer
	item.value, item.since = value, now
	kind, text, isHex := proto.ValueText(value)
	item.point.Type, item.point.Value, item.point.Hex = kind, text, isHex
	if item.point.Behaviour == model.BehaviourUptime {
		s.started = now // an uptime written from outside restarts from the written value
	}
}

// uptime is sysUpTime: hundredths of a second since the device started.
func (s *store) uptime(now time.Time) uint32 {
	return uint32(now.Sub(s.started) / (10 * time.Millisecond))
}

// snapshot is every point with its current value, for the page.
func (s *store) snapshot(now time.Time) []model.Point {
	points := make([]model.Point, len(s.entries))
	for index, item := range s.entries {
		point := item.point
		point.Type, point.Value, point.Hex = proto.ValueText(s.current(item, now))
		if point.Type == "" {
			point.Type = item.point.Type
		}
		points[index] = point
	}
	return points
}
