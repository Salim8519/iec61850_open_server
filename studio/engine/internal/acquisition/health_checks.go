package acquisition

import (
	"time"

	"studio-engine/internal/acquisition/modes"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// readItems is health_checks._read: no request for an empty list.
func readItems(process Worker, items []any) (*pyjson.Object, *pyjson.Object, error) {
	if len(items) == 0 {
		return pyjson.NewObject(), pyjson.NewObject(), nil
	}
	answer, err := process.Request("read", 30*time.Second, pyv.O("refs", items))
	if err != nil {
		return nil, nil, err
	}
	values, errors := modes.ParseAnswer(pyv.Obj(answer))
	return values, errors, nil
}

// readMode reads Mod/Beh/Sim of the IED: {values, errors, checked_at, source}.
func readMode(process Worker, ied *pyjson.Object, now func() time.Time) (*pyjson.Object, error) {
	values, errors, err := readItems(process, modes.ModeItems(modeView(ied)))
	if err != nil {
		return nil, err
	}
	return pyv.O("values", values, "errors", errors, "checked_at", pyv.ISOSeconds(now()), "source", "mms_read"), nil
}

// readHealth reads GOOSE-block settings and the mode; report blocks come from the
// rcb_status answers already read (none with polling access).
func readHealth(process Worker, ied, blocks *pyjson.Object, access string,
	now func() time.Time) (*pyjson.Object, error) {
	stamp := pyv.ISOSeconds(now())
	gocbValues, gocbErrors, err := readItems(process, modes.GocbItems(ied))
	if err != nil {
		return nil, err
	}
	mode, err := readMode(process, ied, now)
	if err != nil {
		return nil, err
	}
	var rcbs any
	if access == "auto_reports" {
		rcbs = pyv.O("blocks", blocks, "checked_at", stamp, "source", "mms_read")
	}
	return pyv.O("rcbs", rcbs,
		"gocbs", pyv.O("values", gocbValues, "errors", gocbErrors, "checked_at", stamp, "source", "mms_read"),
		"mode", mode), nil
}

// modeView is the IED with only its Mod/Beh/Sim points, cached per model object, so a
// 90,000-point model is not rescanned on every mode read (change 0110).
func modeView(ied *pyjson.Object) *pyjson.Object {
	return modeCache.view(ied)
}
