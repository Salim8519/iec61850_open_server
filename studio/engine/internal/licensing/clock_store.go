package licensing

import (
	"time"
	"math/big"
	"os"
	"path/filepath"

	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// Each license clock value is kept in three places so deleting one copy changes nothing
// (licensing/clock_store.py): a dot-file in the license folder, a REG_SZ value under
// HKCU\Software\ScadaBench and a string field of license_state.json. Readers decide on the
// copies that remain; every read or write failure (antivirus, Controlled Folder Access, a
// locked registry) is ignored silently, so a blocked place can never stop ScadaBench. The
// native guard reads the same places (license_clock.cpp).

// RegistryKey is the per-user key that holds the registry copies.
const RegistryKey = `Software\ScadaBench`

// registryNames are the value names of time_anchor, server_time, grace_start,
// revoked_ids (the cancelled license IDs, see revoked_ids.go) and the lease_token and
// lease_answer of lease licenses (see lease.go).
var registryNames = map[string]string{"time_anchor": "TimeAnchor", "server_time": "ServerTime",
	"grace_start": "GraceStart", "revoked_ids": "RevokedLicenses", "lease_token": "LeaseToken",
	"lease_answer": "LeaseAnswer"}

// Registry is the registry place: string values under one key. Get reports false for a
// missing, unreadable or non-REG_SZ value; Set ignores every failure.
type Registry interface {
	Get(name string) (string, bool)
	Set(name, text string)
}

// ClockValue is one named value kept in the file, the registry (nil: none) and
// license_state.json.
type ClockValue struct {
	Name         string
	File         string
	StateFile    string
	Registry     Registry
	registryName string
}

// NewClockValue returns the value name ("time_anchor", "server_time" or "grace_start")
// kept under the license home.
func NewClockValue(name, home string, registry Registry) ClockValue {
	return ClockValue{Name: name, File: filepath.Join(home, "."+name),
		StateFile: filepath.Join(home, "license_state.json"), Registry: registry, registryName: registryNames[name]}
}

// state is license_state.json as an object; anything unreadable or not an object is {}.
func (v ClockValue) state() *pyjson.Object {
	value, ok := pytext.ReadJSON(v.StateFile)
	if object, isObject := value.(*pyjson.Object); ok && isObject {
		return object
	}
	return pyjson.NewObject()
}

// ReadAll returns the copies that exist, as text (missing or unreadable places and empty
// copies are skipped), in the order file, registry, license_state.json.
func (v ClockValue) ReadAll() []string {
	copies := []string{}
	if text, err := pytext.ReadText(os.ReadFile(v.File)); err == nil {
		copies = append(copies, pytext.Strip(universalNewlines(text)))
	}
	if v.Registry != nil {
		if text, ok := v.Registry.Get(v.registryName); ok {
			copies = append(copies, text)
		}
	}
	if text, isText := lookupState(v.state(), v.Name).(string); isText {
		copies = append(copies, text)
	}
	kept := copies[:0]
	for _, copy := range copies {
		if copy != "" {
			kept = append(kept, copy)
		}
	}
	return kept
}

func lookupState(state *pyjson.Object, name string) any {
	value, _ := state.Get(name)
	return value
}

// Places is how many places hold the value: 3 with a registry, otherwise 2.
func (v ClockValue) Places() int {
	if v.Registry != nil {
		return 3
	}
	return 2
}

// ReadInts returns int() of every copy that is an integer, as Python's unbounded ints.
func (v ClockValue) ReadInts() []*big.Int {
	numbers := []*big.Int{}
	for _, copy := range v.ReadAll() {
		if number, err := pytext.ParseInt(copy); err == nil {
			numbers = append(numbers, number)
		}
	}
	return numbers
}

// Write stores text in every place; failures are ignored. license_state.json is
// rewritten only when its copy differs.
func (v ClockValue) Write(text string) {
	writePatiently(v.File, []byte(text))
	if v.Registry != nil {
		v.Registry.Set(v.registryName, text)
	}
	state := v.state()
	if current, found := state.Get(v.Name); found && current == any(text) {
		return
	}
	data, err := pyjson.Dumps(state.Set(v.Name, text))
	if err == nil {
		writePatiently(v.StateFile, data)
	}
}

// writePatiently retries a write a few times: right after a file is written or deleted,
// antivirus or the search indexer may hold it for a moment ("used by another process").
// A write that still fails is ignored, as before (clock_store._write_text).
func writePatiently(path string, data []byte) {
	for attempt := 0; attempt < writeAttempts; attempt++ {
		if pytext.WriteReplace(path, data) == nil {
			return
		}
		time.Sleep(writeRetryDelay)
	}
	_ = os.Remove(path + ".tmp") // a write that never succeeded leaves no half-written file behind
}

const (
	writeAttempts   = 20
	writeRetryDelay = 40 * time.Millisecond
)

// universalNewlines is read_text's newline translation ("\r\n" and "\r" become "\n").
func universalNewlines(text string) string {
	out := make([]byte, 0, len(text))
	for index := 0; index < len(text); index++ {
		if text[index] != '\r' {
			out = append(out, text[index])
			continue
		}
		out = append(out, '\n')
		if index+1 < len(text) && text[index+1] == '\n' {
			index++
		}
	}
	return string(out)
}
