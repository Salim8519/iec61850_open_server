package health

import (
	"fmt"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// verdicts lists RANK's keys in order: a higher index is a worse verdict.
func verdicts() [4]string {
	return [4]string{"ok", "not_checked", "warning", "problem"}
}

// rank is RANK[verdict]; any other text ranks below "ok" (Python raises KeyError).
func rank(verdict string) int {
	for index, known := range verdicts() {
		if known == verdict {
			return index
		}
	}
	return -1
}

// Worst is worst: the worst verdict (the first one on ties), "not_checked" when empty.
func Worst(results []string) string {
	if len(results) == 0 {
		return "not_checked"
	}
	worst := results[0]
	for _, verdict := range results[1:] {
		if rank(verdict) > rank(worst) {
			worst = verdict
		}
	}
	return worst
}

// wireRows is _wire_rows: the listener verdicts of the GOOSE blocks this IED publishes.
func wireRows(ied, goose *pyjson.Object) []any {
	rows := []any{}
	for _, watch := range pyv.Objects(pyv.List(pyv.GetOr(goose, "watches", []any{}))) {
		if pyv.Equal(pyv.Get(pyv.OrObj(watch, "publisher"), "ied_id"), pyv.Get(ied, "id")) {
			rows = append(rows, WireCheck(watch, goose))
		}
	}
	return rows
}

// realRows is _real_rows: connection, model, report, GOOSE-block and mode rows of a real
// IED from its live session status (nil when there is no session).
func realRows(ied, session, goose *pyjson.Object) []any {
	if session == nil {
		return []any{notChecked("connection", "Connection", "Press Run to connect to the device.")}
	}
	state := pyv.Get(session, "state")
	if pyv.Equal(state, "retrying") || pyv.Equal(state, "no_address") {
		message := pyv.Get(session, "message")
		if !pyv.Truthy(message) {
			message = "The device does not answer."
		}
		return []any{row("connection", "Connection", "problem", message, nil, "scd", nil)}
	}
	health := pyv.Get(session, "health")
	if !pyv.Equal(state, "live") || !pyv.Truthy(health) {
		return []any{notChecked("connection", "Connection", "Not connected yet. ScadaBench is reaching the device.")}
	}
	reads := pyv.Obj(health)
	mode := pyv.GetObj(reads, "mode")
	when := pyv.Get(mode, "checked_at")
	message := fmt.Sprintf("Connected to %s:%s.", pyv.PyStr(pyv.Get(session, "host")), pyv.PyStr(pyv.Get(session, "port")))
	rows := []any{row("connection", "Connection", "ok", message, when, "mms_read", nil)}
	if comparison := pyv.Get(session, "comparison"); pyv.Truthy(comparison) {
		rows = append(rows, ModelRows(ied, pyv.Obj(comparison), when)...)
	}
	rows = append(rows, ReportSettingRows(ied, pyv.GetObj(reads, "rcbs"))...)
	rows = append(rows, DeviceGocbChecks(ied, pyv.GetObj(reads, "gocbs"))...)
	return append(rows, ModeRows(ied, mode)...)
}

// IEDRows is ied_rows: every check row of one IED. session is its live session status,
// nil for none (Python None).
func IEDRows(ied *pyjson.Object, running bool, session, goose *pyjson.Object) []any {
	role := pyv.GetOr(ied, "role", "simulated")
	switch {
	case pyv.Equal(role, "off"):
		return []any{notChecked("role", "IED", "This IED is left out.")}
	case !running:
		return []any{notChecked("connection", "Run", "Press Run to check the station.")}
	case pyv.Equal(role, "simulated"):
		rows := []any{notChecked("model", "Data model", "Simulated by ScadaBench; model checks apply to real devices.")}
		return append(rows, wireRows(ied, goose)...)
	}
	return append(realRows(ied, session, goose), wireRows(ied, goose)...)
}

// BuildHealth is build_health: {verdict, summary, ieds: [{id, name, role, verdict,
// checks}], unknown, unknown_note, running, checked_at}. sessions maps an IED id to its
// live session status (a missing key or nil value is Python's None); now is the check time.
func BuildHealth(ieds []*pyjson.Object, running bool, sessions map[string]*pyjson.Object,
	goose *pyjson.Object, now time.Time) *pyjson.Object {
	entries := []any{}
	var stationVerdicts []string
	counts := map[string]int64{}
	for _, ied := range ieds {
		rows := markStale(IEDRows(ied, running, sessions[pyv.GetStr(ied, "id")], goose), now)
		results := make([]string, 0, len(rows))
		for _, check := range pyv.Objects(rows) {
			results = append(results, pyv.GetStr(check, "result"))
		}
		verdict := Worst(results)
		stationVerdicts = append(stationVerdicts, verdict)
		counts[verdict]++
		entries = append(entries, pyv.O("id", pyv.Get(ied, "id"), "name", pyv.Get(ied, "name"),
			"role", pyv.GetOr(ied, "role", "simulated"), "verdict", verdict, "checks", rows))
	}
	unknown, note := []any{}, ""
	if running {
		unknown, note = UnknownPublishers(goose)
	}
	summary := pyjson.NewObject()
	for _, name := range verdicts() {
		summary.Set(name, counts[name])
	}
	return pyv.O("verdict", Worst(stationVerdicts), "summary", summary, "ieds", entries, "unknown", unknown,
		"unknown_note", note, "running", running, "checked_at", isoSeconds(now))
}

// markStale is _mark_stale: a mode row older than 30 s is stale; every row gets "stale".
func markStale(rows []any, now time.Time) []any {
	for _, check := range pyv.Objects(rows) {
		age, known := ageMicros(pyv.Get(check, "observed_at"), now)
		stale := pyv.Equal(pyv.Get(check, "area"), "mode") && known && age > 30_000_000
		check.Set("stale", stale)
	}
	return rows
}

// isoSeconds is now.isoformat(timespec="seconds") for an aware datetime: the offset is
// kept as "+HH:MM" (with ":SS" when it has seconds).
func isoSeconds(t time.Time) string {
	_, offset := t.Zone()
	sign := byte('+')
	if offset < 0 {
		sign, offset = '-', -offset
	}
	text := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d%c%02d:%02d", t.Year(), int(t.Month()), t.Day(),
		t.Hour(), t.Minute(), t.Second(), sign, offset/3600, offset/60%60)
	if offset%60 != 0 {
		text += fmt.Sprintf(":%02d", offset%60)
	}
	return text
}
