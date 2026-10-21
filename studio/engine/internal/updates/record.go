package updates

import (
	"errors"
	"strings"
	"unicode/utf8"

	"studio-engine/internal/licensing"
	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// MaxNotes is versions.MAX_NOTES (characters).
const MaxNotes = 600

// errBadRecord is any ValueError/TypeError of clean_record; check stores "bad_answer".
var errBadRecord = errors.New("release record refused")

// download is versions._download: {"url", "sha256"} of a checked scadabench.com link.
func download(value any, found, required bool) (any, error) {
	if (!found || value == nil) && !required {
		return nil, nil
	}
	object, isObject := value.(*pyjson.Object)
	if !isObject {
		return nil, errBadRecord
	}
	url, _ := object.Get("url")
	digest, _ := object.Get("sha256")
	digestText, isText := digest.(string)
	if !SafeURL(url) || !isText || !sha256Pattern.MatchString(digestText) {
		return nil, errBadRecord
	}
	return pyjson.NewObject().Set("url", url).Set("sha256", digestText), nil
}

func readable(version any) bool {
	_, ok, err := Parse(version)
	return err == nil && ok
}

// CleanRecord is versions.clean_record: validate the signed release record again on this
// PC and keep only the known fields, in Python's order.
func CleanRecord(payload any) (*pyjson.Object, error) {
	record, isObject := payload.(*pyjson.Object)
	if !isObject {
		return nil, errBadRecord
	}
	version, _ := record.Get("version")
	minimum, _ := record.Get("min_version")
	if !licensing.Truthy(minimum) {
		minimum = ""
	}
	notes, found := record.Get("notes")
	if !found {
		notes = ""
	}
	versionText, _ := version.(string)
	if !readable(version) || strings.Contains(versionText, "+") || (licensing.Truthy(minimum) && !readable(minimum)) {
		return nil, errBadRecord
	}
	date, _ := record.Get("date")
	if dateText, isText := date.(string); !isText || !datePattern.MatchString(dateText) {
		return nil, errBadRecord
	}
	page, _ := record.Get("page")
	notesText, isText := notes.(string)
	if !isText || utf8.RuneCountInString(notesText) > MaxNotes || !SafeURL(page) {
		return nil, errBadRecord
	}
	installerValue, installerFound := record.Get("installer")
	installer, err := download(installerValue, installerFound, true)
	if err != nil {
		return nil, err
	}
	portableValue, portableFound := record.Get("portable")
	portable, err := download(portableValue, portableFound, false)
	if err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("version", version).Set("date", date).Set("min_version", minimum).
		Set("notes", pytext.Strip(notesText)).Set("page", page).Set("installer", installer).
		Set("portable", portable), nil
}
