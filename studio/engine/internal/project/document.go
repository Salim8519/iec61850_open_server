// Package project is the pure engineering-project domain of the Go engine: the compact
// archive document, browser contract validation, cross-reference checks, legacy diagram
// migration, IED removal, restore preparation, snapshots and UI preferences, ported byte
// for byte from studio_app/projects and studio_app/domain. It performs no file or network
// I/O; the service layer reads and writes archives and injects the SCL re-import.
package project

import (
	"crypto/sha256"
	"encoding/hex"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// CurrentFormat is the archive format written today (14: SCADA roles and master,
// ADR-0061; 15: standard commands only, ADR-0069).
const CurrentFormat = 15

// IEDFields are the stored per-IED fields besides the SCL source.
var IEDFields = []string{"id", "name", "filename", "ip", "port", "role", "live", "lab_ip", "goose_publish", "test_mode",
	"scd_deadbands"} // stored only while on (ADR-0118)

// SourceDigest is the sha256 hex digest of the UTF-8 SCL text.
func SourceDigest(source string) string {
	digest := sha256.Sum256([]byte(source))
	return hex.EncodeToString(digest[:])
}

// Compact is document.compact: the project without derived runtime models, each original
// SCL text stored once under its digest. Values are shared, not copied.
func Compact(project *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range project.Keys() {
		switch key {
		case "ieds", "format", "scl_sources", "migration_notice":
			continue
		}
		value, _ := project.Get(key)
		result.Set(key, value)
	}
	sources := pyjson.NewObject()
	ieds := []any{}
	stored, _ := pyval.Lookup(project, "ieds", []any{}).([]any)
	for _, item := range stored {
		ied := item.(*pyjson.Object)
		source, _ := ied.Get("source")
		digest := SourceDigest(source.(string))
		sources.Set(digest, source)
		entry := pickFields(ied)
		entry.Set("source_id", digest)
		ieds = append(ieds, entry)
	}
	return result.Set("format", int64(CurrentFormat)).Set("ieds", ieds).Set("scl_sources", sources)
}

func pickFields(ied *pyjson.Object) *pyjson.Object {
	entry := pyjson.NewObject()
	for _, key := range IEDFields {
		if value, found := ied.Get(key); found {
			entry.Set(key, value)
		}
	}
	return entry
}

// compactFormat reports `format in COMPACT_FORMATS` (2..15); lists and dicts raise TypeError.
func compactFormat(format any) (bool, error) {
	if err := pyval.CheckHashable(format); err != nil {
		return false, err
	}
	for version := int64(2); version <= CurrentFormat; version++ {
		if pyval.Equal(format, version) {
			return true, nil
		}
	}
	return false, nil
}

// ExpandDecoded is the end of archive.decode after json.loads of project.json: a
// non-object is "Unsupported project format", an object is expanded.
func ExpandDecoded(document any) (*pyjson.Object, error) {
	object, isDict := document.(*pyjson.Object)
	if !isDict {
		return nil, apperr.New("Unsupported project format")
	}
	return Expand(object)
}

// Expand is document.expand: a format-1 document unchanged, a compact one with each IED's
// source restored after checking the digests.
func Expand(document *pyjson.Object) (*pyjson.Object, error) {
	format := pyval.Lookup(document, "format", nil)
	if pyval.Equal(format, int64(1)) {
		return document, nil
	}
	if known, err := compactFormat(format); err != nil {
		return nil, err
	} else if !known {
		return nil, apperr.New("Unsupported project format")
	}
	sources, isDict := pyval.Lookup(document, "scl_sources", nil).(*pyjson.Object)
	ieds, isList := pyval.Lookup(document, "ieds", nil).([]any)
	if !isDict || sources.Len() > 20 || !isList || len(ieds) > 20 {
		return nil, apperr.New("Invalid compact project source table")
	}
	for _, digest := range sources.Keys() {
		source, _ := sources.Get(digest)
		text, isText := source.(string)
		if !isText || SourceDigest(text) != digest {
			return nil, apperr.New("Project SCL source checksum mismatch")
		}
	}
	restored := []any{}
	for _, item := range ieds {
		ied, isDict := item.(*pyjson.Object)
		var sourceID any
		if isDict {
			sourceID = pyval.Lookup(ied, "source_id", nil)
		}
		key, isText := sourceID.(string)
		if !isDict || !isText || !pyval.Has(sources, key) {
			return nil, apperr.New("IED refers to a missing project SCL source")
		}
		entry := pickFields(ied)
		source, _ := sources.Get(key)
		restored = append(restored, entry.Set("source", source))
	}
	result := pyjson.NewObject()
	for _, key := range document.Keys() {
		if key != "ieds" && key != "scl_sources" {
			value, _ := document.Get(key)
			result.Set(key, value)
		}
	}
	return result.Set("ieds", restored), nil
}
