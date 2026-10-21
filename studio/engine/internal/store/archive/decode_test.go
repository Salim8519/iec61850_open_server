package archive

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

func decodeMessage(payload []byte) (apperr.Kind, string) {
	_, err := Decode(payload)
	if err == nil {
		return -1, ""
	}
	if appErr, ok := err.(*apperr.Error); ok {
		return appErr.Kind, appErr.Message
	}
	return -2, err.Error()
}

func TestDecodeRoundTrip(t *testing.T) {
	doc := testDoc("<SCL>é</SCL>")
	payload, err := Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(payload)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := pyjson.Dumps(decoded)
	want, _ := pyjson.Dumps(doc)
	if !bytes.Equal(again, want) {
		t.Fatalf("round trip\n%s\n%s", again, want)
	}
}

func TestDecodeErrorsMatchArchivePy(t *testing.T) {
	json := func(text string) []byte { return zipOf(t, map[string][]byte{EntryName: []byte(text)}, EntryName) }
	badCRC := json(`{"format": 15}`)
	badCRC[14] ^= 0xff // local header CRC; archive/zip checks the central directory CRC
	central := bytes.LastIndex(badCRC, []byte{0x50, 0x4b, 0x01, 0x02})
	badCRC[central+16] ^= 0xff
	twice := zipOf(t, map[string][]byte{EntryName: []byte("{}"), "x": []byte("{}")}, EntryName, "x")
	cases := []struct {
		payload []byte
		kind    apperr.Kind
		message string
	}{
		{[]byte("not a zip"), apperr.Studio, "Invalid project archive"},
		{json(`{"format": 15`), apperr.Studio, "Invalid project archive"},
		{json(`[1, 2]`), apperr.Studio, "Unsupported project format"},
		{json(`"text"`), apperr.Studio, "Unsupported project format"},
		{json("\xef\xbb\xbf\xef\xbb\xbf{}"), apperr.Studio, "Invalid project archive"},
		{badCRC, apperr.Studio, "Invalid project archive"},
		{twice, apperr.Studio, "Expected a single project.json entry"},
		{zipOf(t, map[string][]byte{"other.json": []byte("{}")}, "other.json"), apperr.Studio,
			"Expected a single project.json entry"},
		{json("{\"a\": \"\xff\"}"), apperr.Value,
			"'utf-8' codec can't decode byte 0xff in position 7: invalid start byte"},
		{json("{\"a\": \"\xe2\x82\"}"), apperr.Value,
			"'utf-8' codec can't decode bytes in position 7-8: invalid continuation byte"},
		{json("{\"a\": \"\xe2"), apperr.Value,
			"'utf-8' codec can't decode byte 0xe2 in position 7: unexpected end of data"},
		{json("\xef\xbb\xbf{\"a\": \"\xc0\"}"), apperr.Value,
			"'utf-8' codec can't decode byte 0xc0 in position 7: invalid start byte"},
	}
	for index, c := range cases {
		kind, message := decodeMessage(c.payload)
		if kind != c.kind || message != c.message {
			t.Errorf("case %d: got %v %q want %v %q", index, kind, message, c.kind, c.message)
		}
	}
}

func TestDecodeAcceptsPythonTextEncodings(t *testing.T) {
	utf16 := []byte{0xff, 0xfe}
	for _, r := range `{"a": "` + "é\U0001F600" + `"}` {
		if r > 0xffff {
			r -= 0x10000
			utf16 = binary.LittleEndian.AppendUint16(utf16, uint16(0xd800+(r>>10)))
			r = 0xdc00 + r&0x3ff
		}
		utf16 = binary.LittleEndian.AppendUint16(utf16, uint16(r))
	}
	for _, text := range [][]byte{[]byte("\xef\xbb\xbf{\"a\": \"\xc3\xa9\xf0\x9f\x98\x80\"}"), utf16} {
		payload := zipOf(t, map[string][]byte{EntryName: text}, EntryName)
		doc, err := Decode(payload)
		if err != nil {
			t.Fatal(err)
		}
		if value, _ := doc.Get("a"); value != "é\U0001F600" {
			t.Errorf("value %q", value)
		}
	}
	if _, err := Decode(zipOf(t, map[string][]byte{EntryName: []byte("{\"a\": \"\xed\xa0\x80\"}")}, EntryName)); err != nil {
		t.Errorf("surrogatepass refused: %v", err)
	}
}

func TestDecodeSizeLimits(t *testing.T) {
	if kind, message := decodeMessage(make([]byte, MaxUnpacked+1)); kind != apperr.Studio ||
		message != "Project archive exceeds 100 MB" {
		t.Errorf("payload limit: %q", message)
	}
	payload := zipOf(t, map[string][]byte{EntryName: []byte("{}")}, EntryName)
	central := bytes.LastIndex(payload, []byte{0x50, 0x4b, 0x01, 0x02})
	binary.LittleEndian.PutUint32(payload[central+24:], MaxLegacyUnpacked+1)
	if _, message := decodeMessage(payload); message != "Unpacked project exceeds the 256 MB legacy recovery limit" {
		t.Errorf("legacy limit: %q", message)
	}
	if testing.Short() {
		return
	}
	for format, want := range map[string]string{"15": "Compact project exceeds 100 MB", "2.0": "Compact project exceeds 100 MB", "1": ""} {
		text := `{"format": ` + format + `, "pad": "` + strings.Repeat("x", MaxUnpacked) + `"}`
		_, message := decodeMessage(zipOf(t, map[string][]byte{EntryName: []byte(text)}, EntryName))
		if message != want {
			t.Errorf("format %s: %q", format, message)
		}
	}
}

func TestIsCompactFormat(t *testing.T) {
	yes := []any{int64(2), int64(15), 2.0, 15.0, 2}
	no := []any{int64(1), int64(16), 1.0, 2.5, true, false, "15", nil}
	for _, value := range yes {
		if !isCompactFormat(value) {
			t.Errorf("%v should be compact", value)
		}
	}
	for _, value := range no {
		if isCompactFormat(value) {
			t.Errorf("%v should not be compact", value)
		}
	}
}
