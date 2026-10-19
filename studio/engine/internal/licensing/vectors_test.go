package licensing

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"studio-engine/internal/pyjson"
)

// outcome is vector_common.outcome: a JSON result or a Python exception description.
type outcome struct {
	OK      json.RawMessage `json:"ok"`
	Error   string          `json:"error"`
	Message string          `json:"message"`
	Code    any             `json:"code"`
}

type unitVectors struct {
	ReleaseKeys map[string]string            `json:"release_keys"`
	TestKey     struct{ Kid, Public string } `json:"test_key"`
	DefaultURL  string                       `json:"default_server_url"`
	Base64      []struct {
		Text  string  `json:"text"`
		Strip *string `json:"strip"`
		Pad   *string `json:"pad"`
	} `json:"base64"`
	Ed25519 []struct {
		Public, Message, Signature string
		OK                         bool `json:"ok"`
	} `json:"ed25519"`
	Certificates []struct {
		outcome
		Text string `json:"text"`
		Keys string `json:"keys"`
	} `json:"certificates"`
	Policy []struct {
		outcome
		Payload *string `json:"payload"`
		Now     int64   `json:"now"`
		MaxSeen int64   `json:"max_seen"`
		Dev     bool    `json:"dev"`
	} `json:"policy"`
	NormalizeKey []struct {
		outcome
		Input string `json:"input"`
	} `json:"normalize_key"`
	MaskKey []struct{ Input, Output string } `json:"mask_key"`
	Compose []struct {
		GUID   string `json:"guid"`
		Serial uint32 `json:"serial"`
		UUID   string `json:"uuid"`
		V2, V1 string
	} `json:"compose"`
	BoundHWID []struct {
		outcome
		Cert, Issued, Current string
		Legacy                *string
	} `json:"bound_hwid"`
	SMBIOS []struct{ Raw, UUID string } `json:"smbios"`
	ISO    []struct {
		Text  string `json:"text"`
		Error bool   `json:"error"`
		Naive []int  `json:"naive"`
		TS    *int64 `json:"ts"`
	} `json:"iso"`
	Anchor []struct {
		Runs       string             `json:"runs"`
		Anchor     *string            `json:"anchor"`
		MaxSeen    json.Number        `json:"max_seen"`
		AnchorOnly json.Number        `json:"anchor_only"`
		Advance    map[string]*string `json:"advance"`
	} `json:"anchor"`
	Solve []struct {
		outcome
		Nonce      string `json:"nonce"`
		Difficulty int64  `json:"difficulty"`
	} `json:"solve"`
}

var (
	unitOnce sync.Once
	units    unitVectors
)

func readVectors(t *testing.T, name string, target any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "generated", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func loadUnits(t *testing.T) *unitVectors {
	unitOnce.Do(func() { readVectors(t, "vectors.json", &units) })
	return &units
}

func mustDecode(t *testing.T, text string) any {
	t.Helper()
	value, err := pyjson.Decode([]byte(text))
	if err != nil {
		t.Fatalf("decode %q: %v", text, err)
	}
	return value
}

func compactText(t *testing.T, value any) string {
	t.Helper()
	data, err := pyjson.Compact(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func fromB64(t *testing.T, text string) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func fromHex(t *testing.T, text string) []byte {
	t.Helper()
	data, err := hex.DecodeString(text)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// testKeys is license_signer.keys(): the test public key under its kid.
func testKeys(t *testing.T) map[string][]byte {
	v := loadUnits(t)
	return map[string][]byte{v.TestKey.Kid: fromHex(t, v.TestKey.Public)}
}

// okString decodes an outcome's "ok" JSON as a string, or reports false.
func (o outcome) okString() (string, bool) {
	var text string
	if o.OK == nil || json.Unmarshal(o.OK, &text) != nil {
		return "", false
	}
	return text, true
}

func b64Text(data []byte) string { return base64.StdEncoding.EncodeToString(data) }
