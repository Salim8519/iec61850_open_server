package pytext

import (
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"studio-engine/internal/testutil"
	"testing"

	"studio-engine/internal/pyjson"
)

func TestPathLikePathlib(t *testing.T) {
	cases := map[string][]string{
		`C:\data\recovery\x.scadabench`: {`C:/data/`, "recovery", "x.scadabench"},
		`C:\data\recovery`:              {`C:\data\.\`, `recovery\`},
		`data\recovery`:                 {`data`, "recovery"},
		`..\data\recovery`:              {`..\data`, "recovery"},
		`\\server\share\d\recovery`:     {`\\server\share\d`, "recovery"},
		`C:x\recovery`:                  {`C:x`, "recovery"},
		`.`:                             {`.`},
	}
	for want, parts := range cases {
		if got := Path(parts[0], parts[1:]...); got != want {
			t.Errorf("Path(%q) = %q want %q", parts, got, want)
		}
	}
}

func TestReprLikePython(t *testing.T) {
	cases := map[string]any{
		`'it'`:                     "it",
		`"it's"`:                   "it's",
		`'a\'b"c'`:                 `a'b"c`,
		`'C:\\x\t\n\x01\x7f\xa0é'`: "C:\\x\t\n\x01\x7f\u00a0\u00e9",
		`'\u2028😀'`:                "\u2028\U0001F600",
		`None`:                     nil,
		`True`:                     true,
		`[1, 2.0, 'a', None]`:      []any{int64(1), 2.0, "a", nil},
		`{'k': [False], 'n': nan}`: pyjson.NewObject().Set("k", []any{false}).Set("n", math.NaN()),
		`123456789012345678901234567890`: func() *big.Int {
			n, _ := new(big.Int).SetString("123456789012345678901234567890", 10)
			return n
		}(),
		`-inf`:  math.Inf(-1),
		`1e-05`: 1e-05,
	}
	for want, value := range cases {
		if got := Repr(value); got != want {
			t.Errorf("Repr(%#v) = %s want %s", value, got, want)
		}
	}
	if Str("text") != "text" || Str(1.0) != "1.0" || Slice("é😀abc", 2) != "é😀" || Len("é😀") != 2 {
		t.Error("Str/Slice/Len")
	}
}

func TestOSErrorText(t *testing.T) {
	dir := testutil.TempDir(t)
	missing := filepath.Join(dir, "none", "x.scadabench")
	_, err := os.ReadFile(missing)
	want := "[Errno 2] No such file or directory: '" + strings.ReplaceAll(missing, `\`, `\\`) + "'"
	if got := OSError(err, true); got != want {
		t.Errorf("errno style %q want %q", got, want)
	}
	err = os.Mkdir(missing, 0o777)
	want = "[WinError 3] The system cannot find the path specified: '" + strings.ReplaceAll(missing, `\`, `\\`) + "'"
	if got := OSError(err, false); got != want {
		t.Errorf("WinError style %q want %q", got, want)
	}
}
