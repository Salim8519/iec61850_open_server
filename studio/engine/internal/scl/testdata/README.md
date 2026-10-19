# SCL corpus and Python goldens (workstream B)

Byte-exact inputs and outputs of the frozen Python SCL layer (`studio/backend/studio_app/scl`),
used by the Go `internal/scl` tests. Only public inputs: `scd/*.scd`, the demo station,
`make_template`, `tests/unit/goose_samples.py`, `tests/unit/sld_fixtures.py` and generated SCL.
`.gitattributes` marks everything here `-text`: `core.autocrlf` must never rewrite these bytes.

## Regenerate

```powershell
$py = "studio/.venv/Scripts/python.exe"
& $py studio/engine/internal/scl/testdata/corpus_gen.py        # corpus/*.scd + corpus/manifest.json
& $py studio/engine/internal/scl/testdata/golden_gen.py        # golden/<case>/..., manifest, xsd_seed
& $py studio/engine/internal/scl/testdata/xml_errors_gen.py    # golden/xml_errors.json
& $py studio/engine/internal/scl/testdata/unicode_errors_gen.py # golden/utf8_errors.json
```

Run in this order (golden_gen reads the corpus manifest). Add `--verbose` to corpus_gen to print
each case's expectation and rejection message. All generators are deterministic; running them
twice must leave `git status` clean.

## Files

- `corpus/<case>.scd`: raw input bytes (BOMs, UTF-16, Latin-1, CR/CRLF kept as written).
- `corpus/manifest.json`: `[{case, purpose, base, expect, sha256}]` sorted by case. `expect` is
  measured with `import_scl` over all IEDs: `reject` (raises), `warn` (any IED warning), `accept`.
- `golden/manifest.json`: per case `{case, input, sha256, ied_names, files, aliases}`. `input` is
  the repository-relative path of the input bytes; `aliases` maps a file that was not written
  because it would be byte-identical to another (`import-each-0.json` -> `import-all.json`).
- `golden/<case>/input.sha256`: hex SHA-256 of the input bytes.
- `preview.json`: `preview_ieds(bytes)`; on an exception `error.json` instead.
- `validate.json`: `validate_schema(bytes)` result, or an error object.
- `import-all.json`: `import_scl(bytes, "case.scd")`; `import-each-<n>.json`: the same with
  `[name]` for the first three IED names (document order, from `ied_names`).
  Each IED's `id` is `ied-<index>` and `source` is `sha256:` + SHA-256 of `source` in UTF-8.
  `cfg-<index>.cfg` / `cfg-each-<n>-<index>.cfg` hold each `cfg` text exactly (LF).
- `import-select-errors.json`: `selected_names` = `[]`, `["nope"]`, `[name, name]`, `[1]`, each with
  `result` (normalised list) or `error`.
- `export-<n>.overrides.json` + `export-<n>.scd` (or `export-<n>.error.json`):
  `export_ied(bytes.decode("utf-8-sig"), name, overrides)` for IED `<n>`. Overrides pick 1-3
  writable scalar signals (BOOLEAN flipped, INT +1 within range, FLOAT 12.5, VisString `"x"`);
  the families used rotate with the case name.
- `export-errors.json`: `[{ied, overrides, error | payload}]` for refused overrides (unknown
  reference, non-dict, wrong type, out of range, Enum ord without label, Dbpos, ...). The
  Timestamp probe succeeds; its payload is `export-errors-<k>.scd`.
- `golden/xsd_seed.json`: `[{sha256, schema, status, errors: [{line, message}]}]` for every
  `validate_schema` call that reached lxml (direct, inside import and inside export), sorted by
  `(sha256, schema)`; seeds the Go XSD result cache.
- `golden/xml_errors.json`: `[{input: base64, result}]`, `result` is `"OK"` or `str(StudioError)`
  of `parse_document(bytes)`: about 400 hand-written inputs (`xml_error_snippets.py`) plus 3000
  single-step mutations (`random.Random(1234)`) of a small valid SCL document.
- `golden/utf8_errors.json`: `[{input: base64, result}]`, `"OK"` or `str(UnicodeDecodeError)` of
  `bytes.decode("utf-8-sig")`. Positions are counted after a stripped BOM.

Error objects are `{"type", "message", "code", "details"}` (`code`/`details` are `null` for
non-`StudioError` exceptions). Every JSON golden is `json.dumps(value)` with Python defaults,
ASCII, no trailing newline. Not covered here: the 20 MB `MAX_XML` limit (test it directly).
