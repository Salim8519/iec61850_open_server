package archive

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

func TestCombineCRCMatchesWholeChecksum(t *testing.T) {
	random := rand.New(rand.NewSource(1))
	for round := 0; round < 200; round++ {
		data := make([]byte, random.Intn(5000))
		random.Read(data)
		cut := 0
		if len(data) > 0 {
			cut = random.Intn(len(data) + 1)
		}
		a, b := data[:cut], data[cut:]
		got := CombineCRC(crc32.ChecksumIEEE(a), crc32.ChecksumIEEE(b), int64(len(b)))
		if want := crc32.ChecksumIEEE(data); got != want {
			t.Fatalf("round %d: combine %08x want %08x", round, got, want)
		}
	}
}

func TestEncodeWritesPythonHeadersAndDumpsBytes(t *testing.T) {
	doc := testDoc(strings.Repeat("<SCL \"x\">\r\n\té</SCL>", 70000), "<SCL/>")
	encoder := NewEncoder()
	encoder.Now = func() time.Time { return time.Date(2026, 10, 20, 13, 45, 59, 0, time.Local) }
	payload, err := encoder.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	data, file := unzip(t, payload)
	want, _ := pyjson.Dumps(doc)
	if !bytes.Equal(data, want) {
		t.Fatalf("project.json differs from pyjson.Dumps (%d vs %d bytes)", len(data), len(want))
	}
	if file.Method != 8 || file.Flags != 0 || file.CreatorVersion != 20 || file.ReaderVersion != 20 ||
		file.ExternalAttrs != 0o600<<16 || len(file.Extra) != 0 {
		t.Fatalf("header fields %+v", file.FileHeader)
	}
	if binary.LittleEndian.Uint16(payload[4:]) != 20 || binary.LittleEndian.Uint16(payload[6:]) != 0 ||
		binary.LittleEndian.Uint16(payload[28:]) != 0 {
		t.Fatalf("local header % x", payload[:30])
	}
	date, clock := binary.LittleEndian.Uint16(payload[12:]), binary.LittleEndian.Uint16(payload[10:])
	if date != (2026-1980)<<9|10<<5|20 || clock != 13<<11|45<<5|29 {
		t.Fatalf("dos date %x time %x", date, clock)
	}
}

func TestCachedEncodeReusesUnchangedSegments(t *testing.T) {
	big := strings.Repeat("<IED name=\"A\"/>\n", 200000)
	doc := testDoc(big, "<SCL>second</SCL>")
	encoder := NewEncoder()
	first, err := encoder.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256Hex(big)
	cached := encoder.sources[digest].seg
	// Change only the diagram (a new immutable top-level value, as the project actor does).
	changed := pyjson.NewObject()
	for _, key := range doc.Keys() {
		value, _ := doc.Get(key)
		changed.Set(key, value)
	}
	changed.Set("diagram", pyjson.NewObject().Set("nodes", []any{}).Set("edges", []any{int64(1)}))
	second, err := encoder.Encode(changed)
	if err != nil {
		t.Fatal(err)
	}
	if encoder.sources[digest].seg != cached || cached.raw != nil {
		t.Fatal("SCL segment was rebuilt or kept its raw bytes")
	}
	for _, pair := range []struct {
		payload []byte
		doc     *pyjson.Object
	}{{first, doc}, {second, changed}} {
		data, _ := unzip(t, pair.payload)
		want, _ := pyjson.Dumps(pair.doc)
		if !bytes.Equal(data, want) {
			t.Fatal("cached encode differs from pyjson.Dumps")
		}
	}
	crc, size, err := encoder.Checksum(changed)
	_, file := unzip(t, second)
	if err != nil || crc != file.CRC32 || uint64(size) != file.UncompressedSize64 {
		t.Fatalf("Checksum %08x/%d, archive %08x/%d", crc, size, file.CRC32, file.UncompressedSize64)
	}
}

func TestEncodeGenericShapes(t *testing.T) {
	for _, doc := range []*pyjson.Object{
		pyjson.NewObject(),
		pyjson.NewObject().Set("scl_sources", pyjson.NewObject()),
		pyjson.NewObject().Set("scl_sources", pyjson.NewObject().Set("a", int64(1))),
		pyjson.NewObject().Set("scl_sources", []any{"x"}).Set("after", nil),
		pyjson.NewObject().Set("a", "x").Set("scl_sources", pyjson.NewObject().Set("d", "t")).Set("z", true),
	} {
		payload, err := Encode(doc)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := unzip(t, payload)
		if want, _ := pyjson.Dumps(doc); !bytes.Equal(data, want) {
			t.Errorf("got %s want %s", data, want)
		}
	}
}

func TestEncodeRefusesLikePython(t *testing.T) {
	cases := map[string]float64{"nan": math.NaN(), "inf": math.Inf(1), "-inf": math.Inf(-1)}
	for repr, value := range cases {
		doc := pyjson.NewObject().Set("a", []any{1.5, pyjson.NewObject().Set("b", value)})
		_, err := Encode(doc)
		var appErr *apperr.Error
		if !asAppErr(err, &appErr) || appErr.Kind != apperr.Value ||
			appErr.Message != "Out of range float values are not JSON compliant: "+repr {
			t.Errorf("%s: %v", repr, err)
		}
	}
	if testing.Short() {
		return
	}
	huge := pyjson.NewObject().Set("name", strings.Repeat("x", MaxUnpacked))
	_, err := Encode(huge)
	var appErr *apperr.Error
	if !asAppErr(err, &appErr) || appErr.Kind != apperr.Studio ||
		appErr.Message != "Engineering project exceeds the 100 MB save limit; nothing was written" {
		t.Fatalf("size limit: %v", err)
	}
}

func asAppErr(err error, target **apperr.Error) bool {
	value, ok := err.(*apperr.Error)
	*target = value
	return ok
}
